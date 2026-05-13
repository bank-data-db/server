package bank_data_test

import (
	"context"
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/shadiestgoat/bankDataDB/db/store"
	"github.com/shadiestgoat/bankDataDB/pb/cards"
	"github.com/shadiestgoat/bankDataDB/tutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI_CardsList(t *testing.T) {
	db := tutils.DB(t)
	s := store.NewStore(db)

	t.Cleanup(func() {
		_, err := db.Exec(context.Background(), `DELETE FROM cards`)
		if err != nil {
			fmt.Println("Error when cleaning up cards in db", err)
		}
	})

	cardNames := []string{"card1", "card TWO!!", "card.... THREE"}
	cardIDs := make([]string, len(cardNames))
	for i, v := range cardNames {
		id, err := s.CardsNew(t.Context(), USER_ID, v)
		require.NoError(t, err)
		cardIDs[i] = id

		// Also fill in cards for not us. Just to make sure we don't mess ourselves up t-t
		_, err = s.CardsNew(t.Context(), USER_2_ID, v)
		require.NoError(t, err)
	}

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
				assert.EqualValues(t, resp.GetTotalCount(), len(cardNames), "the total amount is wrong")
				assert.LessOrEqual(t, len(resp.GetResult()), pageSize, "the result page is greater than page size")
	
				for _, c := range resp.GetResult() {
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
			assert.EqualValues(t, math.Ceil(float64(len(cardNames))/float64(pageSize)), fetches, "fetches have the wrong count")
		}
	}

	t.Run("page_size=1", testForSize(1))
	t.Run("page_size=2", testForSize(2))
}
