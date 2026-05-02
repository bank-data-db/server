package internal

import (
	"context"
	"errors"

	"github.com/shadiestgoat/bankDataDB/data"
)

type TransactionOrderBy string

const (
	TRANS_ORDER_AUTH   = "authedAt"
	TRANS_ORDER_SETTLE = "settledAt"
	TRANS_ORDER_AMT    = "amount"
)

func columnInputToTrans(inp TransactionOrderBy) string {
	switch inp {
	case TRANS_ORDER_AMT:
		return "amount"
	case TRANS_ORDER_AUTH:
		return "authed_at"
	case TRANS_ORDER_SETTLE:
		return "settled_at"
	}

	return "authed_at"
}

func (a *API) GetTransactions(ctx context.Context, authorID string, amount, offset int, orderBy TransactionOrderBy, asc bool) ([]*data.Transaction, error) {
	return a.store.GetTransactions(
		ctx, authorID,
		amount, offset, columnInputToTrans(orderBy), asc,
	)
}

func (a *API) GetTransactionsCount(ctx context.Context, authorID string) (int, error) {
	c, err := a.store.GetTransCount(ctx, authorID)
	return int(c), err
}

func (a *API) GetTransaction(ctx context.Context, id string) (int) {

}

var ErrStaleUpdate = errors.New("stale data")

func (a *API) UpdateTransaction(ctx context.Context, tx *data.Transaction, name *string, catID *string) error {
	if name != nil {
		n := *name
		if n == "" {
			if tx.ResolvedName == nil {
				return ErrStaleUpdate
			}
			tx.ResolvedName = nil
		} else {
			
		}
	}
	if catID != nil {

	}
}
