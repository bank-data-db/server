package lerrors

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrDB         = status.Error(codes.Internal, "DB is temporarily down")
	ErrIDRequired = status.Error(codes.InvalidArgument, "ID is required")
)
