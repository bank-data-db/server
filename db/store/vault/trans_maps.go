package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func (s *DBStore) TransMapsInsert(ctx context.Context, transIDs []string, mappingID string, mappedName bool) error {
	i := 0
	_, err := s.db.CopyFrom(
		ctx,
		pgx.Identifier{`mapped_transactions`},
		[]string{`trans_id`, `mapping_id`, `updated_name`},
		pgx.CopyFromFunc(func() (row []any, err error) {
			if i >= len(transIDs) {
				return nil, nil
			}

			defer func() {
				i++
			}()

			return []any{transIDs[i], mappingID, mappedName}, nil
		}),
	)

	return err
}

type TransMapsBatch struct {
	// trans_id, mapping_id, updated_name
	Rows [][]any
}

func (t *TransMapsBatch) Insert(transID, mappingID string, updatedName bool) {
	t.Rows = append(t.Rows, []any{transID, mappingID, updatedName})
}

func (s *DBStore) TransMapsInsertBatch(ctx context.Context, b *TransMapsBatch) error {
	_, err := s.db.CopyFrom(
		ctx,
		pgx.Identifier{`mapped_transactions`},
		[]string{`trans_id`, `mapping_id`, `updated_name`},
		pgx.CopyFromRows(b.Rows),
	)

	return err
}
