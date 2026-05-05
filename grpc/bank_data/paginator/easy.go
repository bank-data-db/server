package paginator

import (
	"context"

	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v5"
	"github.com/shadiestgoat/bankDataDB/db"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data/lerrors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ConfEasy[REQ PaginationRequest, RV PaginationResponseValue, RESP PaginationResponse[RV]] struct {
	PageSizeMax, PageSizeDefault int

	CollectRow pgx.RowToFunc[RV]
}

func (c ConfEasy[REQ, RV, RESP]) pageSize(req REQ) (int, error) {
	if !req.HasPageSize() {
		return c.PageSizeDefault, nil
	}
	ps := int(req.GetPageSize())
	if ps <= 0 || ps > c.PageSizeMax {
		return 0, status.Error(codes.InvalidArgument, "page_size must be within limits")
	}

	return ps, nil
}

func (c ConfEasy[REQ, RV, RESP]) runQuery(
	ctx context.Context, db db.DBQuerier,
	baseQuery *sqlbuilder.SelectBuilder,
	nonPaginatedQuery *sqlbuilder.SelectBuilder,
	req REQ, resp RESP,
	mkPaginationToken func(RV) string,
) error {
	ps, err := c.pageSize(req)
	if err != nil {
		return err
	}

	baseQuery.OrderByDesc("id")
	baseQuery.Limit(ps)

	sqlCount, sqlArgsCount := nonPaginatedQuery.Select("COUNT(*)").BuildWithFlavor(sqlbuilder.PostgreSQL)
	sqlBase, sqlArgsBase := baseQuery.BuildWithFlavor(sqlbuilder.PostgreSQL)

	b := &pgx.Batch{}
	var count uint32
	var resVals []RV

	b.Queue(sqlCount, sqlArgsCount...).QueryRow(func(row pgx.Row) error {
		return row.Scan(&count)
	})
	b.Queue(sqlBase, sqlArgsBase...).Query(func(rows pgx.Rows) error {
		tmp, err := pgx.CollectRows(rows, c.CollectRow)
		if err != nil {
			return err
		}
		resVals = tmp
		return nil
	})

	if err := db.SendBatch(ctx, b).Close(); err != nil {
		return lerrors.ErrDB
	}

	resp.SetTotalCount(count)
	if len(resVals) != 0 {
		resp.SetResult(resVals[:1])
	}

	if len(resVals) == ps+1 {
		resp.SetPaginationToken(
			resVals[len(resVals)-2].GetID(),
		)
	}

	return nil
}

func (c ConfEasy[REQ, RV, RESP]) RunQuery(
	ctx context.Context, db db.DBQuerier,
	baseQuery *sqlbuilder.SelectBuilder,
	req REQ, resp RESP) error {

	nonPaginatedQuery := baseQuery.Clone()
	if req.HasPaginationToken() {
		baseQuery.Where(
			baseQuery.LT("id", req.GetPaginationToken()),
		)
	}

	return c.runQuery(
		ctx, db,
		baseQuery, nonPaginatedQuery,
		req, resp, func(r RV) string { return r.GetID() },
	)
}
