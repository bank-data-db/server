package bank_data

import (
	"context"
	"log/slog"

	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v5"
	"github.com/shadiestgoat/bankDataDB/db"
	"github.com/shadiestgoat/bankDataDB/db/store"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data/lerrors"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data/paginator"
	"github.com/shadiestgoat/bankDataDB/internal"
	"github.com/shadiestgoat/bankDataDB/pb/bank_svc"
	"github.com/shadiestgoat/bankDataDB/pb/mappings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// MappingDelete implements [svc.BankDataServer].
func (a *API) MappingDelete(ctx context.Context, req *mappings.ReqDelete) (*mappings.RespDelete, error) {
	if req.GetID() == "" {
		return nil, lerrors.ErrIDRequired
	}

	// false is default
	if req.GetOrphanTransactions() {
		c, err := a.store.MappingsDeleteKeepingOrphans(ctx, userID(ctx), req.GetID())
		if err != nil {
			return nil, lerrors.ErrDB
		}
		if c == 0 {
			return nil, status.Error(codes.NotFound, "mapping not found")
		}

		v := &mappings.RespDelete{}
		v.SetAffectedTransactions(0)
		v.SetRemappedTransactions(0)
		return v, nil
	}

	var (
		transCount uint32
		remapped   uint32
	)

	err := a.store.TxFunc(ctx, func(s store.Store) error {
		exists, err := s.MappingsExists(ctx, userID(ctx), req.GetID())
		if err != nil {
			return lerrors.ErrDB
		}
		if !exists {
			return status.Error(codes.NotFound, "mapping not found")
		}

		trans, err := s.MappingsDeleteNoOrphans(ctx)
		if err != nil {
			return err
		}
		transCount = uint32(len(trans))
		if len(trans) == 0 {
			return nil
		}

		m, err := s.MappingGetAll(ctx, userID(ctx))
		if err != nil {
			return nil
		}

		b := &pgx.Batch{}

		for _, t := range trans {
			resName, resCat := internal.MapSpecificTransaction(m, t.Amount.InexactFloat64(), t.Description, t.CardID)
			// We gotta update ONLY fields that got affected by the deleted thingy

			var dbResName, dbResCat **string
			if t.UpCat {
				v := resCat.SafeValue()
				dbResCat = &v
				if resCat != nil {
					a.store.BatchInsertTransMapping(b, t.ID, resCat.MappingID, false)
				}
			}
			if t.UpName {
				v := resName.SafeValue()
				dbResName = &v
				if resName != nil {
					a.store.BatchInsertTransMapping(b, t.ID, resName.MappingID, true)
				}
			}

			if resName != nil || resCat != nil {
				remapped++
			}

			a.store.BatchForceUpdateTrans(b, t.ID, dbResName, dbResCat)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return mappings.RespDelete_builder{AffectedTransactions: new(transCount), RemappedTransactions: new(remapped)}.Build(), nil
}

var paginatorMappings = &paginator.ConfEasy[*mappings.ReqList, *mappings.Mapping, *mappings.RespList]{
	PageSizeMax:     100,
	PageSizeDefault: 75,
	CollectRow: func(row pgx.CollectableRow) (*mappings.Mapping, error) {
		v := mappings.Mapping_builder{}
		var amtMatcher *rune

		if err := row.Scan(
			&v.Id, &v.Name,
			&v.MatchText, &v.MatchAmount, &amtMatcher, &v.MatchCardId,
			&v.ResultName, &v.ResultCategoryId,
			&v.Priority,
		); err != nil {
			return nil, err
		}

		if amtMatcher != nil {
			switch *amtMatcher {
			case db.E_AMT_EXACT:
				v.MatchAmountMode = new(mappings.AmountMatchModeExact)
			case db.E_AMT_GT:
				v.MatchAmountMode = new(mappings.AmountMatchModeGt)
			case db.E_AMT_GTE:
				v.MatchAmountMode = new(mappings.AmountMatchModeGte)
			case db.E_AMT_LT:
				v.MatchAmountMode = new(mappings.AmountMatchModeLt)
			case db.E_AMT_LTE:
				v.MatchAmountMode = new(mappings.AmountMatchModeLte)
			default:
				slog.Warn("Unknown amount match mode stored in db!", "mode", *amtMatcher)
				// keep nil i guess
			}
		}

		return v.Build(), nil
	},
}

// MappingsList implements [svc.BankDataServer].
func (a *API) MappingsList(ctx context.Context, req *mappings.ReqList) (*mappings.RespList, error) {
	sb := sqlbuilder.NewSelectBuilder().From(
		"mappings",
	).Select(
		"id", "name",
		"match_text", "match_amount", "match_amount_matcher", "match_card_id",
		"res_name", "res_category",
		"priority",
	)

	sb.Where(sb.EQ("author_id", userID(ctx)))
	if req.HasCardId() {
		sb.Where(sb.EQ("card_id", req.GetCardID()))
	}

	resp := &mappings.RespList{}
	err := paginatorMappings.RunQuery(ctx, a.db, sb, req, resp)

	return resp, err
}

// MappingsNew implements [svc.BankDataServer].
func (a *API) MappingsNew(ctx context.Context, req *mappings.ReqNew) (*bank_svc.RespNew, error) {
	panic("unimplemented")
}

// MappingsUpdate implements [svc.BankDataServer].
func (a *API) MappingsUpdate(ctx context.Context, req *mappings.Mapping) (*emptypb.Empty, error) {
	panic("unimplemented")
}
