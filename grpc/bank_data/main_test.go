package bank_data_test

import (
	"context"
	"testing"

	"github.com/bank-data-db/server/config"
	"github.com/bank-data-db/server/db"
	"github.com/bank-data-db/server/db/store"
	"github.com/bank-data-db/server/db/store/mock_store"
	"github.com/bank-data-db/server/grpc/bank_data"
	"github.com/bank-data-db/server/tutils"
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
		tutils.InsertTestUserDB(USER_ID)
		tutils.InsertTestUserDB(USER_2_ID)
	}
}
