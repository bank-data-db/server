package patcher

import (
	"context"

	"github.com/bank-data-db/server/db"
	"github.com/huandu/go-sqlbuilder"
	"google.golang.org/protobuf/protoadapt"
)

type patchField struct {
	col string
	has func() bool
	get func() any
}

func PatchField[V any](col string, has func() bool, get func() V) *patchField {
	return &patchField{
		col: col,
		has: has,
		get: func() any {
			return get()
		},
	}
}

// Makes a "patch" request out of a protobuf message. Prerequisites:
//
// 1. table MUST have a author_id field
// 2. req MUST be valid
//
// The cols argument is a pair: the first is the sql col, the next is the grpc field
func Patch[T protoadapt.MessageV2](ctx context.Context, req T, db db.DBQuerier, table, userID, id string, cols ...*patchField) (int64, error) {
	ub := sqlbuilder.NewUpdateBuilder().Update(table)
	ub.Where(ub.EQ("author_id", userID), ub.EQ("id", id))

	for _, c := range cols {
		if !c.has() {
			continue
		}

		ub.SetMore(ub.Assign(c.col, c.get()))
	}

	sql, args := ub.Build()
	res, err := db.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected(), nil
}
