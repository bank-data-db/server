package bank_data_test

import (
	"context"
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
		tutils.InsertTestUserDB(USER_ID)
		tutils.InsertTestUserDB(USER_2_ID)
	}
}
