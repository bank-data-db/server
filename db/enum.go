package db

import "github.com/bank-data-db/proto/mappings_pb"

// fuck your 'real' postgres enums
// Also, strings bc otherwise pgx isnt happy

const (
	E_AMT_EXACT string = "="
	E_AMT_GT    string = ">"
	E_AMT_GTE   string = "g"
	E_AMT_LT    string = "<"
	E_AMT_LTE   string = "l"
)

var (
	EnumAmtMatcherTranslation = map[string]mappings_pb.AmountMatchMode{
		E_AMT_EXACT: mappings_pb.AmountMatchModeExact,
		E_AMT_GT:    mappings_pb.AmountMatchModeGt,
		E_AMT_GTE:   mappings_pb.AmountMatchModeGte,
		E_AMT_LT:    mappings_pb.AmountMatchModeLt,
		E_AMT_LTE:   mappings_pb.AmountMatchModeLte,
	}
	EnumAmtMatcherTranslationOther = map[mappings_pb.AmountMatchMode]string{}
)

func init() {
	for k, v := range EnumAmtMatcherTranslation {
		EnumAmtMatcherTranslationOther[v] = k
	}
}
