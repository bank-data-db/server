package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func queueUnmap(b *pgx.Batch, mappingID string, name bool) {
	// so in theory you shouldnt use bool = bool, but use IS TRUE/IS FALSE
	// HOWEVER updated_name has a NOT NULL restriction so we are good in saying =

	col := "resolved_category"
	if name {
		col = "resolved_name"
	}

	b.Queue(`
	WITH deleted AS (
		DELETE FROM mapped_transactions WHERE mapping_id = $1 AND updated_name = $2 RETURNING trans_id
	) UPDATE transactions SET ` + col + " = NULL FROM deleted d WHERE transactions.id = d.trans_id", mappingID, name)
}

// Delete the mapped_transaction AND unset the needed column.
func (s *DBStore) TransactionsUnmapForMappingID(ctx context.Context, mappingID string, unmapName, unmapCat bool) error {
	if !unmapName && !unmapCat {
		return nil
	}

	b := &pgx.Batch{}

	if unmapCat {
		queueUnmap(b, mappingID, false)
	}
	if unmapName {
		queueUnmap(b, mappingID, true)
	}

	return s.TxFunc(ctx, func(s Store) error {
		return s.SendBatch(ctx, b)
	})
}
