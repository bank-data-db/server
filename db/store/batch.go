package store

import (
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
