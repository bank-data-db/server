package internal_test

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/shadiestgoat/bankDataDB/bank_parser"
	"github.com/shadiestgoat/bankDataDB/data"
	"github.com/shadiestgoat/bankDataDB/db/store"
	"github.com/shadiestgoat/bankDataDB/tutils"
	"github.com/shadiestgoat/bankDataDB/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func findTransactionRowByDesc(rows [][]any, desc string) []any {
	for _, row := range rows {
		if strings.TrimSpace(row[4].(string)) == desc {
			return row
		}
	}

	return nil
}

// Assert that row with desc [desc] exists. Returns nil if it does not exist
func assertTransByDesc(t *testing.T, rows [][]any, desc string) []any {
	trans := findTransactionRowByDesc(rows, desc)

	assert.NotNil(t, trans, "Transaction "+desc+" should exist")

	return trans
}

func findTransMapByDesc(transRows, transMapsRows [][]any, desc string) []any {
	trans := findTransactionRowByDesc(transRows, desc)
	if trans == nil {
		return nil
	}

	for _, row := range transMapsRows {
		if row[0].(string) == trans[0].(string) {
			return row
		}
	}

	return nil
}

func assertTransMapByDesc(t *testing.T, transRows, transMapsRows [][]any, desc string) []any {
	row := findTransMapByDesc(transRows, transMapsRows, desc)

	assert.NotNil(t, row, "Mapped Transaction for "+desc+" should exist")

	return row
}

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestUploadBankIter(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		// trans 1 - exists
		// trans 2 - mapped using desc
		// trans 3 - mapped using desc & amount
		transactions := []*bank_parser.Transaction{
			{SettledAt: date(2025, 8, 10), AuthedAt: date(2025, 8, 10), Description: "ABC", Amt: -1.29, AmtAfterTransaction: new(15_419.44)},
			{SettledAt: date(2025, 8, 10), AuthedAt: date(2025, 8, 10), Description: "DEF", Amt: -10.79, AmtAfterTransaction: new(15_420.73)},
			{SettledAt: date(2025, 8, 8), AuthedAt: date(2025, 8, 06), Description: "Ghi", Amt: -42.17, AmtAfterTransaction: new(15_431.52)},
			{SettledAt: date(2025, 8, 07), AuthedAt: date(2025, 8, 06), Description: "Jkl", Amt: -0.99, AmtAfterTransaction: new(15_473.69)},
			{SettledAt: date(2025, 8, 07), AuthedAt: date(2025, 8, 05), Description: "MNO", Amt: -3.52, AmtAfterTransaction: new(15_474.68)},
			{SettledAt: date(2025, 8, 06), AuthedAt: date(2025, 8, 06), Description: "PQR", Amt: -1_400, AmtAfterTransaction: new(15_478.20)},
			{SettledAt: date(2025, 8, 06), AuthedAt: date(2025, 8, 06), Description: "STU", Amt: 700.00, AmtAfterTransaction: new(16_878.20)},
			{SettledAt: date(2025, 8, 06), AuthedAt: date(2025, 8, 05), Description: "VXY", Amt: -4.90, AmtAfterTransaction: new(16_178.20)},
			{SettledAt: date(2025, 8, 06), AuthedAt: date(2025, 8, 06), Description: "ZAB", Amt: -35.49, AmtAfterTransaction: new(16_183.10)},
		}
		api, s := tutils.NewMockAPI(t)

		s.EXPECT().MappingGetAll(mock.Anything, USER_ID).Return([]*data.Mapping{
			{
				InpText: (*data.MarshallableRegexp)(regexp.MustCompilePOSIX("^PQR$")),
				ResName: utils.Ptr("The PQR Transaction"),
			},
			{
				InpText: (*data.MarshallableRegexp)(regexp.MustCompilePOSIX("X")),
				ResName: utils.Ptr("The VXY Transaction"),
			},
			{
				InpAmt:        utils.Ptr(700.0),
				ResCategoryID: utils.Ptr("catID STU"),
			},
			{
				InpAmt:        utils.Ptr(-1.0),
				ResCategoryID: utils.Ptr("None"),
			},
		}, nil)

		// Pretend a transaction exists & the rest don't
		s.EXPECT().DoesTransactionExist(
			mock.Anything,
			USER_ID,
			date(2025, 8, 5), date(2025, 8, 7),
			"MNO", -3.52,
		).Return(true, nil)
		s.EXPECT().DoesTransactionExist(
			mock.Anything,
			mock.Anything,
			mock.Anything, mock.Anything,
			mock.Anything, mock.Anything,
		).Return(false, nil)

		s.EXPECT().InsertCheckpoint(mock.Anything, date(2025, 8, 10), 15_419.44)
		s.EXPECT().InsertCheckpoint(mock.Anything, date(2025, 8, 8), 15_431.52)
		s.EXPECT().InsertCheckpoint(mock.Anything, date(2025, 8, 7), 15_473.69)
		s.EXPECT().InsertCheckpoint(mock.Anything, date(2025, 8, 6), 15_478.20)

		tx := tutils.MockStoreTx(t, s)

		// Expect checkpoints to be sent out
		tx.EXPECT().SendBatch(mock.Anything, mock.Anything).Return(nil)

		transactionRows := [][]any{}
		tx.EXPECT().InsertTransactions(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, v *store.TransactionBatch) (int64, error) {
			transactionRows = v.Rows
			return int64(len(v.Rows)), nil
		})

		// trans_id, mapping_id, updated_name
		transMapRows := [][]any{}

		tx.EXPECT().TransMapsInsertBatch(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, v *store.TransMapsBatch) error {
			transMapRows = v.Rows
			return nil
		})

		resp, err := api.UploadBankIter(t.Context(), slices.Values(transactions), USER_ID)
		require.NoError(t, err)

		// Assert all the different transactions correctly being inserted & mapped
		assert.Equal(t, 8, resp.NewTransactions)
		assert.Equal(t, 1, resp.SkippedTransactions)
		assert.Equal(t, 5, resp.UnmappedTransactions)

		// already exists - don't enter it again
		assert.Nil(t, findTransactionRowByDesc(transactionRows, "MNO"))

		// Full match (w/ padding)
		if trans := assertTransByDesc(t, transactionRows, "PQR"); trans != nil {
			if assert.NotNil(t, trans[6], "PQR must have a resolved name") {
				assert.Equal(t, *trans[6].(*string), "The PQR Transaction")
			}

			assertTransMapByDesc(t, transactionRows, transMapRows, "PQR")
		}
		// Partial name match
		if trans := assertTransByDesc(t, transactionRows, "VXY"); trans != nil {
			if assert.NotNil(t, trans[6], "VXY must have a resolved name") {
				assert.Equal(t, *trans[6].(*string), "The VXY Transaction")
			}

			assertTransMapByDesc(t, transactionRows, transMapRows, "VXY")
		}
		// amount match
		if trans := assertTransByDesc(t, transactionRows, "STU"); trans != nil {
			if assert.NotNil(t, trans[7], "STU must have a resolved category") {
				assert.Equal(t, *trans[7].(*string), "catID STU")
			}

			assertTransMapByDesc(t, transactionRows, transMapRows, "STU")
		}
	})
}
