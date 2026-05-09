package bank_data

import (
	"context"
	"log/slog"

	"github.com/shadiestgoat/bankDataDB/db"
	"github.com/shadiestgoat/bankDataDB/db/store"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data/lerrors"
	"github.com/shadiestgoat/bankDataDB/internal"
	"github.com/shadiestgoat/bankDataDB/pb/bank_svc_pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ bank_svc_pb.BankDataServer = &API{}

type API struct {
	bank_svc_pb.UnsafeBankDataServer

	store store.Store
	// For query builder type of thing
	// Usually, queries are in [store.Store] but for some builders
	db db.DBQuerier
}

func NewAPI() *API {
	db := db.GetDB(slog.Default().With("parent_module", "bank_data"))
	return &API{
		store: store.NewStore(db),
		db:    db,
	}
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

func NewAuthInterceptor() grpc.UnaryServerInterceptor {
	store := store.NewStore(db.GetDB(slog.Default().With("parent_module", "bank_data_auth_interceptor")))

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if _, ok := info.Server.(*API); !ok {
			return handler(ctx, req)
		}

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
