package store_test

import (
	"strings"
	"testing"

	"github.com/bank-data-db/server/data"
	"github.com/bank-data-db/server/db/store"
	"github.com/bank-data-db/server/tutils"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseTransID(t *testing.T) string {
	// Preparing for parallel tests....
	return t.Name() + "-trans-"
}

func TestTransactionsUnmapForMappingID(t *testing.T) {
	catID := newTestCat(t)

	m1 := data.Mapping{
		ResName:       new("Result Nmae"),
		ResCategoryID: new(catID),
	}
	m2 := m1

	newTestMap(t, &m1)
	newTestMap(t, &m2)

	setup := func(t *testing.T) store.Store {
		s := newStore(t)

		bID := baseTransID(t)

		trans := []*store.TransactionsInsertParams{}

		// n = name only
		// c = category only
		// nc = name + category
		types := []string{"n", "c", "nc"}
		b := &pgx.Batch{}

		for _, idType := range types {
			id := bID + idType

			trans = append(trans, &store.TransactionsInsertParams{
				ID:               id,
				ResolvedName:     new("Trans Name!"),
				ResolvedCategory: new(catID),
			})

			if len(idType) == 2 {
				s.BatchInsertTransMapping(b, id, m1.ID, true)
				s.BatchInsertTransMapping(b, id, m1.ID, false)
			} else {
				// map the appropriate col using the main mapping id, and
				// the other col using the secondary
				s.BatchInsertTransMapping(b, id, m1.ID, idType == "n")
				s.BatchInsertTransMapping(b, id, m2.ID, idType != "n")
			}
		}

		newTestTrans(t, trans)

		err := s.SendBatch(t.Context(), b)
		require.NoError(t, err)

		return s
	}

	// de-based id -> [name, catID]
	getTrans := func(t *testing.T) map[string][2]*string {
		db := tutils.DB(t)
		rows, err := db.Query(t.Context(), `SELECT id, resolved_name, resolved_category FROM transactions WHERE id LIKE $1`, baseTransID(t)+"%")
		require.NoError(t, err)

		m := map[string][2]*string{}

		for rows.Next() {
			var id string
			var n, c *string
			err := rows.Scan(&id, &n, &c)
			require.NoError(t, err)

			m[strings.TrimPrefix(id, baseTransID(t))] = [2]*string{n, c}
		}

		require.Len(t, m, 3, "Sanity check for trans count")

		return m
	}

	// Asserts that only 1 column was nulled out. if name is true, it indicates only name should be nulled
	assertSingleNull := func(t *testing.T, name bool, vals [2]*string, testCtx string) {
		eNull, eNotNull := vals[1], vals[0]
		col := "category"

		if name {
			eNull, eNotNull = vals[0], vals[1]
			col = "name"
		}

		assert.Nil(t, eNull, "expected column resolved_%s to be nulled out in %s, but it was not", col, testCtx)
		assert.NotNil(t, eNotNull, "wrong column was nulled in %s! Expected resolved_%s!", testCtx, col)
	}

	t.Run("only", func(t *testing.T) {
		runTest := func(name bool) func(t *testing.T) {
			return func(t *testing.T) {
				// ensure that only the needed col is affected,
				// AND only on the transactions that are mapped by this map
				// So this means, for categories, c & nc. n has a mapped category, but by a different mapping

				s := setup(t)
				err := s.TransactionsUnmapForMappingID(t.Context(), m1.ID, name, !name)
				require.NoError(t, err)

				trans := getTrans(t)

				// TODO: is this too "smart"?
				// I should prob refactor this to be dumber. Its very hard to read lmao
				// However, it is currently 2:45am and so this looks fucking fantastic to me
				// (fantastic in the worst sense of the word tho)

				i := 1
				single, notSingle := "c", "n"
				if name {
					i = 0
					single, notSingle = "n", "c"
				}

				assertSingleNull(t, name, trans["nc"], "a double mapped transaction")
				assertSingleNull(t, name, trans[single], "a single mapped transaction")
				assert.NotNil(t, trans[notSingle][i], "unmapping isn't bound the mapping id!")
			}
		}

		t.Run("category", runTest(false))
		t.Run("name", runTest(true))
	})

	t.Run("both", func(t *testing.T) {
		s := setup(t)
		err := s.TransactionsUnmapForMappingID(t.Context(), m1.ID, true, true)
		require.NoError(t, err)

		trans := getTrans(t)

		assert.Equal(t, [2]*string{nil, nil}, trans["nc"], "in a double mapped transaction didn't get nulled")

		assertSingleNull(t, true, trans["n"], "a name transaction")
		assertSingleNull(t, false, trans["c"], "a category transaction")
	})
}
