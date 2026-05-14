package factories

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

func Store(t *testing.T) store.Store {
	return store.NewStore(DB(t))
}

func DB(t *testing.T) db.DBQuerier {
	return db.GetDB(tutils.NewLogger(t))
}

const (
	USER_ID   string = "prim-usr-123"
	USER_ID_2 string = "prim-usr-123-2"
	CARD_ID   string = "prim-card-123"
)

func CleanupRow(t *testing.T, table string, id string) {
	t.Cleanup(func() {
		// Use the default logger so that if smt goes wrong, we get the full picture
		db := db.GetDB(slog.Default())
		// Ignore the err since its alr logged by the slog
		// We can't use t.Log bc the test would already be complete
		db.Exec(context.Background(), `DELETE FROM `+table+` WHERE id = $1`, id) //nolint:errcheck
	})
}

// creates a category & cleans it up
func NewCategory(t *testing.T) string {
	catID, err := Store(t).CategoriesNew(t.Context(), USER_ID, "catName-"+snownode.NewID(), "1", "ffffff")
	require.NoError(t, err)
	CleanupRow(t, `categories`, catID)

	return catID
}

// Creates a card & cleans it up. However, you probably want to use CARD_ID instead
func NewCard(t *testing.T) string {
	cardID, err := Store(t).CardsNew(t.Context(), USER_ID, "cardName-"+snownode.NewID())
	require.NoError(t, err)
	CleanupRow(t, `cards`, cardID)

	return cardID
}

// Creates a mapping with the values of m. ID of the mapping will be ignored
// ID of the mapping will appear in the m.ID (though its also returned for convenience sake)
// Will be cleaned up afterwards
func NewMapping(t *testing.T, m *data.Mapping) string {
	s := Store(t)
	id, err := s.MappingNew(t.Context(), USER_ID, m)
	require.NoError(t, err)
	m.ID = id
	CleanupRow(t, `mappings`, id)

	return id
}

// Insert trans, filling in default values if they don't exist
// filling: id, user id, card id, authed at, settled at, amount
// The transactions will be cleaned up after
func NewTrans(t *testing.T, trans []*store.TransactionsInsertParams) {
	s := Store(t)

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

// Creates a new user & cleans it up at the end of the test
// However, you likely want to use USER_ID
func NewUser(t *testing.T) string {
	id := snownode.NewID()
	_, err := DB(t).Exec(
		context.Background(),
		`INSERT INTO users (id, username, password) VALUES ($1, $2, $3)`,
		id, id, "fake-password-no-auth",
	)
	require.NoError(t, err)

	CleanupRow(t, `users`, id)

	return id
}

func RunWithPrimitives(m *testing.M) {
	config.LoadForTests()
	if db.DBDefined() {
		InitUser(USER_ID)
		InitUser(USER_ID_2)
		InitCard()
	}

	m.Run()

	if !db.DBDefined() {
		return
	}

	db := db.GetDB(slog.Default())
	db.Exec(context.Background(), `DELETE FROM cards WHERE id = $1`, CARD_ID)                  //nolint:errcheck
	db.Exec(context.Background(), `DELETE FROM users WHERE id IN ($1,$2)`, USER_ID, USER_ID_2) //nolint:errcheck
}

func InitUser(id string) {
	_, err := db.GetDB(slog.New(slog.DiscardHandler)).Exec(
		context.Background(),
		`INSERT INTO users (id, username, password) VALUES ($1, $2, $3)`,
		id, id, "fake-password-no-auth",
	)
	if err != nil {
		if !db.UniqueConstraint(err) {
			panic("Error setting up db user: " + err.Error())
		}
	}
}

func InitCard() {
	_, err := db.GetDB(slog.New(slog.DiscardHandler)).Exec(
		context.Background(),
		`INSERT INTO cards (id, name, user_id) VALUES ($1, $2, $3)`,
		CARD_ID, CARD_ID, USER_ID,
	)
	if err != nil {
		if !db.UniqueConstraint(err) {
			panic("Error setting up db card: " + err.Error())
		}
	}
}
