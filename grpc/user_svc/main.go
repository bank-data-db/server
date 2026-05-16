package user_svc

import (
	"context"
	"errors"

	"github.com/bank-data-db/proto/user_svc_pb"
	"github.com/bank-data-db/server/db/store"
	"github.com/bank-data-db/server/internal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ user_svc_pb.UserServiceServer = &API{}

func NewAPI(s store.Store) *API {
	return &API{store: s}
}

type API struct {
	user_svc_pb.UnsafeUserServiceServer

	store store.Store
}

// CreateToken implements [user_svc.UserServiceServer].
func (a *API) CreateToken(ctx context.Context, req *user_svc_pb.ReqLogin) (*user_svc_pb.RespLogin, error) {
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

	return user_svc_pb.RespLogin_builder{Token: new(jwt)}.Build(), nil
}
