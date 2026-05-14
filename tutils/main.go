package tutils

import (
	"context"
	"log/slog"
	"testing"
	"unsafe"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shadiestgoat/bankDataDB/db"
	"github.com/shadiestgoat/bankDataDB/db/store"
	"github.com/shadiestgoat/bankDataDB/db/store/mock_store"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

var ErrDBUnique error =  &pgconn.PgError{Code: pgerrcode.UniqueViolation}

func RequireGRPCStatus(t *testing.T, c codes.Code, err error) {
	require.Equal(t, c, status.Code(err))
}

func InsertTestUserDB(id string) {
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
