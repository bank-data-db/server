package bank_data

import (
	"context"

	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v5"
	"github.com/rivo/uniseg"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data/lerrors"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data/paginator"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data/patcher"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data/validator"
	"github.com/shadiestgoat/bankDataDB/pb/bank_svc"
	"github.com/shadiestgoat/bankDataDB/pb/categories"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/emptypb"
)

// CategoriesDelete implements [svc.BankDataServer].
func (a *API) CategoriesDelete(ctx context.Context, req *bank_svc.ReqDelete) (*emptypb.Empty, error) {
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

var validatorCategory = &validator.Validator{
	Validations: []validator.Validation{
		validator.NewFieldValidation("color", true, func(raw protoreflect.Value) *string {
			c := raw.String()
			if len(c) != 6 {
				return new("invalid color - must be hex without #")
			}
			for _, v := range c {
				if !((v >= '0' && v <= '9') || (v >= 'a' && v <= 'f')) {
					return new("invalid color - all characters must be lowercase!")
				}
			}

			return nil
		}),
		validator.NewFieldValidation("icon", true, func(v protoreflect.Value) *string {
			if uniseg.GraphemeClusterCount(v.String()) != 1 {
				return new("Needs to only be 1 character")
			}

			return nil
		}),
		validator.NewFieldValidation("name", true, func(v protoreflect.Value) *string {
			if len(v.String()) == 0 {
				return new("name is required")
			}

			return nil
		}),
	},
}

// CategoriesNew implements [svc.BankDataServer].
func (a *API) CategoriesNew(ctx context.Context, req *categories.ReqNew) (*bank_svc.RespNew, error) {
	if err := validatorCategory.Validate(req); err != nil {
		return nil, err
	}

	id, err := a.store.CategoriesNew(ctx, userID(ctx), req.GetName(), req.GetIcon(), req.GetColor())
	if err != nil {
		return nil, lerrors.ErrDB
	}

	return bank_svc.RespNew_builder{Id: new(id)}.Build(), nil
}

// CategoriesUpdate implements [svc.BankDataServer].
func (a *API) CategoriesUpdate(ctx context.Context, req *categories.Category) (*emptypb.Empty, error) {
	if req.GetID() == "" {
		return nil, status.Error(codes.InvalidArgument, "ID is required")
	}
	if err := validatorCategory.Validate(req); err != nil {
		return nil, err
	}

	return easyExecRowsResp(patcher.Patch(
		ctx, req, a.db, "categories", userID(ctx), req.GetID(),
		patcher.PatchField("name", req.HasName, req.GetName),
		patcher.PatchField("color", req.HasColor, req.GetColor),
		patcher.PatchField("icon", req.HasIcon, req.GetIcon),
	))
}
