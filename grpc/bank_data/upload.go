package bank_data

import (
	"context"
	"strings"

	"github.com/shadiestgoat/bankDataDB/bank_parser"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data/lerrors"
	"github.com/shadiestgoat/bankDataDB/internal"
	"github.com/shadiestgoat/bankDataDB/pb/bank_svc_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UploadBankSheet implements [svc.BankDataServer].
func (a *API) UploadBankSheet(ctx context.Context, req *bank_svc_pb.ReqBankSheet) (*bank_svc_pb.RespBankSheet, error) {
	bankSheet := req.GetBankSheet()

	seq, err := bank_parser.Iter(ctx, strings.NewReader(bankSheet))
	if err != nil || seq == nil {
		return nil, status.Error(codes.InvalidArgument, "unrecognizable bank sheet")
	}

	resp, err := internal.UploadBankIter(ctx, a.store, req.GetCardID(), seq, userID(ctx))
	if err != nil {
		return nil, lerrors.ErrDB
	}

	return bank_svc_pb.RespBankSheet_builder{
		NewTransactions:       new(uint64(resp.NewTransactions)),
		DuplicateTransactions: new(uint64(resp.SkippedTransactions)),
		UnmappedTransactions:  new(uint64(resp.UnmappedTransactions)),
	}.Build(), nil
}
