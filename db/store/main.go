package store

import (
	"context"

	"github.com/bank-data-db/server/db"
	"github.com/jackc/pgx/v5"
)

//go:generate go tool github.com/sqlc-dev/sqlc/cmd/sqlc generate
//go:generate ./../../gen/generate-post.sh
//go:generate go tool github.com/vburenin/ifacemaker -f "*.go" -s DBStore -i Store -p store -c "DONT EDIT: Auto generated" -o "store.go"
//go:generate go tool github.com/vektra/mockery/v3

func NewStore(db db.DBQuerier) *DBStore { return &DBStore{db} }

type DBStore struct{ db db.DBQuerier }

func (s *DBStore) SendBatch(ctx context.Context, b *pgx.Batch) error {
	return s.db.SendBatch(ctx, b).Close()
}

func (s *DBStore) TxFunc(ctx context.Context, h func(s Store) error) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		return h(&DBStore{tx})
	})
}

// Gets a raw DB conn from a store. Be careful using this.
func (s *DBStore) GetDB() db.DBQuerier {
	return s.db
}
