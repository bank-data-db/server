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

func (m Mapping) Matches(tAmt float64, desc string, cardID string) bool {
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
			inpAmt := *m.InpAmt

			switch *m.InpAmtMatcher {
			case mappings_pb.AmountMatchModeExact:
				if inpAmt != tAmt {
					return false
				}
			case mappings_pb.AmountMatchModeGt:
				if inpAmt <= tAmt {
					return false
				}
			case mappings_pb.AmountMatchModeGte:
				if inpAmt < tAmt {
					return false
				}
			case mappings_pb.AmountMatchModeLt:
				if inpAmt >= tAmt {
					return false
				}
			case mappings_pb.AmountMatchModeLte:
				if inpAmt > tAmt {
					return false
				}
			}
		}
	}

	return true
}

func (m Mapping) InpTextOrNil() *string {
	if m.InpText == nil {
		return nil
	}

	return new(m.InpText.String())
}
