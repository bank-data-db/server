package v2

import (
	"context"
	"strings"

	"github.com/shadiestgoat/bankDataDB/bank_parser"
	"github.com/shadiestgoat/bankDataDB/pb/svc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UploadBankSheet implements [svc.BankDataServer].
func (a *API) UploadBankSheet(ctx context.Context, req *svc.ReqBankSheet) (*svc.RespBankSheet, error) {
	bankSheet := req.GetBankSheet()

	seq, err := bank_parser.Iter(ctx, strings.NewReader(bankSheet))
	if err != nil || seq == nil {
		return nil, status.Error(codes.InvalidArgument, "unrecognizable bank sheet")
	}

	panic("unimplemented")
}
