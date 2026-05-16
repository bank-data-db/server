package bank_data_test

import (
	"context"
	"math"
	"testing"

	"github.com/shadiestgoat/bankDataDB/db/store"
	"github.com/shadiestgoat/bankDataDB/db/store/mock_store"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data"
	"github.com/shadiestgoat/bankDataDB/pb/errors"
	"github.com/shadiestgoat/bankDataDB/tutils/factories"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func apiCtx(t *testing.T) context.Context {
	return bank_data.ContextWithUserID(t.Context(), factories.USER_ID)
}

func newAPI(t *testing.T) (*mock_store.MockStore, *bank_data.API) {
	s := mock_store.NewMockStore(t)

	return s, bank_data.NewAPI(nil, s)
}

func newAPIWithRealDB(t *testing.T) *bank_data.API {
	db := factories.DB(t)

	return bank_data.NewAPI(db, store.NewStore(db))
}

func TestMain(t *testing.M) {
	factories.RunWithPrimitives(t)
}

func assertValidationErrFields(t *testing.T, err error, fields ...string) {
	if !assert.Error(t, err) {
		return
	}

	s, ok := status.FromError(err)
	if !assert.True(t, ok, "non-status error returned") {
		return
	}

	assert.Equal(t, codes.InvalidArgument, s.Code())
	details := s.Details()

	assert.NotEmpty(t, details)
	foundFields := []string{}
	for _, d := range details {
		err, ok := d.(*errors.ValidationError)
		if !assert.True(t, ok, "not a validation error in the details") {
			continue
		}

		fields := err.GetFields()
		if !assert.NotEmpty(t, fields) {
			continue
		}

		foundFields = append(foundFields, fields...)
	}

	assert.ElementsMatch(t, fields, foundFields)
}

type PaginationResp[T interface{ GetID() string }] interface {
	GetTotalCount() uint32
	GetResult() []T
	HasPaginationToken() bool
	GetPaginationToken() string
}

func assertEndpointList[T interface{ GetID() string }, Resp PaginationResp[T]](t *testing.T, allIDs []string, pageSize int, list func(pageSize uint32, tok *string) (Resp, error)) {
	fetches := 0
	var tok *string

	fetchedIDs := []string{}

	for {
		resp, err := list(uint32(pageSize), tok)
		fetches++

		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, len(allIDs), int(resp.GetTotalCount()), "the total amount is wrong")
		assert.LessOrEqual(t, len(resp.GetResult()), pageSize, "the result page is greater than page size")

		for _, c := range resp.GetResult() {
			fetchedIDs = append(fetchedIDs, c.GetID())
		}

		if !resp.HasPaginationToken() {
			break
		}
		tok = new(resp.GetPaginationToken())
	}

	// Test to make sure we are terminating early
	assert.EqualValues(t, math.Ceil(float64(len(allIDs))/float64(pageSize)), fetches, "fetches have the wrong count")
	assert.ElementsMatch(t, fetchedIDs, allIDs, "ID mismatch! listA is the fetched list.")
}
