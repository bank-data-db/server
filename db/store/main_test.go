package store_test

import (
	"testing"

	"github.com/shadiestgoat/bankDataDB/db/store"
	"github.com/shadiestgoat/bankDataDB/tutils/factories"
)

func TestMain(m *testing.M) {
	factories.RunWithPrimitives(m)
}

func newStore(t *testing.T) store.Store {
	return store.NewStore(factories.DB(t))
}
