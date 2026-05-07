package internal

import (
	"github.com/shadiestgoat/bankDataDB/data"
)

type MappingRes struct {
	Res       string
	MappingID string
}

func (m *MappingRes) SafeValue() *string {
	if m == nil {
		return nil
	}
	return &m.Res
}

// Map a specific transaction. all MUST be ordered by priority
func MapSpecificTransaction(all []*data.Mapping, amount float64, desc string, cardID string) (name *MappingRes, cat *MappingRes) {
	for _, m := range all {
		if m.Matches(amount, desc, cardID) {
			applyMatchResult(&name, m.ResName, m.ID)
			applyMatchResult(&cat, m.ResCategoryID, m.ID)
		}

		if name != nil && cat != nil {
			return
		}
	}

	return
}

func applyMatchResult(dst **MappingRes, src *string, mappingID string) {
	if *dst == nil && src != nil {
		*dst = &MappingRes{
			Res:       *src,
			MappingID: mappingID,
		}
	}
}
