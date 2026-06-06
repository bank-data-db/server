package store

import (
	"context"

	"github.com/bank-data-db/server/data"
	"github.com/bank-data-db/server/db"
	"github.com/bank-data-db/server/snownode"
)

func (s DBStore) CardsNew(ctx context.Context, userID string, name string) (string, error) {
	id := snownode.NewID()
	_, err := s.db.Exec(
		ctx,
		`INSERT INTO cards (id, user_id, name) VALUES ($1, $2, $3)`,
		id, userID, name,
	)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (s DBStore) CategoriesNew(ctx context.Context, authorID string, name string, icon string, color string) (string, error) {
	id := snownode.NewID()
	_, err := s.db.Exec(
		ctx,
		`INSERT INTO categories (id, author_id, name, icon, color) VALUES ($1, $2, $3, $4, $5)`,
		id, authorID, name, icon, color,
	)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (s *DBStore) MappingNew(ctx context.Context, authorID string, m *data.Mapping) (string, error) {
	id := snownode.NewID()
	var amtMatcher *string
	if m.InpAmtMatcher != nil {
		amtMatcher = new(db.EnumAmtMatcherTranslationOther[*m.InpAmtMatcher])
	}

	_, err := s.db.Exec(
		ctx,
		`INSERT INTO mappings (
			id, author_id,
			name,
			match_text, match_card_id,
			match_amount, match_amount_matcher,
			res_name, res_category,
			priority
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		id, authorID,
		m.Name,
		m.InpTextOrNil(), m.InpCardID,
		m.InpAmt, amtMatcher,
		m.ResName, m.ResCategoryID,
		m.Priority,
	)
	if err != nil {
		return "", err
	}

	return id, nil
}
