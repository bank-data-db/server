// DONT EDIT: Auto generated

package store

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/shadiestgoat/bankDataDB/data"
)

// Store ...
type Store interface {
	CardsDelete(ctx context.Context, userID string, iD string) (int64, error)
	CardsUpdate(ctx context.Context, userID string, iD string, name string) (int64, error)
	CategoriesDelete(ctx context.Context, authorID string, iD string) (int64, error)
	TransactionsDelete(ctx context.Context, authorID string, iD string) (int64, error)
	MappingsDeleteKeepingOrphans(ctx context.Context, authorID string, iD string) (int64, error)
	MappingsDeleteNoOrphans(ctx context.Context) ([]*MappingsDeleteNoOrphansRow, error)
	MappingsExists(ctx context.Context, authorID string, iD string) (bool, error)
	SendBatch(ctx context.Context, b *pgx.Batch) error
	TxFunc(ctx context.Context, h func(s Store) error) error
	MappingGetAll(ctx context.Context, authorID string) ([]*data.Mapping, error)
	MappingGetByID(ctx context.Context, authorID, mappingID string) (*data.Mapping, error)
	CardsNew(ctx context.Context, userID string, name string) (string, error)
	NewCategory(ctx context.Context, authorID string, name string, icon string, color string) (string, error)
}
