package bank_data_test

import (
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data/lerrors"
	"github.com/shadiestgoat/bankDataDB/pb/bank_svc_pb"
	"github.com/shadiestgoat/bankDataDB/pb/cards"
	"github.com/shadiestgoat/bankDataDB/tutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestAPI_CardDelete(t *testing.T) {
	cardID := "123"
	req := bank_svc_pb.ReqDelete_builder{Id: new(cardID)}

	t.Run("exists", func(t *testing.T) {
		s, api := newAPI(t)
		s.EXPECT().CardsDelete(mock.Anything, USER_ID, cardID).Return(1, nil)
		_, err := api.CardDelete(apiCtx(t), req.Build())

		require.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		s, api := newAPI(t)
		s.EXPECT().CardsDelete(mock.Anything, USER_ID, cardID).Return(0, pgx.ErrTxClosed) // idk, some random db error
		_, err := api.CardDelete(apiCtx(t), req.Build())

		require.Equal(t, err, lerrors.ErrDB) // wrap check, lmao
	})

	t.Run("not_exist", func(t *testing.T) {
		s, api := newAPI(t)
		s.EXPECT().CardsDelete(mock.Anything, USER_ID, cardID).Return(0, nil)
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
		s.EXPECT().CardsNew(mock.Anything, USER_ID, "something").Return("", tutils.ErrDBUnique)

		_, err := api.CardsNew(apiCtx(t), cards.ReqNew_builder{Name: new("something")}.Build())

		tutils.RequireGRPCStatus(t, codes.AlreadyExists, err)
	})

	t.Run("happy", func(t *testing.T) {
		s, api := newAPI(t)
		s.EXPECT().CardsNew(mock.Anything, USER_ID, "something").Return("123", nil)

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
		s.EXPECT().CardsUpdate(mock.Anything, USER_ID, "123123", "NewName").Return(0, nil)

		_, err := api.CardsUpdate(apiCtx(t), cards.Card_builder{Id: new("123123"), Name: new("NewName")}.Build())

		tutils.RequireGRPCStatus(t, codes.NotFound, err)
	})

	t.Run("happy", func(t *testing.T) {
		s, api := newAPI(t)
		s.EXPECT().CardsUpdate(mock.Anything, USER_ID, "123123", "NewName").Return(1, nil)

		_, err := api.CardsUpdate(apiCtx(t), cards.Card_builder{Id: new("123123"), Name: new("NewName")}.Build())

		require.NoError(t, err)
	})
}
