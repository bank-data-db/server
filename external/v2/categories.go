package v2

import (
	"context"

	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v5"
	"github.com/shadiestgoat/bankDataDB/external/v2/lerrors"
	"github.com/shadiestgoat/bankDataDB/external/v2/paginator"
	"github.com/shadiestgoat/bankDataDB/pb/categories"
	"github.com/shadiestgoat/bankDataDB/pb/svc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// CategoriesDelete implements [svc.BankDataServer].
func (a *API) CategoriesDelete(ctx context.Context, req *svc.ReqDelete) (*emptypb.Empty, error) {
	if req.GetID() == "" {
		return nil, lerrors.ErrIDRequired
	}

	return easyExecRowsResp(a.store.CategoriesDelete(ctx, userID(ctx), req.GetID()))
}

var paginatorCategory = &paginator.ConfEasy[*categories.ReqList, *categories.Category, *categories.RespList]{
	PageSizeMax:     100,
	PageSizeDefault: 75,
	CollectRow: func(row pgx.CollectableRow) (*categories.Category, error) {
		c := categories.Category_builder{}

		if err := row.Scan(&c.Id, &c.Name, &c.Color, &c.Icon); err != nil {
			return nil, err
		}

		return c.Build(), nil
	},
}

func (a *API) CategoriesList(ctx context.Context, req *categories.ReqList) (*categories.RespList, error) {
	sb := sqlbuilder.NewSelectBuilder()
	resp := &categories.RespList{}
	err := paginatorCategory.RunQuery(
		ctx, a.db,
		sb.From(
			"categories",
		).Select("id", "name", "color", "icon").Where(
			sb.EQ("author_id", userID(ctx)),
		),
		req, resp,
	)

	return resp, err
}

// CategoriesNew implements [svc.BankDataServer].
func (a *API) CategoriesNew(ctx context.Context, req *categories.ReqNew) (*svc.RespNew, error) {
	panic("unimplemented")
}

// CategoriesUpdate implements [svc.BankDataServer].
func (a *API) CategoriesUpdate(ctx context.Context, req *categories.Category) (*emptypb.Empty, error) {
	panic("Not implemented")
}
