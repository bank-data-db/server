package db

import "github.com/bank-data-db/proto/mappings_pb"

// fuck your 'real' postgres enums
const (
	E_AMT_EXACT rune = '='
	E_AMT_GT    rune = '>'
	E_AMT_GTE   rune = 'g'
	E_AMT_LT    rune = '<'
	E_AMT_LTE   rune = 'l'
)

var (
	EnumAmtMatcherTranslation = map[rune]mappings_pb.AmountMatchMode{
		E_AMT_EXACT: mappings_pb.AmountMatchModeExact,
		E_AMT_GT:    mappings_pb.AmountMatchModeGt,
		E_AMT_GTE:   mappings_pb.AmountMatchModeGte,
		E_AMT_LT:    mappings_pb.AmountMatchModeLt,
		E_AMT_LTE:   mappings_pb.AmountMatchModeLte,
	}
	EnumAmtMatcherTranslationOther = map[mappings_pb.AmountMatchMode]rune{}
)

func init() {
	for k, v := range EnumAmtMatcherTranslation {
		EnumAmtMatcherTranslationOther[v] = k
	}
}
