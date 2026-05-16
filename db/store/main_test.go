package store_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/bank-data-db/server/config"
	"github.com/bank-data-db/server/data"
	"github.com/bank-data-db/server/db"
	"github.com/bank-data-db/server/db/store"
	"github.com/bank-data-db/server/snownode"
	"github.com/bank-data-db/server/tutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func init() {
	config.LoadForTests()
	tutils.InsertTestUserDB(USER_ID)
	_, err := db.GetDB(slog.New(slog.DiscardHandler)).Exec(
		context.Background(),
		`INSERT INTO cards (id, name, user_id) VALUES ($1, $2, $3)`,
		CARD_ID, "Card Name!", USER_ID,
	)
	if err != nil {
		if !db.UniqueConstraint(err) {
			panic("Error setting up db card: " + err.Error())
		}
	}
}

func newStore(t *testing.T) store.Store {
	return store.NewStore(tutils.DB(t))
}

const (
	USER_ID string = "123"
	CARD_ID string = "123"
)

func cleanupRow(t *testing.T, table string, id string) {
	t.Cleanup(func() {
		// Use the default logger so that if smt goes wrong, we get the full picture
		db := db.GetDB(slog.Default())
		// Ignore the err since its alr logged by the slog
		// We can't use t.Log bc the test would already be complete
		db.Exec(context.Background(), `DELETE FROM `+table+` WHERE id = $1`, id) //nolint:errcheck
	})
}

// creates a category & cleans it up
func newTestCat(t *testing.T) string {
	catID, err := newStore(t).CategoriesNew(t.Context(), USER_ID, "catName-"+snownode.NewID(), "1", "ffffff")
	require.NoError(t, err)
	cleanupRow(t, `categories`, catID)

	return catID
}

// Creates a card & cleans it up. However, you probably want to use CARD_ID instead
func newTestCard(t *testing.T) string {
	cardID, err := newStore(t).CardsNew(t.Context(), USER_ID, "cardName-"+snownode.NewID())
	require.NoError(t, err)
	cleanupRow(t, `cards`, cardID)

	return cardID
}

// Creates a mapping with the values of m. ID of the mapping will be ignored
// ID of the mapping will appear in the m.ID (though its also returned for convenience sake)
// Will be cleaned up afterwards
func newTestMap(t *testing.T, m *data.Mapping) string {
	s := newStore(t)
	id, err := s.MappingNew(t.Context(), USER_ID, m)
	require.NoError(t, err)
	m.ID = id
	cleanupRow(t, `mappings`, id)

	return id
}

// Insert trans, filling in default values if they don't exist
// filling: id, user id, card id, authed at, settled at, amount
// The transactions will be cleaned up after
func newTestTrans(t *testing.T, trans []*store.TransactionsInsertParams) {
	s := newStore(t)

	n := time.Now()
	var defDec decimal.Decimal
	for _, v := range trans {
		if v.ID == "" {
			v.ID = snownode.NewID()
		}
		if v.AuthorID == "" {
			v.AuthorID = USER_ID
		}
		if v.CardID == "" {
			v.CardID = CARD_ID
		}
		if v.AuthedAt.IsZero() {
			v.AuthedAt = n
		}
		if v.SettledAt.IsZero() {
			v.SettledAt = n
		}
		if v.Amount == defDec { // checks if uninitialized
			v.Amount = decimal.New(1, 1)
		}
	}

	_, err := s.TransactionsInsert(t.Context(), trans)
	require.NoError(t, err)
	t.Cleanup(func() {
		arr := make([]string, len(trans))
		for i, v := range trans {
			arr[i] = v.ID
		}

		db := db.GetDB(slog.Default())
		db.Exec(context.Background(), `DELETE FROM transactions WHERE id = ANY ($1)`, arr) //nolint:errcheck
	})
}
