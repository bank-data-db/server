package bank_data

import (
	"context"
	"time"

	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v5"
	"github.com/shadiestgoat/bankDataDB/db/store"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data/lerrors"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data/paginator"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data/validator"
	"github.com/shadiestgoat/bankDataDB/internal"
	"github.com/shadiestgoat/bankDataDB/pb/bank_svc_pb"
	"github.com/shadiestgoat/bankDataDB/pb/transactions"
	"github.com/shadiestgoat/bankDataDB/snownode"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
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

var validatorTransactions = &validator.Validator{
	Validations: []validator.Validation{
		validator.NewRequiredFieldValidation("card_id"),
		validator.NewRequiredFieldValidation("settled_at"),
		validator.NewRequiredFieldValidation("authed_at"),
		validator.NewRequiredFieldValidation("description"),
		validator.NewFieldValidation("amount", true, func(pr protoreflect.Value) *string {
			v := pr.Float()
			cents := v * 100

			// float64(int(x)) = drop the part after the dot.
			// As a side note, idk if this is fallible to the floating point arithmetic???
			if float64(int(cents)) != cents {
				return new("Too price: MUST contain at most 2 decimal places")
			}

			return nil
		}),
	},
}

// TransactionsNew implements [svc.BankDataServer].
func (a *API) TransactionsNew(ctx context.Context, req *transactions.ReqNew) (*transactions.RespNew, error) {
	if err := validatorTransactions.Validate(req); err != nil {
		return nil, err
	}

	t := &store.TransactionsInsertParams{
		ID:          snownode.NewID(),
		AuthorID:    userID(ctx),
		CardID:      req.GetCardID(),
		AuthedAt:    time.UnixMilli(req.GetAuthedAt()),
		SettledAt:   time.UnixMilli(req.GetSettledAt()),
		Description: req.GetDescription(),
		Amount:      decimal.NewFromFloat(req.GetAmount()),
	}
	if req.HasResolvedCategoryId() {
		t.ResolvedCategory = new(req.GetResolvedCategoryID())
	}
	if req.HasResolvedName() {
		t.ResolvedName = new(req.GetResolvedName())
	}
	bat := &pgx.Batch{}

	if !req.GetDoNotResolve() && (!req.HasResolvedCategoryId() || !req.HasResolvedName()) {
		maps, err := a.store.MappingGetAll(ctx, userID(ctx))
		if err != nil {
			return nil, lerrors.ErrDB
		}
		rn, rc := internal.MapSpecificTransaction(maps, req.GetAmount(), req.GetDescription(), req.GetCardID())
		if t.ResolvedName == nil && rn != nil {
			t.ResolvedName = &rn.Res
			a.store.BatchInsertTransMapping(bat, t.ID, rn.MappingID, true)
		}
		if t.ResolvedCategory == nil && rc != nil {
			t.ResolvedName = &rc.Res
			a.store.BatchInsertTransMapping(bat, t.ID, rc.MappingID, false)
		}
	}

	err := a.store.TxFunc(ctx, func(s store.Store) error {
		_, err := s.TransactionsInsert(ctx, []*store.TransactionsInsertParams{t})
		if err != nil {
			return err
		}

		if bat.Len() != 0 {
			return s.SendBatch(ctx, bat)
		}

		return nil
	})
	if err != nil {
		return nil, lerrors.ErrDB
	}

	return transactions.RespNew_builder{
		Id:                 new(t.ID),
		ResolvedName:       t.ResolvedName,
		ResolvedCategoryId: t.ResolvedCategory,
	}.Build(), nil
}

func (a *API) TransactionsUpdate(ctx context.Context, req *transactions.ReqUpdate) (*emptypb.Empty, error) {
	if req.GetID() == "" {
		return nil, lerrors.ErrIDRequired
	}
	ok, err := a.store.TransactionsExists(ctx, req.GetID(), userID(ctx))
	if err != nil {
		return nil, lerrors.ErrDB
	}
	if !ok {
		return nil, status.Error(codes.NotFound, "")
	}

	b := &pgx.Batch{}
	var name, catID **string
	if req.HasResolvedName() {
		if req.GetResolvedName() == "" {
			name = new(*string)
			a.store.BatchMappedTransactionDeleteNoMappingID(b, req.GetID(), true)
		} else {
			v := new(req.GetResolvedName())
			name = &v
		}
	}

	if req.HasResolvedCategoryId() {
		if req.GetResolvedName() == "" {
			catID = new(*string)
			a.store.BatchMappedTransactionDeleteNoMappingID(b, req.GetID(), false)
		} else {
			ok, err := a.store.CategoriesExists(ctx, req.GetID(), userID(ctx))
			if err != nil {
				return nil, lerrors.ErrDB
			}
			if !ok {
				return nil, status.Error(codes.InvalidArgument, "Category ID is invalid")
			}

			v := new(req.GetResolvedName())
			catID = &v
		}
	}

	a.store.BatchForceUpdateTrans(b, req.GetID(), name, catID)

	err = a.store.SendBatch(ctx, b)
	if err != nil {
		// TODO: Perhaps we should investigate errors for stuff like conflicts
		return nil, lerrors.ErrDB
	}

	return &emptypb.Empty{}, nil
}
