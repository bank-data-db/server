package internal

import (
	"context"
	"iter"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shadiestgoat/bankDataDB/bank_parser"
	"github.com/shadiestgoat/bankDataDB/db/store"
	"github.com/shopspring/decimal"
)

type InsertResp struct {
	NewTransactions      int `json:"newTransactions"`
	SkippedTransactions  int `json:"skippedTransactions"`
	UnmappedTransactions int `json:"unmappedTransactions"`
}

func (a *API) UploadBankIter(ctx context.Context, transactions iter.Seq[*bank_parser.Transaction], authorID string) (*InsertResp, error) {
	mappings, err := a.store.MappingGetAll(ctx, authorID)
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
	batchTrans := &store.TransactionBatch{}
	batchTransMaps := &store.TransMapsBatch{}

	for t := range transactions {
		amt := decimal.NewFromFloat(t.Amt)
		// TODO: Batching this would be nicer
		exist, err := a.store.DoesTransactionExist(ctx, authorID, t.AuthedAt, t.SettledAt, t.Description, amt)
		if err != nil {
			a.log(ctx).Errorf("Can't verify transaction existing: %v", err)
			continue
		}
		if exist {
			a.log(ctx).Infof("Skipping transaction insert because it already exists")
			resp.SkippedTransactions++
			continue
		}

		resolvedName, resolvedCat := a.MapSpecificTransaction(mappings, t.Description, t.Amt)
		if resolvedCat == nil && resolvedName == nil {
			resp.UnmappedTransactions++
		}

		tID := batchTrans.Insert(t.AuthedAt, t.SettledAt, authorID, t.Description, amt, resolvedName.SafeValue(), resolvedCat.SafeValue())

		if resolvedCat != nil {
			batchTransMaps.Insert(tID, resolvedCat.MappingID, false)
		}
		if resolvedName != nil {
			batchTransMaps.Insert(tID, resolvedName.MappingID, false)
		}

		if t.AmtAfterTransaction != nil {
			if lastRowCheckpointDate != t.SettledAt {
				a.store.InsertCheckpoint(batchCheckpoints, t.SettledAt, *t.AmtAfterTransaction)
			}
			lastRowCheckpointDate = t.SettledAt
		}
	}

	a.log(ctx).Infow("Writing transactions to db", "amount", len(batchTrans.Rows))
	err = a.store.TxFunc(ctx, func(s store.Store) error {
		c, err := s.InsertTransactions(ctx, batchTrans)
		if err != nil {
			return err
		}
		resp.NewTransactions = int(c)

		err = s.SendBatch(ctx, batchCheckpoints)
		if err != nil {
			// Not a hard stopping err
			a.log(ctx).Errorw("Couldn't insert checkpoints", "error", err)
			return err
		}

		return s.TransMapsInsertBatch(ctx, batchTransMaps)
	})

	if err != nil {
		return nil, err
	}

	return resp, nil
}
