package bank_data_test

import (
	"strconv"
	"testing"

	"github.com/bank-data-db/proto/bank_svc_pb"
	"github.com/bank-data-db/proto/cards_pb"
	"github.com/bank-data-db/server/grpc/bank_data/lerrors"
	"github.com/bank-data-db/server/tutils"
	"github.com/bank-data-db/server/tutils/factories"
	"github.com/jackc/pgx/v5"
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
		_, err := api.CardsNew(apiCtx(t), cards_pb.ReqNew_builder{Name: new("a")}.Build())

		tutils.RequireGRPCStatus(t, codes.InvalidArgument, err)
	})

	t.Run("duplicate", func(t *testing.T) {
		s, api := newAPI(t)
		s.EXPECT().CardsNew(mock.Anything, factories.USER_ID, "something").Return("", tutils.ErrDBUnique)

		_, err := api.CardsNew(apiCtx(t), cards_pb.ReqNew_builder{Name: new("something")}.Build())

		tutils.RequireGRPCStatus(t, codes.AlreadyExists, err)
	})

	t.Run("happy", func(t *testing.T) {
		s, api := newAPI(t)
		s.EXPECT().CardsNew(mock.Anything, factories.USER_ID, "something").Return("123", nil)

		resp, err := api.CardsNew(apiCtx(t), cards_pb.ReqNew_builder{Name: new("something")}.Build())

		require.NoError(t, err)
		require.Equal(t, resp.GetID(), "123")
	})
}

func TestAPI_CardsUpdate(t *testing.T) {
	t.Run("short_name", func(t *testing.T) {
		_, api := newAPI(t)
		_, err := api.CardsUpdate(apiCtx(t), cards_pb.Card_builder{Name: new("a")}.Build())

		tutils.RequireGRPCStatus(t, codes.InvalidArgument, err)
	})

	t.Run("no_id", func(t *testing.T) {
		// I'm specifically making a situation where the id is empty but present
		_, api := newAPI(t)
		_, err := api.CardsUpdate(apiCtx(t), cards_pb.Card_builder{Name: new("ssjnajdkmlsad"), Id: new("")}.Build())

		tutils.RequireGRPCStatus(t, codes.InvalidArgument, err)
	})

	t.Run("not_exist", func(t *testing.T) {
		s, api := newAPI(t)
		s.EXPECT().CardsUpdate(mock.Anything, factories.USER_ID, "123123", "NewName").Return(0, nil)

		_, err := api.CardsUpdate(apiCtx(t), cards_pb.Card_builder{Id: new("123123"), Name: new("NewName")}.Build())

		tutils.RequireGRPCStatus(t, codes.NotFound, err)
	})

	t.Run("happy", func(t *testing.T) {
		s, api := newAPI(t)
		s.EXPECT().CardsUpdate(mock.Anything, factories.USER_ID, "123123", "NewName").Return(1, nil)

		_, err := api.CardsUpdate(apiCtx(t), cards_pb.Card_builder{Id: new("123123"), Name: new("NewName")}.Build())

		require.NoError(t, err)
	})
}

func TestAPI_CardsList(t *testing.T) {
	s := factories.Store(t)

	// 1 gets added later on, for a total of 4
	cardIDs := make([]string, 3)
	for i := range cardIDs {
		id, err := s.CardsNew(t.Context(), factories.USER_ID, strconv.Itoa(i))
		require.NoError(t, err)
		factories.CleanupRow(t, `cards`, id)
		cardIDs[i] = id

		// Also fill in cards for not us. Just to make sure we don't mess ourselves up t-t
		id2, err := s.CardsNew(t.Context(), factories.USER_ID_2, strconv.Itoa(i))
		require.NoError(t, err)
		factories.CleanupRow(t, `cards`, id2)
	}

	cardIDs = append(cardIDs, factories.CARD_ID)

	t.Run("pagination", func(t *testing.T) {
		testForSize := func(pageSize int) func(t *testing.T) {
			return func(t *testing.T) {
				api := newAPIWithRealDB(t)

			assertEndpointList(t, cardIDs, pageSize, func(pageSize uint32, tok *string) (*cards_pb.RespList, error) {
				return api.CardsList(apiCtx(t), cards_pb.ReqList_builder{
					PageSize:        &pageSize,
					PaginationToken: tok,
				}.Build())
			})
		}
	}

		// 1 for common off-by-1 mistakes
		t.Run("page_size=1", testForSize(1))
		// 2 for exact division
		t.Run("page_size=2", testForSize(2))
		// 3 for in-exact division
		t.Run("page_size=3", testForSize(3))
	})

	t.Run("value", func(t *testing.T) {
		api := newAPIWithRealDB(t)
		resp, err := api.CardsList(apiCtx(t), cards_pb.ReqList_builder{PageSize: new(uint32(1))}.Build())
		require.NoError(t, err)
		require.Len(t, resp.GetResult(), 1)
		card := resp.GetResult()[0]

		assert.NotEmpty(t, card.GetID())
		assert.NotEmpty(t, card.GetName())
	})
}
