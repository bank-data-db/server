package bank_data

import (
	"context"
	"time"

	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v5"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data/lerrors"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data/paginator"
	"github.com/shadiestgoat/bankDataDB/pb/bank_svc_pb"
	"github.com/shadiestgoat/bankDataDB/pb/transactions"
	"google.golang.org/protobuf/types/known/emptypb"
)

// TransactionsDelete implements [svc.BankDataServer].
func (a *API) TransactionsDelete(ctx context.Context, req *bank_svc_pb.ReqDelete) (*emptypb.Empty, error) {
	if req.GetID() == "" {
		return nil, lerrors.ErrIDRequired
	}

	return easyExecRowsResp(a.store.TransactionsDelete(ctx, userID(ctx), req.GetID()))
}

var paginatorTransactions = &paginator.ConfSort[transactions.OrderField, *transactions.ReqList, *transactions.Transaction, *transactions.RespList]{
	ConfEasy: paginator.ConfEasy[*transactions.ReqList, *transactions.Transaction, *transactions.RespList]{
		PageSizeMax:     100,
		PageSizeDefault: 75,
		CollectRow: func(row pgx.CollectableRow) (*transactions.Transaction, error) {
			v := transactions.Transaction_builder{}
			var authed, settled time.Time

			if err := row.Scan(
				v.Id,
				v.CardId,
				&settled, &authed,
				v.Description, v.Amount,
				&v.ResolvedName, &v.ResolvedCategoryId,
			); err != nil {
				return nil, err
			}

			v.AuthedAt = new(authed.UnixMilli())
			v.SettledAt = new(settled.UnixMilli())

			return v.Build(), nil
		},
	},
	EnumToCol: map[transactions.OrderField]*paginator.ColCfg[*transactions.Transaction]{
		transactions.OrderFieldAuthedAt: paginator.ColCfgUnixMilli(
			"authed_at", func(v *transactions.Transaction) int64 { return v.GetAuthedAt() },
		),
		transactions.OrderFieldSettledAt: paginator.ColCfgUnixMilli(
			"settled_at", func(v *transactions.Transaction) int64 { return v.GetSettledAt() },
		),
		transactions.OrderFieldAmount: paginator.ColCfgFloat(
			"amount", func(v *transactions.Transaction) float64 { return v.GetAmount() },
		),
	},
	DefaultSortCol: transactions.OrderFieldAuthedAt,
}

func (a *API) TransactionsList(ctx context.Context, req *transactions.ReqList) (*transactions.RespList, error) {
	sb := sqlbuilder.NewSelectBuilder().From(
		"transactions",
	).Select(
		"id",
		"card_id",
		"settled_at", "authed_at",
		"description", "amount",
		"resolved_name", "resolved_category",
	)

	sb.Where(sb.EQ("author_id", userID(ctx)))
	if req.HasCardId() {
		sb.Where(sb.EQ("card_id", req.GetCardID()))
	}

	resp := &transactions.RespList{}
	err := paginatorTransactions.RunQuery(ctx, a.db, sb, req, resp)

	return resp, err
}

// TransactionsNew implements [svc.BankDataServer].
func (a *API) TransactionsNew(ctx context.Context, req *transactions.ReqNew) (*bank_svc_pb.RespNew, error) {
	panic("unimplemented")
}

func (a *API) TransactionsUpdate(ctx context.Context, req *transactions.Transaction) (*emptypb.Empty, error) {
	panic("unimplemented")
}
