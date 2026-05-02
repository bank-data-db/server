package store

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/shadiestgoat/bankDataDB/data"
	"github.com/shadiestgoat/bankDataDB/db"
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
