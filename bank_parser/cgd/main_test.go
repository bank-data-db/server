package cgd_test

import (
	"strings"
	"testing"
	"time"

	"github.com/bank-data-db/server/bank_parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 1. test portuguese vs english
// 	1. guess
// 	2. numeric
// 2. test csv vs tsv

func assertTransactions(t *testing.T, arr []*bank_parser.Transaction, data string) {
	iter, err := bank_parser.Iter(t.Context(), strings.NewReader(strings.TrimSpace(data)))
	require.NoError(t, err)

	i := 0
	for v := range iter {
		var e *bank_parser.Transaction
		if i < len(arr) {
			e = arr[i]
		}

		assert.Equal(t, e, v, "At index %d", i)
		i++
	}

	assert.Equal(t, len(arr), i, "Wrong # of transactions")
}

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestNewParser(t *testing.T) {
	t.Run("tsv", func(t *testing.T) {
		assertTransactions(t, []*bank_parser.Transaction{
			{
				AuthedAt:            date(2025, 8, 10),
				SettledAt:           date(2025, 8, 10),
				Description:         "[Diversos] ABC",
				Amt:                 -1.29,
				AmtAfterTransaction: new(15_419.44),
			},
			{
				AuthedAt:            date(2025, 8, 10),
				SettledAt:           date(2025, 8, 10),
				Description:         "[Diversos] DEF",
				Amt:                 -10.79,
				AmtAfterTransaction: new(15_420.73),
			},
		}, `
View current operations and balances - DATE HERE

Account 	ACCOUNT_ID - EUR - CAIXA ACCOUNT
Start Date 	SOME DATE
End Date 	SOME DATE

Op. Date 	Value Date 	Description 	Debit 	Credit 	Balance Accounting 	Balance available 	Categoria (EN) 	
10-08-2025	10-08-2025	ABC 	1.29		15,419.44	--	Diversos 	
10-08-2025	10-08-2025	DEF 	10.79		15,420.73	--	Diversos
`)
	})

	t.Run("csv", func(t *testing.T) {
		t.Run("portuguese", func(t *testing.T) {
			assertTransactions(t, []*bank_parser.Transaction{
				{
					AuthedAt:            date(2024, 6, 28),
					SettledAt:           date(2024, 6, 29),
					Description:         "[LEVANTAMENTOS] TEST",
					Amt:                 -0.02,
					AmtAfterTransaction: new(15_435.88),
				},
				{
					AuthedAt:            date(2024, 6, 28),
					SettledAt:           date(2024, 6, 29),
					Description:         "[LEVANTAMENTOS] TEST 2",
					Amt:                 -0.42,
					AmtAfterTransaction: new(15_435.90),
				},
			}, `
Consultar saldos e movimentos � ordem - DATE

Conta ;ID - EUR - ACCT 
Data de in�cio ;DATE
Data de fim ;DATE

Data mov. ;Data valor ;Descri��o ;D�bito ;Cr�dito ;Saldo contabil�stico ;Saldo dispon�vel ;Categoria ;
29-06-2024;28-06-2024;TEST ;0,02;;15.435,88;15.435,88;LEVANTAMENTOS ;
29-06-2024;28-06-2024;TEST 2 ;0,42;;15.435,90;15.435,90;LEVANTAMENTOS ;
`)
		})

		t.Run("english", func(t *testing.T) {
			assertTransactions(t, []*bank_parser.Transaction{
				{
					AuthedAt:            date(2025, 8, 10),
					SettledAt:           date(2025, 8, 10),
					Description:         "[Diversos] ABC",
					Amt:                 -1.29,
					AmtAfterTransaction: new(15_419.44),
				},
				{
					AuthedAt:            date(2025, 8, 10),
					SettledAt:           date(2025, 8, 10),
					Description:         "[Diversos] DEF",
					Amt:                 -10.79,
					AmtAfterTransaction: new(15_420.73),
				},
			}, `
View current operations and balances - DATE HERE

Account ;ACCOUNT_ID - EUR - CAIXA ACCOUNT
Start Date ;SOME DATE
End Date ;SOME DATE

Op. Date ;Value Date ;Description ;Debit ;Credit ;Balance Accounting ;Balance available ;Categoria (EN) ;
10-08-2025;10-08-2025;ABC ;1.29;;15,419.44;--;Diversos ;
10-08-2025;10-08-2025;DEF ;10.79;;15,420.73;--;Diversos
`)
		})
	})
}

func requireGuessedID(t *testing.T, data string, expectedID string) {
	bank_parser.GuessID(strings.NewReader(strings.TrimSpace(data)))
}

func TestGuesses(t *testing.T) {
	t.Run("english", func(t *testing.T) {
		requireGuessedID(t, `
View current operations and balances - DATE HERE

Account 	ACCOUNT_ID - EUR - CAIXA ACCOUNT
Start Date 	SOME DATE
End Date 	SOME DATE

Op. Date 	Value Date 	Description 	Debit 	Credit 	Balance Accounting 	Balance available 	Categoria (EN) 	
10-08-2025	10-08-2025	ABC 	1.29		15,419.44	--	Diversos 	
10-08-2025	10-08-2025	DEF 	10.79		15,420.73	--	Diversos
`, "cgd/en")
	})
	t.Run("portuguese", func(t *testing.T) {
		requireGuessedID(t, `
Consultar saldos e movimentos � ordem - DATE HERE

Conta ;ACCOUNT_ID - EUR - CAIXA ACCOUNT 
Data de in�cio ;DATE
Data de fim ;DATE
`, "cgd/pt")
	})
}
