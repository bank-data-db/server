package store_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/bank-data-db/proto/mappings_pb"
	"github.com/bank-data-db/server/data"
	"github.com/bank-data-db/server/db/store"
	"github.com/bank-data-db/server/tutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionsMapsMapExisting(t *testing.T) {
	catID := newTestCat(t)

	t.Run("target", func(t *testing.T) {
		name := "Resolved Name!!"

		m := &data.Mapping{
			Name:          "",
			InpCardID:     new(CARD_ID),
			ResName:       new(name),
			ResCategoryID: new(catID),
		}
		newTestMap(t, m)

		transID := "123"

		runTest := func(name bool) func(t *testing.T) {
			return func(t *testing.T) {
				newTestTrans(t, []*store.TransactionsInsertParams{
					{
						ID:        transID,
						AuthorID:  USER_ID,
						CardID:    CARD_ID,
						AuthedAt:  time.Now(),
						SettledAt: time.Now(),
						Amount:    decimal.New(1, 1),
					},
				})

				s := newStore(t)
				db := tutils.DB(t)

				updated, err := s.TransactionsMapsMapExisting(t.Context(), name, USER_ID, m)
				require.NoError(t, err)
				assert.Equal(t, 1, updated)

				var resolvedName, resolvedCat *string
				err = db.QueryRow(t.Context(), `SELECT resolved_name, resolved_category FROM transactions WHERE id = $1`, transID).Scan(
					&resolvedName, &resolvedCat,
				)
				require.NoError(t, err)

				c := 0
				err = db.QueryRow(t.Context(), `SELECT COUNT(*) FROM mapped_transactions WHERE trans_id = $1`, transID).Scan(&c)
				require.NoError(t, err)
				require.Equal(t, 1, c, "Inserted > 1 mapped_transaction rows")

				updatedName := false
				err = db.QueryRow(t.Context(), `SELECT updated_name FROM mapped_transactions WHERE trans_id = $1`, transID).Scan(&updatedName)
				require.NoError(t, err)
				assert.Equal(t, name, updatedName, "mapped_transactions has the wrong updated_name value :(")

				var target, other, expected = resolvedCat, resolvedName, m.ResCategoryID

				if name {
					target, other, expected = resolvedName, resolvedCat, m.ResName
				}

				assert.Nil(t, other, "Updated the wrong column!")
				if assert.NotNil(t, target, "Didn't set the target column") {
					assert.EqualValues(t, *expected, *target, "Set the target column incorrectly :(")
				}
			}
		}

		t.Run("name", runTest(true))
		t.Run("category", runTest(false))
	})

	t.Run("matching", func(t *testing.T) {
		// tests the mapping against existing transactions
		// The transactions MUST have a id="e", which is expected to match
		// AND an id="n", which si expected to NOT match
		easyMapTest := func(t *testing.T, m *data.Mapping) {
			m.ResName = new("ResolvedName")
			newTestMap(t, m)

			s := newStore(t)

			c, err := s.TransactionsMapsMapExisting(t.Context(), true, USER_ID, m)
			require.NoError(t, err)

			assert.Equal(t, 1, c, "Didn't match the right amount of transaction")
			db := tutils.DB(t)

			rows, err := db.Query(t.Context(), `SELECT id, resolved_name FROM transactions WHERE id IN ('e', 'n')`)
			require.NoError(t, err)

			for rows.Next() {
				var id string
				var name *string

				err := rows.Scan(&id, &name)
				require.NoError(t, err)

				if id == "e" {
					assert.NotNil(t, name, "Didn't match the expected")
				} else {
					assert.Nil(t, name, "Matched the un-expected!")
				}
			}
		}

		t.Run("card_id", func(t *testing.T) {
			cardID2 := newTestCard(t)

			newTestTrans(t, []*store.TransactionsInsertParams{
				{
					ID:     "e",
					CardID: CARD_ID,
				},
				{
					ID:     "n",
					CardID: cardID2,
				},
			})

			easyMapTest(t, &data.Mapping{
				InpCardID: new(CARD_ID),
			})
		})
		t.Run("text", func(t *testing.T) {
			newTestTrans(t, []*store.TransactionsInsertParams{
				{
					ID:          "e",
					Description: "ABCDE 123",
				},
				{
					ID:          "n",
					Description: "EVIL MCEVIL",
				},
			})

			easyMapTest(t, &data.Mapping{
				InpText: regexp.MustCompilePOSIX(`^A.+`),
			})
		})

		t.Run("amount", func(t *testing.T) {
			t.Run("lt", func(t *testing.T) {
				newTestTrans(t, []*store.TransactionsInsertParams{
					{
						ID:     "e",
						Amount: decimal.NewFromInt(10),
					},
					{
						ID:     "n",
						Amount: decimal.NewFromInt(11),
					},
				})

				easyMapTest(t, &data.Mapping{
					InpAmt:        new(11.0),
					InpAmtMatcher: new(mappings_pb.AmountMatchModeLt),
				})
			})
			t.Run("gt", func(t *testing.T) {
				newTestTrans(t, []*store.TransactionsInsertParams{
					{
						ID:     "e",
						Amount: decimal.NewFromInt(11),
					},
					{
						ID:     "n",
						Amount: decimal.NewFromInt(10),
					},
				})

				easyMapTest(t, &data.Mapping{
					InpAmt:        new(10.5),
					InpAmtMatcher: new(mappings_pb.AmountMatchModeGt),
				})
			})
			t.Run("exact", func(t *testing.T) {
				newTestTrans(t, []*store.TransactionsInsertParams{
					{
						ID:     "e",
						Amount: decimal.NewFromInt(10),
					},
					{
						ID:     "n",
						Amount: decimal.NewFromInt(11),
					},
				})

				easyMapTest(t, &data.Mapping{
					InpAmt:        new(10.0),
					InpAmtMatcher: new(mappings_pb.AmountMatchModeExact),
				})
			})
		})

		t.Run("multi", func(t *testing.T) {
			// Both will match numerically, but only e will match via text

			newTestTrans(t, []*store.TransactionsInsertParams{
				{
					ID:          "e",
					Amount:      decimal.NewFromInt(10),
					Description: "ABC 123",
				},
				{
					ID:          "n",
					Amount:      decimal.NewFromInt(10),
					Description: "NOT ABC 123",
				},
			})

			easyMapTest(t, &data.Mapping{
				InpAmt:        new(10.0),
				InpAmtMatcher: new(mappings_pb.AmountMatchModeExact),
				InpText:       regexp.MustCompilePOSIX(`^ABC.+`),
			})
		})

		t.Run("already_matched", func(t *testing.T) {
			assertMapped := func(t *testing.T, transID string, targetMap *data.Mapping) {
				c := tutils.DB(t)

				var resolvedName *string

				err := c.QueryRow(t.Context(), `SELECT resolved_name FROM transactions WHERE id = $1`, transID).Scan(&resolvedName)
				require.NoError(t, err)

				rows, err := c.Query(t.Context(), `SELECT mapping_id FROM mapped_transactions WHERE trans_id = $1`, transID)
				require.NoError(t, err)

				if assert.NotNil(t, resolvedName) {
					assert.Equal(t, *targetMap.ResName, *resolvedName)
				}

				count := 0
				for rows.Next() {
					count++

					var mapID string
					err := rows.Scan(&mapID)
					require.NoError(t, err)

					assert.Equal(t, targetMap.ID, mapID, "Wrong mapping ID")
				}

				assert.Equal(t, 1, count, "Too many mapped transactions!")
			}

			t.Run("with_mapping", func(t *testing.T) {
				mapHigh := &data.Mapping{
					InpCardID: new(CARD_ID),
					ResName:   new("Name!"),
					Priority:  99,
				}
				mapLow := &data.Mapping{
					InpCardID: new(CARD_ID),
					ResName:   new("Name 2!"),
					Priority:  0,
				}
				newTestMap(t, mapHigh)
				newTestMap(t, mapLow)

				t.Run("old_higher_priority", func(t *testing.T) {
					newTestTrans(t, []*store.TransactionsInsertParams{{ID: "e"}})

					_, err := newStore(t).TransactionsMapsMapExisting(t.Context(), true, USER_ID, mapHigh)
					require.NoError(t, err)

					c, err := newStore(t).TransactionsMapsMapExisting(t.Context(), true, USER_ID, mapLow)
					require.NoError(t, err)

					assert.Equal(t, 0, c, "Overwritten a high priority mapping :(")

					assertMapped(t, "e", mapHigh)
				})

				t.Run("old_lower_priority", func(t *testing.T) {
					newTestTrans(t, []*store.TransactionsInsertParams{{ID: "e"}})

					_, err := newStore(t).TransactionsMapsMapExisting(t.Context(), true, USER_ID, mapLow)
					require.NoError(t, err)

					c, err := newStore(t).TransactionsMapsMapExisting(t.Context(), true, USER_ID, mapHigh)
					require.NoError(t, err)

					assert.Equal(t, 1, c, "Didn't overwrite a lower priority :(")

					assertMapped(t, "e", mapHigh)
				})

				t.Run("old_eq_priority", func(t *testing.T) {
					mapEq := &data.Mapping{
						InpCardID: new(CARD_ID),
						ResName:   new("Name 3!"),
						Priority:  0,
					}

					newTestMap(t, mapEq)
					newTestTrans(t, []*store.TransactionsInsertParams{{ID: "e"}})

					_, err := newStore(t).TransactionsMapsMapExisting(t.Context(), true, USER_ID, mapLow)
					require.NoError(t, err)

					c, err := newStore(t).TransactionsMapsMapExisting(t.Context(), true, USER_ID, mapEq)
					require.NoError(t, err)

					assert.Equal(t, 0, c, "Overwritten an equal mapping??? NOOO")

					assertMapped(t, "e", mapLow)
				})
			})

			t.Run("with_manual", func(t *testing.T) {
				m := &data.Mapping{
					InpCardID: new(CARD_ID),
					ResName:   new("Res Name (evil)"),
				}
				newTestMap(t, &data.Mapping{})
				newTestTrans(t, []*store.TransactionsInsertParams{{ID: "e", ResolvedName: new("Manual Name")}})

				count, err := newStore(t).TransactionsMapsMapExisting(t.Context(), true, USER_ID, m)
				require.NoError(t, err)

				assert.Equal(t, 0, count, "Overwritten a manual transaction :(")

				var resolvedName *string

				c := tutils.DB(t)
				err = c.QueryRow(t.Context(), `SELECT resolved_name FROM transactions WHERE id = $1`, "e").Scan(&resolvedName)
				require.NoError(t, err)

				mapped := 0
				err = c.QueryRow(t.Context(), `SELECT COUNT(*) FROM mapped_transactions WHERE trans_id = $1`, "e").Scan(&mapped)
				require.NoError(t, err)

				if assert.NotNil(t, resolvedName) {
					assert.Equal(t, "Manual Name", *resolvedName)
				}

				assert.Equal(t, 0, mapped, "Inserted a mapped_transaction")
			})
		})
	})
}
