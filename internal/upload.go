package internal

import (
	"context"
	"iter"
	"log/slog"
	"time"

	"github.com/bank-data-db/server/bank_parser"
	"github.com/bank-data-db/server/db/store"
	"github.com/bank-data-db/server/snownode"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

type InsertResp struct {
	NewTransactions      uint `json:"newTransactions"`
	SkippedTransactions  uint `json:"skippedTransactions"`
	UnmappedTransactions uint `json:"unmappedTransactions"`
}

func UploadBankIter(ctx context.Context, s store.Store, defaultCardID string, transactions iter.Seq[*bank_parser.Transaction], authorID string) (*InsertResp, error) {
	mappings, err := s.MappingGetAll(ctx, authorID)
	if err != nil {
		return nil, err
	}

	resp := &InsertResp{
		NewTransactions:      0,
		SkippedTransactions:  0,
		UnmappedTransactions: 0,
	}

	var lastRowCheckpointDate time.Time
	batchCheckpoints := &pgx.Batch{}
	transInsert := []*store.TransactionsInsertParams{}
	batchTransMaps := []*store.MappedTransactionsInsertParams{}

	for t := range transactions {
		amt := decimal.NewFromFloat(t.Amt)
		cardID := defaultCardID
		if t.CardID != nil {
			cardID = *t.CardID
		}

		// TODO: Batching this would be nicer
		exist, err := s.TransactionsExistsNoID(ctx, cardID, t.AuthedAt, t.SettledAt, t.Description, amt)
		if err != nil {
			slog.ErrorContext(ctx, "Can't verify transaction existing", "error", err)
			continue
		}
		if exist {
			slog.DebugContext(ctx, "Skipping transaction insert because it already exists")
			resp.SkippedTransactions++
			continue
		}

		resolvedName, resolvedCat := MapSpecificTransaction(mappings, t.Amt, t.Description, cardID)
		if resolvedCat == nil && resolvedName == nil {
			resp.UnmappedTransactions++
		}

		tID := snownode.NewID()
		transInsert = append(transInsert, &store.TransactionsInsertParams{
			ID:               tID,
			AuthorID:         authorID,
			CardID:           cardID,
			AuthedAt:         t.AuthedAt,
			SettledAt:        t.SettledAt,
			Description:      t.Description,
			Amount:           amt,
			ResolvedName:     resolvedName.SafeValue(),
			ResolvedCategory: resolvedCat.SafeValue(),
		})

		if resolvedCat != nil {
			batchTransMaps = append(batchTransMaps, &store.MappedTransactionsInsertParams{
				TransID:     tID,
				MappingID:   resolvedCat.MappingID,
				UpdatedName: false,
			})
		}
		if resolvedName != nil {
			batchTransMaps = append(batchTransMaps, &store.MappedTransactionsInsertParams{
				TransID:     tID,
				MappingID:   resolvedName.MappingID,
				UpdatedName: true,
			})
		}

		if t.AmtAfterTransaction != nil {
			if !cmpDate(lastRowCheckpointDate, t.SettledAt) {
				s.BatchCheckpointsNew(batchCheckpoints, cardID, t.SettledAt, *t.AmtAfterTransaction)
			}
			lastRowCheckpointDate = t.SettledAt
		}
	}

	slog.InfoContext(ctx, "Writing transactions to db", "trans_amount", len(transInsert))
	err = s.TxFunc(ctx, func(s store.Store) error {
		c, err := s.TransactionsInsert(ctx, transInsert)
		if err != nil {
			return err
		}
		resp.NewTransactions = uint(c)

		err = s.SendBatch(ctx, batchCheckpoints)
		if err != nil {
			slog.ErrorContext(ctx, "Couldn't insert checkpoints", "error", err)
			return err
		}

		_, err = s.MappedTransactionsInsert(ctx, batchTransMaps)
		return err
	})

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func cmpDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()

	return ay == by && am == bm && ad == bd
}
