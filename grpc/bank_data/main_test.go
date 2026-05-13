package bank_data_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/shadiestgoat/bankDataDB/config"
	"github.com/shadiestgoat/bankDataDB/db"
	"github.com/shadiestgoat/bankDataDB/db/store"
	"github.com/shadiestgoat/bankDataDB/db/store/mock_store"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data"
	"github.com/shadiestgoat/bankDataDB/tutils"
)

const (
	USER_ID   = "CUSTOM_USER_ID_HEHE"
	USER_2_ID = "OTHER USER!!!"
)

func apiCtx(t *testing.T) context.Context {
	return bank_data.ContextWithUserID(t.Context(), USER_ID)
}

func newAPI(t *testing.T) (*mock_store.MockStore, *bank_data.API) {
	s := mock_store.NewMockStore(t)

	return s, bank_data.NewAPI(nil, s)
}

func newAPIWithRealDB(t *testing.T) *bank_data.API {
	db := tutils.DB(t)

	return bank_data.NewAPI(db, store.NewStore(db))
}

func init() {
	config.LoadForTests()
	if db.DBDefined() {
		mkTestUser(USER_ID)
		mkTestUser(USER_2_ID)
	}
}

func mkTestUser(id string) {
	_, err := db.GetDB(slog.Default()).Exec(
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
