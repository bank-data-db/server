package store_test

import (
	"testing"

	"github.com/bank-data-db/server/db/store"
	"github.com/bank-data-db/server/tutils/factories"
)

func TestMain(m *testing.M) {
	factories.RunWithPrimitives(m)
}

func newStore(t *testing.T) store.Store {
	return store.NewStore(factories.DB(t))
}
