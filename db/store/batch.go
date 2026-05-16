package store

import (
	"time"

	"github.com/jackc/pgx/v5"
)

// I want these to be on DBStore for ease of test mocking

func (DBStore) BatchForceUpdateTrans(batch *pgx.Batch, id string, name, catID **string) {
	args := pgx.NamedArgs{
		"id": id,
	}
	sql := `UPDATE transactions SET `

	if name != nil {
		args["name"] = *name
		sql += `resolved_name = @name`
	}

	if catID != nil {
		if name != nil {
			sql += ", "
		}
		args["catID"] = *catID
		sql += "resolved_category_id = @catID"
	}

	sql += " WHERE id = @id"

	batch.Queue(sql, args)
}

func (DBStore) BatchInsertTransMapping(batch *pgx.Batch, transID, mappingID string, updatesName bool) {
	batch.Queue(
		`INSERT INTO mapped_transactions (trans_id, mapping_id, updated_name) VALUES ($1, $2, $3)`,
		transID, mappingID, updatesName,
	)
}

func (DBStore) BatchCheckpointsNew(batch *pgx.Batch, cardID string, date time.Time, amt float64) {
	batch.Queue(`INSERT INTO checkpoints (created_at, card_id, amount) VALUES ($1, $2) ON CONFLICT (idx_uniq_checkpoint) DO UPDATE SET amount = $2`, date, amt)
}

func (DBStore) BatchMappedTransactionDeleteNoMappingID(batch *pgx.Batch, transID string, name bool) {
	batch.Queue(
		`DELETE FROM mapped_transactions WHERE transaction_id = $1 AND updated_name = $2`,
		transID, name,
	)
}
