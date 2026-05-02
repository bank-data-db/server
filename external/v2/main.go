package v2

import (
	"context"

	"github.com/shadiestgoat/bankDataDB/db"
	"github.com/shadiestgoat/bankDataDB/db/store"
	"github.com/shadiestgoat/bankDataDB/external/v2/lerrors"
	"github.com/shadiestgoat/bankDataDB/internal"
	"github.com/shadiestgoat/bankDataDB/pb/svc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ svc.BankDataServer = &API{}

type API struct {
	svc.UnimplementedBankDataServer

	store store.Store
	// For query builder type of thing
	// Usually, queries are in [store.Store] but for some builders
	db db.DBQuerier
}

func userID(ctx context.Context) string {
	v := ctx.Value(ctx_user_id)
	if v == nil {
		// we want to be as loud in failures as possible in our auth section
		panic("User key not present, somehow")
	}
	id, ok := v.(string)
	if !ok {
		panic("User ID not a string, somehow?")
	}

	return id
}

// Return grpc err on error, not found if c == 0
func easyExecRowsResp(c int64, err error) (*emptypb.Empty, error) {
	if err != nil {
		return nil, lerrors.ErrDB
	}
	if c == 0 {
		return nil, status.Error(codes.NotFound, "")
	}

	return &emptypb.Empty{}, nil
}

type ctxKey int

const (
	ctx_user_id ctxKey = iota
)

func NewAuthInterceptor(store store.Store) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "No auth token present")
		}

		h, ok := md["authorization"]
		if !ok || len(h) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "Invalid auth header")
		}

		userID := internal.ExchangeToken(ctx, store, h[0])
		if userID == nil {
			return nil, status.Errorf(codes.Unauthenticated, "Bad authentication")
		}

		return handler(context.WithValue(ctx, ctx_user_id, *userID), req)
	}
}
