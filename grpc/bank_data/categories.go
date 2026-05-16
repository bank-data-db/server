package bank_data

import (
	"context"

	"github.com/bank-data-db/proto/bank_svc_pb"
	"github.com/bank-data-db/proto/categories_pb"
	"github.com/bank-data-db/server/grpc/bank_data/lerrors"
	"github.com/bank-data-db/server/grpc/bank_data/paginator"
	"github.com/bank-data-db/server/grpc/bank_data/patcher"
	"github.com/bank-data-db/server/grpc/bank_data/validator"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v5"
	"github.com/rivo/uniseg"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/emptypb"
)

// CategoriesDelete implements [svc.BankDataServer].
func (a *API) CategoriesDelete(ctx context.Context, req *bank_svc_pb.ReqDelete) (*emptypb.Empty, error) {
	if req.GetID() == "" {
		return nil, lerrors.ErrIDRequired
	}

	err := a.store.MappingsDeleteForCategoryDelete(ctx, new(req.GetID()))
	if err != nil {
		return nil, lerrors.ErrDB
	}

	return easyExecRowsResp(a.store.CategoriesDelete(ctx, userID(ctx), req.GetID()))
}

var paginatorCategory = &paginator.ConfEasy[*categories_pb.ReqList, *categories_pb.Category, *categories_pb.RespList]{
	PageSizeMax:     100,
	PageSizeDefault: 75,
	CollectRow: func(row pgx.CollectableRow) (*categories_pb.Category, error) {
		c := categories_pb.Category_builder{}

		if err := row.Scan(&c.Id, &c.Name, &c.Color, &c.Icon); err != nil {
			return nil, err
		}

		return c.Build(), nil
	},
}

func (a *API) CategoriesList(ctx context.Context, req *categories_pb.ReqList) (*categories_pb.RespList, error) {
	sb := sqlbuilder.NewSelectBuilder()
	resp := &categories_pb.RespList{}
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

func validColorChar(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')
}

var validatorCategory = &validator.Validator{
	Validations: []validator.Validation{
		validator.NewFieldValidation("color", true, func(raw protoreflect.Value) *string {
			c := raw.String()
			if len(c) != 6 {
				return new("invalid color - must be hex without #")
			}
			for _, v := range c {
				if !validColorChar(v) {
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
func (a *API) CategoriesNew(ctx context.Context, req *categories_pb.ReqNew) (*bank_svc_pb.RespNew, error) {
	if err := validatorCategory.Validate(req); err != nil {
		return nil, err
	}

	id, err := a.store.CategoriesNew(ctx, userID(ctx), req.GetName(), req.GetIcon(), req.GetColor())
	if err != nil {
		return nil, lerrors.ErrDB
	}

	return bank_svc_pb.RespNew_builder{Id: new(id)}.Build(), nil
}

// CategoriesUpdate implements [svc.BankDataServer].
func (a *API) CategoriesUpdate(ctx context.Context, req *categories_pb.Category) (*emptypb.Empty, error) {
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
