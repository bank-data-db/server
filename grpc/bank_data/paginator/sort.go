package paginator

import (
	"context"
	"strings"

	"github.com/huandu/go-sqlbuilder"
	"github.com/shadiestgoat/bankDataDB/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PaginationSortRequest[SE ~int32] interface {
	PaginationRequest

	GetOrderBy() SE
	HasOrderBy() bool
	GetDescending() bool
	HasDescending() bool
}

type ColCfg[RV PaginationResponseValue] struct {
	DBName     string
	Marshal    func(v RV) string
	Unmarshall func(v string) (any, error)
}

type ConfSort[SE ~int32, REQ PaginationSortRequest[SE], RV PaginationResponseValue, RESP PaginationResponse[RV]] struct {
	ConfEasy[REQ, RV, RESP]

	EnumToCol      map[SE]*ColCfg[RV]
	DefaultSortCol SE
}

func (conf ConfSort[SE, REQ, RV, RESP]) sortCol(req REQ) (SE, error) {
	if !req.HasOrderBy() {
		return conf.DefaultSortCol, nil
	}
	se := req.GetOrderBy()

	_, ok := conf.EnumToCol[se]
	if !ok {
		return 0, status.Error(codes.InvalidArgument, "unknown sort column")
	}

	return se, nil
}

func (conf ConfSort[SE, REQ, RV, RESP]) RunQuery(
	ctx context.Context, db db.DBQuerier,
	baseQuery *sqlbuilder.SelectBuilder,
	req REQ, resp RESP,
) error {
	nonPaginatedQuery := baseQuery.Clone()

	rawCol, err := conf.sortCol(req)
	if err != nil {
		return err
	}
	col := conf.EnumToCol[rawCol]

	desc := true
	if req.HasDescending() {
		desc = req.GetDescending()
		baseQuery.OrderByDesc(col.DBName)
	} else {
		baseQuery.OrderByAsc(col.DBName)
	}

	if req.HasPaginationToken() {
		parts := strings.Split(req.GetPaginationToken(), "|")
		if len(parts) != 2 {
			return status.Errorf(codes.InvalidArgument, "please use the pagination token provided from a previous request")
		}

		dir := baseQuery.LT
		if desc {
			dir = baseQuery.GT
		}

		dbVal, err := col.Unmarshall(parts[0])
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "please use the pagination token provided from a previous request")
		}

		baseQuery.Where(
			baseQuery.Or(
				dir(col.DBName, dbVal),
				baseQuery.And(
					baseQuery.EQ(col.DBName, dbVal),
					baseQuery.LT("id", parts[1]),
				),
			),
		)
	}

	return conf.runQuery(
		ctx, db,
		baseQuery, nonPaginatedQuery,
		req, resp, func(r RV) string {
			return conf.EnumToCol[rawCol].Marshal(r) + "|" + r.GetID()
		},
	)
}
