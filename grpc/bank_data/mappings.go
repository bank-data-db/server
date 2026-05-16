package bank_data

import (
	"context"
	"log/slog"
	"regexp"

	"github.com/bank-data-db/proto/mappings_pb"
	"github.com/bank-data-db/server/data"
	"github.com/bank-data-db/server/db"
	"github.com/bank-data-db/server/db/store"
	"github.com/bank-data-db/server/grpc/bank_data/lerrors"
	"github.com/bank-data-db/server/grpc/bank_data/paginator"
	"github.com/bank-data-db/server/grpc/bank_data/validator"
	"github.com/bank-data-db/server/internal"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/emptypb"
)

// MappingDelete implements [svc.BankDataServer].
func (a *API) MappingDelete(ctx context.Context, req *mappings_pb.ReqDelete) (*mappings_pb.RespDelete, error) {
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

		v := &mappings_pb.RespDelete{}
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

	return mappings_pb.RespDelete_builder{AffectedTransactions: new(transCount), RemappedTransactions: new(remapped)}.Build(), nil
}

var paginatorMappings = &paginator.ConfEasy[*mappings_pb.ReqList, *mappings_pb.Mapping, *mappings_pb.RespList]{
	PageSizeMax:     100,
	PageSizeDefault: 75,
	CollectRow: func(row pgx.CollectableRow) (*mappings_pb.Mapping, error) {
		v := mappings_pb.Mapping_builder{}
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
				v.MatchAmountMode = new(mappings_pb.AmountMatchModeExact)
			case db.E_AMT_GT:
				v.MatchAmountMode = new(mappings_pb.AmountMatchModeGt)
			case db.E_AMT_GTE:
				v.MatchAmountMode = new(mappings_pb.AmountMatchModeGte)
			case db.E_AMT_LT:
				v.MatchAmountMode = new(mappings_pb.AmountMatchModeLt)
			case db.E_AMT_LTE:
				v.MatchAmountMode = new(mappings_pb.AmountMatchModeLte)
			default:
				slog.Warn("Unknown amount match mode stored in db!", "mode", *amtMatcher) //nolint:sloglint
				// keep nil i guess
			}
		}

		return v.Build(), nil
	},
}

// MappingsList implements [svc.BankDataServer].
func (a *API) MappingsList(ctx context.Context, req *mappings_pb.ReqList) (*mappings_pb.RespList, error) {
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

	resp := &mappings_pb.RespList{}
	err := paginatorMappings.RunQuery(ctx, a.db, sb, req, resp)

	return resp, err
}

func validateTransName(v protoreflect.Value) *string {
	n := v.String()
	if len(n) < 2 {
		return new("Name is too short")
	}
	return nil
}

var validatorMapping = &validator.Validator{
	Validations: []validator.Validation{
		validator.NewFieldValidation(`name`, true, validateTransName),
		validator.NewMessageValidation(
			[]string{"match_text", "match_amount", "match_card_id"},
			func(msg *mappings_pb.Mapping) *string {
				if !msg.HasId() {
					return nil
				}
				if !msg.HasMatchText() && !msg.HasMatchAmount() && !msg.HasMatchCardId() {
					return new("at least one matcher is required")
				}
				return nil
			},
		),
		validator.NewMessageValidation(
			[]string{"result_category_id", "result_name"},
			func(msg *mappings_pb.Mapping) *string {
				if !msg.HasId() {
					return nil
				}

				if !msg.HasResultCategoryId() && !msg.HasResultName() {
					return new("at least one result is required")
				}
				return nil
			},
		),
		validator.NewMessageValidation(
			[]string{"match_amount_mode", "match_amount"},
			func(msg *mappings_pb.Mapping) *string {
				if msg.HasMatchAmount() != msg.HasMatchAmountMode() {
					return new("to specify amount, you must specify both mode and number")
				}
				return nil
			},
		),
		validator.NewFieldValidation(
			"match_text", false,
			func(prv protoreflect.Value) *string {
				t := prv.String()
				_, err := regexp.CompilePOSIX(t)
				if err != nil {
					return new("Regex Compile Error: " + err.Error())
				}

				return nil
			},
		),
	},
}

// MappingsNew implements [svc.BankDataServer].
func (a *API) MappingsNew(ctx context.Context, req *mappings_pb.ReqNew) (*mappings_pb.RespNew, error) {
	if err := validatorMapping.Validate(req); err != nil {
		return nil, err
	}

	var resp = &mappings_pb.RespNew{}

	err := a.store.TxFunc(ctx, func(s store.Store) error {
		m := &data.Mapping{
			Name:     req.GetName(),
			Priority: int(req.GetPriority()),

			ResName:       new(string),
			ResCategoryID: new(string),
		}

		if req.HasMatchText() {
			// alr validated as valid regex
			m.InpText = regexp.MustCompilePOSIX(req.GetMatchText())
		}
		if req.HasMatchAmount() && req.HasMatchAmountMode() {
			m.InpAmt = new(req.GetMatchAmount())
			m.InpAmtMatcher = new(req.GetMatchAmountMode())
		}
		if req.HasMatchCardId() {
			m.InpCardID = new(req.GetMatchCardID())
		}
		if req.HasResultCategoryId() {
			m.ResCategoryID = new(req.GetResultCategoryID())
		}
		if req.HasResultName() {
			m.ResName = new(req.GetResultName())
		}

		id, err := s.MappingNew(ctx, userID(ctx), m)
		if err != nil {
			return err
		}

		m.ID = id

		mappedCats, mappedNames, err := internal.MapAllTransactions(ctx, s, m, userID(ctx))
		if err != nil {
			return err
		}

		resp.SetMappedCategories(mappedCats)
		resp.SetMappedNames(mappedNames)

		// in the same tx bc if this for SOME REASON fails, we don't want to have committed work w/ a failed request
		total, err := s.MappingsTransactionCount(ctx, m.ID)
		if err != nil {
			return err
		}
		resp.SetMappedTransactions(uint32(total))

		return nil
	})
	if err != nil {
		return nil, lerrors.ErrDB
	}

	return resp, nil
}

// MappingsUpdate implements [svc.BankDataServer].
func (a *API) MappingsUpdate(ctx context.Context, req *mappings_pb.ReqUpdate) (*emptypb.Empty, error) {
	if err := validatorMapping.Validate(req); err != nil {
		return nil, err
	}

	m, err := a.store.MappingGetByID(ctx, userID(ctx), req.GetID())
	if err != nil {
		return nil, lerrors.ErrDB
	}
	if m == nil {
		return nil, status.Error(codes.NotFound, "mapping not found")
	}

	sb := sqlbuilder.Update("mappings")
	sb.Where(sb.EQ("id", req.GetID()))

	// we gotta do 2 things: patch the mapping row AND retroactively update transactions mapped using this shit

	// prep the patcher
	matchersChanged := req.HasMatchText() || req.HasPriority() ||
		patchMappingFieldPatcher(sb, "match_amount", &m.InpAmt, req.HasMatchAmount, req.GetMatchAmount) ||
		patchMappingFieldPatcher(sb, "match_amount_matcher", &m.InpAmtMatcher, req.HasMatchAmountMode, req.GetMatchAmountMode) ||
		patchMappingFieldString(sb, "match_card_id", &m.InpCardID, req.HasMatchCardId, req.GetMatchCardID)

	if req.HasMatchText() {
		// EYEROLL
		if req.GetMatchText() == "" {
			sb.SetMore(sb.Assign("match_text", nil))
			m.InpText = nil
		} else {
			m.InpText = regexp.MustCompilePOSIX(req.GetMatchText())
			sb.SetMore(sb.Assign("match_text", req.GetMatchText()))
		}
	}

	if req.HasName() {
		// name is validated to not be empty
		sb.SetMore(sb.Assign("name", req.GetName()))
		m.Name = req.GetName()
	}
	if req.HasPriority() {
		sb.SetMore(sb.Assign("priority", req.GetPriority()))
		m.Priority = int(req.GetPriority())
	}

	resNameChanged := patchMappingFieldString(sb, "res_name", &m.ResName, req.HasResultName, req.GetResultName)
	resCatChanged := patchMappingFieldString(sb, "res_category", &m.ResCategoryID, req.HasResultCategoryId, req.GetResultCategoryID)

	// Now we gotta re-validate that we didn't mess anything up
	if err := validatorMapping.Validate(mappings_pb.ReqNew_builder{
		Name:             &m.Name,
		ResultCategoryId: m.ResCategoryID,
		ResultName:       m.ResName,
		MatchText:        m.InpTextOrNil(),
		MatchAmountMode:  m.InpAmtMatcher,
		MatchAmount:      m.InpAmt,
		MatchCardId:      m.InpCardID,
		Priority:         new(int32(m.Priority)),
	}.Build()); err != nil {
		return nil, err
	}

	// Finally, ONE LAST VALIDATION
	if req.HasResultCategoryId() && m.ResCategoryID != nil {
		exists, err := a.store.CategoriesExists(ctx, *m.ResCategoryID, userID(ctx))
		if err != nil {
			return nil, lerrors.ErrDB
		}
		if !exists {
			return nil, status.Error(codes.InvalidArgument, "category does not exist")
		}
	}

	if sb.NumAssignment() == 0 {
		// No op
		return &emptypb.Empty{}, nil
	}

	// Ok NOW we need to do that actual smart stuff
	// the theory is that we need to retroactively fix everything
	// This means that if matchers changed, we need to re-map based on this mapping
	// But if matchers didn't change, then we are good to just update transactions connected to this mapping

	err = a.store.TxFunc(ctx, func(s store.Store) error {
		sql, args := sb.Build()
		t, err := s.GetDB().Exec(ctx, sql, args...)
		if err != nil {
			return err
		}
		if t.RowsAffected() == 0 {
			// One more pre-emptive check: if it got through
			// and we actually updated nothing, then we don't need to do the rest
			return nil
		}

		if matchersChanged {
			// we need to unmap all transactions, then run the mapper again
			if err := internal.UnmapForMapping(ctx, s, m); err != nil {
				return err
			}

			_, _, err := internal.MapAllTransactions(ctx, s, m, userID(ctx))
			if err != nil {
				return err
			}

			// We remapped everything, so names and stuff would have been updated
			return nil
		}

		if resNameChanged {
			if err := s.MappingsRemapExistingName(ctx, m.ID, m.ResName); err != nil {
				return err
			}
		}

		if resCatChanged {
			if err := s.MappingsRemapExistingCategoryID(ctx, m.ID, m.ResCategoryID); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, lerrors.ErrDB
	}

	return &emptypb.Empty{}, nil
}

type Patcher[T any] interface {
	HasDelete() bool
	GetDelete() bool
	HasValue() bool
	GetValue() T
}

func patchMappingFieldString(sb *sqlbuilder.UpdateBuilder, sqlCol string, d **string, has func() bool, get func() string) bool {
	return patchMappingField(sb, sqlCol, d, has, get, func() bool {
		v := get()
		return v == ""
	})
}

func patchMappingFieldPatcher[T mappings_pb.AmountMatchMode | float64, P Patcher[T]](sb *sqlbuilder.UpdateBuilder, sqlCol string, d **T, has func() bool, get func() P) bool {
	return patchMappingField(sb, sqlCol, d, func() bool {
		if !has() {
			return false
		}
		v := get()

		return v.HasValue() || v.HasDelete()
	}, func() T {
		return get().GetValue()
	}, func() bool {
		return get().GetDelete()
	})
}

// returns "needs patching"
func patchMappingField[T mappings_pb.AmountMatchMode | float64 | string](sb *sqlbuilder.UpdateBuilder, sqlCol string, d **T, has func() bool, get func() T, setNull func() bool) bool {
	if !has() {
		return false
	}

	v := get()
	if setNull() {
		if *d == nil {
			return false
		}
		*d = nil
		sb.SetMore(sb.Assign(sqlCol, nil))
		return true
	}

	curVal := *d
	if curVal != nil && *curVal == v {
		return false
	}

	*d = new(v)
	sb.SetMore(sb.Assign(sqlCol, v))
	return true
}
