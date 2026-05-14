package bank_data_test

import (
	"context"
	"testing"

	"github.com/shadiestgoat/bankDataDB/db/store"
	"github.com/shadiestgoat/bankDataDB/db/store/mock_store"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data"
	"github.com/shadiestgoat/bankDataDB/tutils/factories"
)

func apiCtx(t *testing.T) context.Context {
	return bank_data.ContextWithUserID(t.Context(), factories.USER_ID)
}

func newAPI(t *testing.T) (*mock_store.MockStore, *bank_data.API) {
	s := mock_store.NewMockStore(t)

	return s, bank_data.NewAPI(nil, s)
}

func newAPIWithRealDB(t *testing.T) *bank_data.API {
	db := factories.DB(t)

	return bank_data.NewAPI(db, store.NewStore(db))
}

func TestMain(t *testing.M) {
	factories.RunWithPrimitives(t)
}
