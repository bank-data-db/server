package usersvc

import (
	"context"
	"errors"

	"github.com/shadiestgoat/bankDataDB/db"
	"github.com/shadiestgoat/bankDataDB/db/store"
	"github.com/shadiestgoat/bankDataDB/internal"
	"github.com/shadiestgoat/bankDataDB/pb/user_svc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ user_svc.UserServiceServer = &API{}

type API struct {
	user_svc.UnsafeUserServiceServer

	store store.Store
	// For query builder type of thing
	// Usually, queries are in [store.Store] but for some builders
	db db.DBQuerier
}

// CreateToken implements [user_svc.UserServiceServer].
func (a *API) CreateToken(ctx context.Context, req *user_svc.ReqLogin) (*user_svc.RespLogin, error) {
	if len(req.GetPassword()) < 5 || len(req.GetUsername()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "username & password MUST be provided")
	}

	jwt, err := internal.Login(ctx, a.store, req.GetUsername(), req.GetPassword())
	if err != nil {
		if errors.Is(err, internal.ErrBadAuth) {
			return nil, status.Error(codes.Unauthenticated, "username/password is incorrect")
		}
		return nil, status.Error(codes.Internal, "unknown error")
	}

	return user_svc.RespLogin_builder{Token: new(jwt)}.Build(), nil
}
