package bank_data_test

import (
	"testing"

	"github.com/bank-data-db/proto/bank_svc_pb"
	"github.com/bank-data-db/proto/categories_pb"
	"github.com/bank-data-db/server/data"
	"github.com/bank-data-db/server/db/store"
	"github.com/bank-data-db/server/tutils"
	"github.com/bank-data-db/server/tutils/factories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestAPI_CategoriesList(t *testing.T) {
	s := factories.Store(t)

	allIDs := make([]string, 4)
	for i := range allIDs {
		id, err := s.CategoriesNew(t.Context(), factories.USER_ID, "some-name", "1", "ffffff")
		require.NoError(t, err)
		factories.CleanupRow(t, `categories`, id)
		allIDs[i] = id

		// Also fill in cards for not us. Just to make sure we don't mess ourselves up t-t
		id2, err := s.CategoriesNew(t.Context(), factories.USER_ID_2, "some-name", "1", "ffffff")
		require.NoError(t, err)
		factories.CleanupRow(t, `categories`, id2)
	}

	t.Run("pagination", func(t *testing.T) {
		testForSize := func(pageSize int) func(t *testing.T) {
			return func(t *testing.T) {
				api := newAPIWithRealDB(t)

				assertEndpointList(t, allIDs, pageSize, func(pageSize uint32, tok *string) (*categories_pb.RespList, error) {
					return api.CategoriesList(apiCtx(t), categories_pb.ReqList_builder{
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
		resp, err := api.CategoriesList(apiCtx(t), categories_pb.ReqList_builder{
			PageSize: new(uint32(1)),
		}.Build())
		require.NoError(t, err)
		require.Len(t, resp.GetResult(), 1)
		v := resp.GetResult()[0]

		assert.NotEmpty(t, v.GetID(), "id not present :(")
		assert.Equal(t, "ffffff", v.GetColor())
		assert.Equal(t, "some-name", v.GetName())
		assert.Equal(t, "1", v.GetIcon())
	})
}

func TestAPI_CategoriesNew(t *testing.T) {
	t.Run("partial", func(t *testing.T) {
		_, api := newAPI(t)
		_, err := api.CategoriesNew(apiCtx(t), categories_pb.ReqNew_builder{
			Color: new("ffffff"),
		}.Build())

		assertValidationErrFields(t, err, "name", "icon") // color is valid, so not that
	})

	t.Run("invalid", func(t *testing.T) {
		_, api := newAPI(t)
		_, err := api.CategoriesNew(apiCtx(t), categories_pb.ReqNew_builder{
			Name:  new(""),
			Color: new("not-a-color"),
			Icon:  new("123"),
		}.Build())

		assertValidationErrFields(t, err, "name", "icon", "color")
	})

	t.Run("happy", func(t *testing.T) {
		s, api := newAPI(t)
		eID := "123"

		s.EXPECT().CategoriesNew(mock.Anything, factories.USER_ID, "Name", "1", "ff00ff").Return(eID, nil)

		resp, err := api.CategoriesNew(apiCtx(t), categories_pb.ReqNew_builder{
			Name:  new("Name"),
			Color: new("ff00ff"),
			Icon:  new("1"),
		}.Build())
		require.NoError(t, err)

		assert.Equal(t, resp.GetID(), eID)
	})
}

func TestAPI_CategoriesUpdate(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		_, api := newAPI(t)
		_, err := api.CategoriesUpdate(apiCtx(t), categories_pb.Category_builder{
			Id:    new("123"),
			Name:  new(""),
			Color: new("zzzz!!!zzz"),
			Icon:  new("--"),
		}.Build())

		assertValidationErrFields(t, err, "name", "color", "icon")
	})

	t.Run("single", func(t *testing.T) {
		api := newAPIWithRealDB(t)
		id, err := factories.Store(t).CategoriesNew(t.Context(), factories.USER_ID, "Old", "Old", "Old")
		require.NoError(t, err)
		factories.CleanupRow(t, `categories`, id)

		_, err = api.CategoriesUpdate(apiCtx(t), categories_pb.Category_builder{
			Id:   new(id),
			Name: new("New Name!"),
		}.Build())
		require.NoError(t, err)

		resp, err := api.CategoriesList(apiCtx(t), &categories_pb.ReqList{})
		require.NoError(t, err)
		require.NotEmpty(t, resp.GetResult())
		require.Equal(t, id, resp.GetResult()[0].GetID(), "unexpected categories exist")

		newCat := resp.GetResult()[0]
		assert.Equal(t, "Old", newCat.GetIcon()) // other stuff didn't get updated
		assert.Equal(t, "New Name!", newCat.GetName())
	})

	t.Run("full", func(t *testing.T) {
		api := newAPIWithRealDB(t)
		id, err := factories.Store(t).CategoriesNew(t.Context(), factories.USER_ID, "Old", "Old", "Old")
		require.NoError(t, err)
		factories.CleanupRow(t, `categories`, id)

		_, err = api.CategoriesUpdate(apiCtx(t), categories_pb.Category_builder{
			Id:    new(id),
			Name:  new("New Name!"),
			Color: new("ff00ff"),
			Icon:  new("1"),
		}.Build())
		require.NoError(t, err)

		resp, err := api.CategoriesList(apiCtx(t), &categories_pb.ReqList{})
		require.NoError(t, err)
		require.NotEmpty(t, resp.GetResult())
		require.Equal(t, id, resp.GetResult()[0].GetID(), "unexpected categories exist")

		newCat := resp.GetResult()[0]
		assert.Equal(t, "New Name!", newCat.GetName())
		assert.Equal(t, "ff00ff", newCat.GetColor())
		assert.Equal(t, "1", newCat.GetIcon())
	})
}

func TestAPI_CategoriesDelete(t *testing.T) {
	t.Run("not_found", func(t *testing.T) {
		s, api := newAPI(t)

		tx := tutils.MockStoreTx(t, s)
		tx.EXPECT().MappingsDeleteForCategoryDelete(mock.Anything, new("cat")).Return(nil)
		tx.EXPECT().CategoriesDelete(mock.Anything, factories.USER_ID, "cat").Return(0, nil)

		_, err := api.CategoriesDelete(apiCtx(t), bank_svc_pb.ReqDelete_builder{
			Id: new("cat"),
		}.Build())

		tutils.RequireGRPCStatus(t, codes.NotFound, err)
	})
	t.Run("found", func(t *testing.T) {
		// I want to test db-specific behavior here, so real db
		api := newAPIWithRealDB(t)

		catID := factories.NewCategory(t)

		mapTwoID := factories.NewMapping(t, &data.Mapping{
			ResName:       new("Heheh :3"),
			ResCategoryID: new(catID),
		})

		mapExclusiveID := factories.NewMapping(t, &data.Mapping{
			ResCategoryID: new(catID),
		})

		transID := "trans!"

		factories.NewTrans(t, []*store.TransactionsInsertParams{
			{
				ID:               transID,
				ResolvedCategory: new(catID),
			},
		})

		_, err := api.CategoriesDelete(apiCtx(t), bank_svc_pb.ReqDelete_builder{Id: new(catID)}.Build())
		require.NoError(t, err)

		s := factories.Store(t)

		e, err := s.MappingsExists(t.Context(), factories.USER_ID, mapExclusiveID)
		require.NoError(t, err)

		assert.False(t, e, "the mapping with just categoryID didn't get deleted")

		m2, err := s.MappingGetByID(t.Context(), factories.USER_ID, mapTwoID)
		require.NoError(t, err)

		if assert.NotNil(t, m2, "The mapping that had both resolved fields got deleted") {
			assert.Nil(t, m2.ResCategoryID, "the category id for the mapping with 2 resolved fields didn't get nulled")
			assert.NotNil(t, m2.ResName, "the name for the mapping with 2 resolved fields DID get nulled")
		}

		var transResCat *string
		err = factories.DB(t).QueryRow(t.Context(), `SELECT resolved_category FROM transactions WHERE id = $1`, transID).Scan(&transResCat)
		require.NoError(t, err)

		assert.Nil(t, transResCat, "The resolved category on a transaction didn't get nulled")
	})
}
