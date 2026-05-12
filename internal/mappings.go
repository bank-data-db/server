package internal

import (
	"context"

	"github.com/shadiestgoat/bankDataDB/data"
	"github.com/shadiestgoat/bankDataDB/db/store"
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

// Maps a mapping onto existing transactions. Note that ID MUST exist already
// This uses a transaction under the hood
func MapAllTransactions(ctx context.Context, s store.Store, m *data.Mapping, userID string) (names uint32, categories uint32, err error) {
	err = s.TxFunc(ctx, func(s store.Store) error {
		if m.ResCategoryID != nil {
			c, err := s.TransactionsMapsMapExisting(ctx, false, userID, m)
			if err != nil {
				return err
			}
			names = uint32(c)
		}
		if m.ResName != nil {
			c, err := s.TransactionsMapsMapExisting(ctx, true, userID, m)
			if err != nil {
				return err
			}
			categories = uint32(c)
		}
		return nil
	})

	return
}

func UnmapForMapping(ctx context.Context, s store.Store, m *data.Mapping) error {
	// we need to unmap all transactions, then run the mapper again
	return s.TransactionsUnmapForMappingID(ctx, m.ID, m.ResName != nil, m.ResCategoryID != nil)
}
