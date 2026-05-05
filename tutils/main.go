package tutils

import (
	"context"
	"log/slog"
	"testing"
	"unsafe"

	"github.com/shadiestgoat/bankDataDB/db"
	"github.com/shadiestgoat/bankDataDB/db/store"
	"github.com/shadiestgoat/bankDataDB/db/store/mock_store"
	"github.com/stretchr/testify/mock"
)

func MarkDBTest(t *testing.T) {
	if !db.DBDefined() {
		t.Skip("Skipping DB Test: no DB Defined!")
	}

	t.Log("Running a DB Test")
}

func MockStoreTx(t *testing.T, s *mock_store.MockStore) *mock_store.MockStore {
	tx := mock_store.NewMockStore(t)

	s.EXPECT().TxFunc(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, h func(s store.Store) error) error {
		return h(tx)
	})

	return tx
}

type testLogWriter struct {
	t *testing.T
}

func (t *testLogWriter) Write(b []byte) (int, error) {
	// Fuck you
	t.t.Log(unsafe.String(&b[0], len(b)-1))
	return len(b), nil
}

func NewLogger(t *testing.T) *slog.Logger {
	return slog.New(slog.NewTextHandler(&testLogWriter{t}, nil)).With("test_name", t.Name())
}

func DB(t *testing.T) db.DBQuerier {
	return db.GetDB(NewLogger(t))
}
