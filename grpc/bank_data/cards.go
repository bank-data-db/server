package bank_data

import (
	"context"
	"strings"

	"github.com/bank-data-db/proto/bank_svc_pb"
	"github.com/bank-data-db/proto/cards_pb"
	"github.com/bank-data-db/server/db"
	"github.com/bank-data-db/server/grpc/bank_data/lerrors"
	"github.com/bank-data-db/server/grpc/bank_data/paginator"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (a *API) CardDelete(ctx context.Context, req *bank_svc_pb.ReqDelete) (*emptypb.Empty, error) {
	userID := userID(ctx)
	id := req.GetID()

	if id == "" {
		return nil, lerrors.ErrIDRequired
	}

	return easyExecRowsResp(a.store.CardsDelete(ctx, userID, id))
}

var paginatorCard = &paginator.ConfEasy[*cards_pb.ReqList, *cards_pb.Card, *cards_pb.RespList]{
	PageSizeMax:     100,
	PageSizeDefault: 75,
	CollectRow: func(row pgx.CollectableRow) (*cards_pb.Card, error) {
		var id, name string

		if err := row.Scan(&id, &name); err != nil {
			return nil, err
		}

		return cards_pb.Card_builder{
			Id:   new(id),
			Name: new(name),
		}.Build(), nil
	},
}

func (a *API) CardsList(ctx context.Context, req *cards_pb.ReqList) (*cards_pb.RespList, error) {
	sb := sqlbuilder.NewSelectBuilder()
	resp := &cards_pb.RespList{}
	err := paginatorCard.RunQuery(
		ctx, a.db,
		sb.From(
			"cards",
		).Select("id", "name").Where(sb.EQ("user_id", userID(ctx))),
		req, resp,
	)

	return resp, err
}

func validateCardName(n string) (string, error) {
	n = strings.TrimSpace(n)
	if len(n) < 3 {
		return "", status.Error(codes.InvalidArgument, "Name too short")
	}

	return n, nil
}

func (a *API) CardsNew(ctx context.Context, req *cards_pb.ReqNew) (*bank_svc_pb.RespNew, error) {
	n, err := validateCardName(req.GetName())
	if err != nil {
		return nil, err
	}

	id, err := a.store.CardsNew(ctx, userID(ctx), n)
	if err != nil {
		if db.UniqueConstraint(err) {
			return nil, status.Error(codes.AlreadyExists, "A card with this name already exists")
		}

		return nil, lerrors.ErrDB
	}

	return bank_svc_pb.RespNew_builder{Id: new(id)}.Build(), nil
}

func (a *API) CardsUpdate(ctx context.Context, req *cards_pb.Card) (*emptypb.Empty, error) {
	n, err := validateCardName(req.GetName())
	if err != nil {
		return nil, err
	}
	if req.GetID() == "" {
		return nil, lerrors.ErrIDRequired
	}

	return easyExecRowsResp(a.store.CardsUpdate(ctx, userID(ctx), req.GetID(), n))
}
