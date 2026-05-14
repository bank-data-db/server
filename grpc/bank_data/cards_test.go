package bank_data_test

import (
	"math"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data/lerrors"
	"github.com/shadiestgoat/bankDataDB/pb/bank_svc_pb"
	"github.com/shadiestgoat/bankDataDB/pb/cards"
	"github.com/shadiestgoat/bankDataDB/tutils"
	"github.com/shadiestgoat/bankDataDB/tutils/factories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestAPI_CardDelete(t *testing.T) {
	cardID := "123"
	req := bank_svc_pb.ReqDelete_builder{Id: new(cardID)}

	t.Run("exists", func(t *testing.T) {
		s, api := newAPI(t)
		s.EXPECT().CardsDelete(mock.Anything, factories.USER_ID, cardID).Return(1, nil)
		_, err := api.CardDelete(apiCtx(t), req.Build())

		require.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		s, api := newAPI(t)
		s.EXPECT().CardsDelete(mock.Anything, factories.USER_ID, cardID).Return(0, pgx.ErrTxClosed) // idk, some random db error
		_, err := api.CardDelete(apiCtx(t), req.Build())

		require.Equal(t, err, lerrors.ErrDB) // wrap check, lmao
	})

	t.Run("not_exist", func(t *testing.T) {
		s, api := newAPI(t)
		s.EXPECT().CardsDelete(mock.Anything, factories.USER_ID, cardID).Return(0, nil)
		_, err := api.CardDelete(apiCtx(t), req.Build())

		tutils.RequireGRPCStatus(t, codes.NotFound, err)
	})
}

func TestAPI_CardsNew(t *testing.T) {
	t.Run("short_name", func(t *testing.T) {
		_, api := newAPI(t)
		_, err := api.CardsNew(apiCtx(t), cards.ReqNew_builder{Name: new("a")}.Build())

		tutils.RequireGRPCStatus(t, codes.InvalidArgument, err)
	})

	t.Run("duplicate", func(t *testing.T) {
		s, api := newAPI(t)
		s.EXPECT().CardsNew(mock.Anything, factories.USER_ID, "something").Return("", tutils.ErrDBUnique)

		_, err := api.CardsNew(apiCtx(t), cards.ReqNew_builder{Name: new("something")}.Build())

		tutils.RequireGRPCStatus(t, codes.AlreadyExists, err)
	})

	t.Run("happy", func(t *testing.T) {
		s, api := newAPI(t)
		s.EXPECT().CardsNew(mock.Anything, factories.USER_ID, "something").Return("123", nil)

		resp, err := api.CardsNew(apiCtx(t), cards.ReqNew_builder{Name: new("something")}.Build())

		require.NoError(t, err)
		require.Equal(t, resp.GetID(), "123")
	})
}

func TestAPI_CardsUpdate(t *testing.T) {
	t.Run("short_name", func(t *testing.T) {
		_, api := newAPI(t)
		_, err := api.CardsUpdate(apiCtx(t), cards.Card_builder{Name: new("a")}.Build())

		tutils.RequireGRPCStatus(t, codes.InvalidArgument, err)
	})

	t.Run("no_id", func(t *testing.T) {
		// I'm specifically making a situation where the id is empty but present
		_, api := newAPI(t)
		_, err := api.CardsUpdate(apiCtx(t), cards.Card_builder{Name: new("ssjnajdkmlsad"), Id: new("")}.Build())

		tutils.RequireGRPCStatus(t, codes.InvalidArgument, err)
	})

	t.Run("not_exist", func(t *testing.T) {
		s, api := newAPI(t)
		s.EXPECT().CardsUpdate(mock.Anything, factories.USER_ID, "123123", "NewName").Return(0, nil)

		_, err := api.CardsUpdate(apiCtx(t), cards.Card_builder{Id: new("123123"), Name: new("NewName")}.Build())

		tutils.RequireGRPCStatus(t, codes.NotFound, err)
	})

	t.Run("happy", func(t *testing.T) {
		s, api := newAPI(t)
		s.EXPECT().CardsUpdate(mock.Anything, factories.USER_ID, "123123", "NewName").Return(1, nil)

		_, err := api.CardsUpdate(apiCtx(t), cards.Card_builder{Id: new("123123"), Name: new("NewName")}.Build())

		require.NoError(t, err)
	})
}

func TestAPI_CardsList(t *testing.T) {
	s := factories.Store(t)

	cardNames := []string{"card1", "card TWO!!", "card.... THREE"}
	cardIDs := make([]string, len(cardNames))
	for i, v := range cardNames {
		id, err := s.CardsNew(t.Context(), factories.USER_ID, v)
		require.NoError(t, err)
		factories.CleanupRow(t, `cards`, id)
		cardIDs[i] = id

		// Also fill in cards for not us. Just to make sure we don't mess ourselves up t-t
		id2, err := s.CardsNew(t.Context(), factories.USER_ID_2, v)
		require.NoError(t, err)
		factories.CleanupRow(t, `cards`, id2)
	}

	totalCards := len(cardNames) + 1 // +1 is bc we have the "primitive" card

	testForSize := func(pageSize int) func(t *testing.T) {
		return func(t *testing.T) {
			api := newAPIWithRealDB(t)
			fetches := 0
			var tok *string

			for {
				resp, err := api.CardsList(apiCtx(t), cards.ReqList_builder{
					PageSize:        new(uint32(pageSize)),
					PaginationToken: tok,
				}.Build())
				fetches++

				require.NoError(t, err)
				assert.Equal(t, totalCards, int(resp.GetTotalCount()), "the total amount is wrong")
				assert.LessOrEqual(t, len(resp.GetResult()), pageSize, "the result page is greater than page size")

				for _, c := range resp.GetResult() {
					if c.GetID() == factories.CARD_ID {
						continue
					}

					ci := slices.Index(cardNames, c.GetName())
					if assert.NotEqual(t, -1, ci) {
						assert.Equal(t, cardIDs[ci], c.GetID())
					}
				}

				if !resp.HasPaginationToken() {
					break
				}
				tok = new(resp.GetPaginationToken())
			}

			// Test to make sure we are terminating early
			assert.EqualValues(t, math.Ceil(float64(totalCards)/float64(pageSize)), fetches, "fetches have the wrong count")
		}
	}

	// 1 for common off-by-1 mistakes
	t.Run("page_size=1", testForSize(1))
	// 2 for exact division
	t.Run("page_size=2", testForSize(2))
	// 3 for in-exact division
	t.Run("page_size=3", testForSize(2))
}
