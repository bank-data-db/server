package store

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/shadiestgoat/bankDataDB/data"
	"github.com/shadiestgoat/bankDataDB/db"
	"github.com/shadiestgoat/bankDataDB/pb/mappings"
)

func (s *DBStore) MappingGetAll(ctx context.Context, authorID string) ([]*data.Mapping, error) {
	rows, err := s.db.Query(
		ctx,
		`SELECT`+sel_cols+`FROM mappings WHERE author_id = $1 ORDER BY priority DESC`,
		authorID,
	)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*data.Mapping, error) {
		return scanMappingRow(row)
	})
}

func (s *DBStore) MappingGetByID(ctx context.Context, authorID, mappingID string) (*data.Mapping, error) {
	row := s.db.QueryRow(
		ctx,
		`SELECT`+sel_cols+`FROM mappings WHERE authed_id = $1 AND id = $2`,
		authorID, mappingID,
	)

	m, err := scanMappingRow(row)
	if err != nil {
		if db.NoRows(err) {
			return nil, nil
		}
		return nil, err
	}

	return m, nil
}

const sel_cols = `
id, name, priority,
match_text, match_card_id
match_amount, match_amount_matcher
res_name, res_category
`

func scanMappingRow(row interface{ Scan(dest ...any) error }) (*data.Mapping, error) {
	mapping := &data.Mapping{}
	var rawRegex *string
	var rawAmtMatcher *rune

	err := row.Scan(
		&mapping.ID, &mapping.Name, &mapping.Priority,
		&rawRegex, &mapping.InpCardID,
		&mapping.InpAmt, &rawAmtMatcher,
		&mapping.ResName, &mapping.ResCategoryID,
	)
	if err != nil {
		return nil, err
	}

	if rawRegex != nil {
		reg, err := regexp.CompilePOSIX(*rawRegex)
		if err != nil {
			slog.Error("Somehow received bad regex from DB!", "mapping_id", mapping.ID)
			return nil, err
		} else {
			mapping.InpText = reg
		}
	}

	if rawAmtMatcher != nil {
		res, ok := db.EnumAmtMatcherTranslation[*rawAmtMatcher]
		if !ok {
			slog.Error("Unknown enum in DB!", "mapping_id", mapping.ID, "value", *rawAmtMatcher)
			return nil, fmt.Errorf("bad db value")
		}

		mapping.InpAmtMatcher = &res
	}

	return mapping, nil
}

// Map existing transactions based on a mapping, inserting mapped_transactions values as well
func (s *DBStore) TransactionsMapsMapExisting(ctx context.Context, updateName bool, authorID string, m *data.Mapping) (int, error) {
	// TODO: In the future, maybe we can limit this to X rows per chunk or smt?
	args := pgx.NamedArgs{
		"author_id":  authorID,
		"priority":   m.Priority,
		"match_name": updateName,
		"mapping_id": m.ID,
	}
	if updateName {
		args["new_val"] = m.ResName
	} else {
		args["new_val"] = m.ResCategoryID
	}

	conditions := []string{}

	if m.InpAmt != nil && m.InpAmtMatcher != nil {
		args["amt"] = *m.InpAmt
		switch *m.InpAmtMatcher {
		case mappings.AmountMatchModeExact:
			conditions = append(conditions, "amount = @amt")
		case mappings.AmountMatchModeGt:
			conditions = append(conditions, "amount > @amt")
		case mappings.AmountMatchModeGte:
			conditions = append(conditions, "amount >= @amt")
		case mappings.AmountMatchModeLt:
			conditions = append(conditions, "amount < @amt")
		case mappings.AmountMatchModeLte:
			conditions = append(conditions, "amount <= @amt")
		}
	}
	if m.InpText != nil {
		conditions = append(conditions, "description ~ @desc")
		args["desc"] = m.InpText.String()
	}
	if m.InpCardID != nil {
		conditions = append(conditions, "card_id = @card_id")
		args["card_id"] = *m.InpCardID
	}

	col := "resolved_category"
	if updateName {
		col = "resolved_name"
	}

	res, err := s.db.Exec(
		ctx,
		fmt.Sprintf(
			`
			WITH eligible AS (
				SELECT t.id, mapping_id
				FROM transactions AS t
				LEFT JOIN mapped_transactions ON t.id = trans_id AND updated_name = @match_name
				LEFT JOIN mappings AS m ON m.id = mapping_id
				WHERE
					t.author_id = @author_id
						AND
					-- is mapped or not manually overridden
					(priority IS NOT NULL OR %s IS NULL)
						AND
					-- is not mapped or mapping has lower priority
					(priority IS NULL OR priority < @priority)
					AND %s
			), deleted AS (
				DELETE FROM mapped_transactions mp
				USING eligible e
				WHERE mp.mapping_id = e.mapping_id AND updated_name = @match_name
			), updated AS (
				UPDATE transactions
				SET %s = @new_val
				FROM eligible
				WHERE transactions.id = eligible.id
				RETURNING transactions.id
			) INSERT INTO mapped_transactions (trans_id, mapping_id, updated_name)
				SELECT id AS trans_id, @mapping_id AS mapping_id, @match_name AS updated_name FROM updated
			`, col, col, strings.Join(conditions, " AND "),
		),
		args,
	)
	if err != nil {
		return 0, err
	}

	return int(res.RowsAffected()), nil
}
