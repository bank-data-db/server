package data

import (
	"log/slog"
	"regexp"

	"github.com/bank-data-db/proto/mappings_pb"
)

type Mapping struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`

	InpText       *regexp.Regexp `json:"inputText,omitempty"`
	InpAmtMatcher *mappings_pb.AmountMatchMode
	InpAmt        *float64 `json:"inputAmount,omitempty"`
	InpCardID     *string

	ResName       *string `json:"resName,omitempty"`
	ResCategoryID *string `json:"resCategoryID,omitempty"`

	Priority int `json:"priority"`
}

func (m Mapping) Matches(amount float64, desc string, cardID string) bool {
	if m.InpText != nil && !m.InpText.MatchString(desc) {
		return false
	}
	if m.InpCardID != nil && *m.InpCardID != cardID {
		return false
	}

	if m.InpAmt != nil {
		if m.InpAmtMatcher == nil {
			slog.Warn("Somehow received a non-nil amount but nil amt matcher!") //nolint:sloglint
		} else {
			matchAmt := *m.InpAmt

			switch *m.InpAmtMatcher {
			case mappings_pb.AmountMatchModeExact:
				return matchAmt == amount
			case mappings_pb.AmountMatchModeGt:
				return matchAmt > amount
			case mappings_pb.AmountMatchModeGte:
				return matchAmt >= amount
			case mappings_pb.AmountMatchModeLt:
				return matchAmt < amount
			case mappings_pb.AmountMatchModeLte:
				return matchAmt <= amount
			}
		}
	}

	// Would rather break shit loudly than quietly
	panic("impossible condition!")
}

func (m Mapping) InpTextOrNil() *string {
	if m.InpText == nil {
		return nil
	}

	return new(m.InpText.String())
}
