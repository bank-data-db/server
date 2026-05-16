package store_test

import (
	"regexp"
	"testing"

	"github.com/bank-data-db/proto/mappings_pb"
	"github.com/bank-data-db/server/data"
	"github.com/stretchr/testify/require"
)

func TestMappingNew(t *testing.T) {
	s := newStore(t)

	catID := newTestCat(t)

	m := &data.Mapping{
		Name:          "Mapping Name",
		InpText:       regexp.MustCompilePOSIX("abc.+"),
		InpAmtMatcher: new(mappings_pb.AmountMatchModeExact),
		InpAmt:        new(1.1),
		InpCardID:     new(CARD_ID),
		ResName:       new("Yahoo"),
		ResCategoryID: new(catID),
		Priority:      99,
	}

	// We aren't using the util function because I want to test MappingNew
	// directly and impl of the util might change in the future
	id, err := s.MappingNew(t.Context(), USER_ID, m)
	require.NoError(t, err)
	m.ID = id // so that comparison works lmao
	cleanupRow(t, `mappings`, id)

	m2, err := s.MappingGetByID(t.Context(), USER_ID, id)
	require.NoError(t, err)

	require.Equal(t, m, m2)
}
