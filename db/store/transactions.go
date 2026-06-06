package store

import (
	"context"
	"slices"

	"github.com/jackc/pgx/v5"
)

func queueUnmap(b *pgx.Batch, mappingID string, name bool) *pgx.QueuedQuery {
	// so in theory you shouldnt use bool = bool, but use IS TRUE/IS FALSE
	// HOWEVER updated_name has a NOT NULL restriction so we are good in saying =

	col := "resolved_category"
	if name {
		col = "resolved_name"
	}

	return b.Queue(`
	WITH deleted AS (
		DELETE FROM mapped_transactions WHERE mapping_id = $1 AND updated_name = $2 RETURNING trans_id
	) UPDATE transactions SET `+col+` = NULL FROM deleted d WHERE transactions.id = d.trans_id
	 RETURNING id, amount, card_id, description
	`, mappingID, name)
}

func collector(cat bool, rows pgx.Rows) ([]*MappingsUnmapTransactionsRow, error) {
	res := []*MappingsUnmapTransactionsRow{}
	for rows.Next() {
		v := &MappingsUnmapTransactionsRow{
			UpName: !cat,
			UpCat:  cat,
		}
		if err := rows.Scan(&v.ID, &v.Amount, &v.CardID, &v.Description); err != nil {
			return nil, err
		}
		res = append(res, v)
	}

	return res, nil
}

// Delete the mapped_transaction AND unset the needed column.
func (s *DBStore) TransactionsUnmapForMappingID(ctx context.Context, mappingID string, unmapName, unmapCat bool) ([]*MappingsUnmapTransactionsRow, error) {
	if !unmapName && !unmapCat {
		return nil, nil
	}

	b := &pgx.Batch{}

	unmappedByCat := []*MappingsUnmapTransactionsRow{}
	unmappedByName := []*MappingsUnmapTransactionsRow{}

	if unmapCat {
		queueUnmap(b, mappingID, false).Query(func(rows pgx.Rows) error {
			c, err := collector(true, rows)
			unmappedByCat = c
			return err
		})
	}
	if unmapName {
		queueUnmap(b, mappingID, true).Query(func(rows pgx.Rows) error {
			c, err := collector(false, rows)
			unmappedByName = c
			return err
		})
	}

	err := s.TxFunc(ctx, func(s Store) error {
		return s.SendBatch(ctx, b)
	})
	if err != nil {
		return nil, err
	}

	slices.SortFunc(unmappedByCat, func(a, b *MappingsUnmapTransactionsRow) int {
		if a.ID > b.ID {
			return -1
		} else {
			return 1
		}
	})
	slices.SortFunc(unmappedByName, func(a, b *MappingsUnmapTransactionsRow) int {
		if a.ID > b.ID {
			return -1
		} else {
			return 1
		}
	})

	total := make([]*MappingsUnmapTransactionsRow, len(unmappedByCat), len(unmappedByCat)+len(unmappedByName))
	copy(total, unmappedByCat)

	for _, v := range unmappedByName {
		i, ok := slices.BinarySearchFunc(unmappedByCat, v.ID, func(a *MappingsUnmapTransactionsRow, v string) int {
			if a.ID == v {
				return 0
			}
			if a.ID < v {
				return -1
			}
			return 1
		})
		if ok {
			total[i].UpName = true
		} else {
			total = append(total, v)
		}
	}

	return total, nil
}
