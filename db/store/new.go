package store

import (
	"context"

	"github.com/shadiestgoat/bankDataDB/snownode"
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

func (s DBStore) NewCategory(ctx context.Context, authorID string, name string, icon string, color string) (string, error) {
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
