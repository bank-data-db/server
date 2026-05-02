package db

import "github.com/shadiestgoat/bankDataDB/pb/mappings"

// fuck your 'real' postgres enums
const (
	E_AMT_EXACT rune = '='
	E_AMT_GT    rune = '>'
	E_AMT_GTE   rune = 'g'
	E_AMT_LT    rune = '<'
	E_AMT_LTE   rune = 'l'
)

var (
	EnumAmtMatcherTranslation = map[rune]mappings.AmountMatchMode{
		E_AMT_EXACT: mappings.AmountMatchModeExact,
		E_AMT_GT:    mappings.AmountMatchModeGt,
		E_AMT_GTE:   mappings.AmountMatchModeGte,
		E_AMT_LT:    mappings.AmountMatchModeLt,
		E_AMT_LTE:   mappings.AmountMatchModeLte,
	}
)
