package v2

import (
	"context"
	"strings"

	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v5"
	"github.com/shadiestgoat/bankDataDB/external/v2/lerrors"
	"github.com/shadiestgoat/bankDataDB/external/v2/paginator"
	"github.com/shadiestgoat/bankDataDB/pb/cards"
	"github.com/shadiestgoat/bankDataDB/pb/svc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (a *API) CardDelete(ctx context.Context, req *svc.ReqDelete) (*emptypb.Empty, error) {
	userID := userID(ctx)
	id := req.GetID()

	if id == "" {
		return nil, lerrors.ErrIDRequired
	}

	return easyExecRowsResp(a.store.CardsDelete(ctx, userID, id))
}

var paginatorCard = &paginator.ConfEasy[*cards.ReqList, *cards.Card, *cards.RespList]{
	PageSizeMax:     100,
	PageSizeDefault: 75,
	CollectRow: func(row pgx.CollectableRow) (*cards.Card, error) {
		var id, name string

		if err := row.Scan(&id, &name); err != nil {
			return nil, err
		}

		return cards.Card_builder{
			Id:   new(id),
			Name: new(name),
		}.Build(), nil
	},
}

func (a *API) CardsList(ctx context.Context, req *cards.ReqList) (*cards.RespList, error) {
	sb := sqlbuilder.NewSelectBuilder()
	resp := &cards.RespList{}
	err := paginatorCard.RunQuery(
		ctx, a.db,
		sb.From(
			"cards",
		).Select("id", "name").Where(sb.EQ("user_id", userID(ctx))),
		req, resp,
	)

	return resp, err
}

func validateName(n string) (string, error) {
	n = strings.TrimSpace(n)
	if len(n) < 3 {
		return "", status.Error(codes.InvalidArgument, "Name too short")
	}

	return n, nil
}

func (a *API) CardsNew(ctx context.Context, req *cards.ReqNew) (*svc.RespNew, error) {
	n, err := validateName(req.GetName())
	if err != nil {
		return nil, err
	}

	id, err := a.store.CardsNew(ctx, userID(ctx), n)
	if err != nil {
		return nil, lerrors.ErrDB
	}

	return svc.RespNew_builder{Id: new(id)}.Build(), nil
}

func (a *API) CardsUpdate(ctx context.Context, req *cards.Card) (*emptypb.Empty, error) {
	n, err := validateName(req.GetName())
	if err != nil {
		return nil, err
	}
	if req.GetID() == "" {
		return nil, lerrors.ErrIDRequired
	}

	return easyExecRowsResp(a.store.CardsUpdate(ctx, n, req.GetID(), n))
}
