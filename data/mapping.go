package data

import (
	"log/slog"
	"regexp"

	"github.com/shadiestgoat/bankDataDB/pb/mappings"
)

type Mapping struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`

	InpText       *regexp.Regexp `json:"inputText,omitempty"`
	InpAmtMatcher *mappings.AmountMatchMode
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
			slog.Warn("Somehow received a non-nil amount but nil amt matcher!")
		} else {
			matchAmt := *m.InpAmt

			switch *m.InpAmtMatcher {
			case mappings.AmountMatchModeExact:
				return matchAmt == amount
			case mappings.AmountMatchModeGt:
				return matchAmt > amount
			case mappings.AmountMatchModeGte:
				return matchAmt >= amount
			case mappings.AmountMatchModeLt:
				return matchAmt < amount
			case mappings.AmountMatchModeLte:
				return matchAmt <= amount
			}
		}
	}

	// Would rather break shit loudly than quietly
	panic("impossible condition!")
}
