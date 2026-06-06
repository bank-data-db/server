// DONT EDIT: Auto generated

package store

import (
	"context"
	"time"

	"github.com/bank-data-db/server/data"
	"github.com/bank-data-db/server/db"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// Store ...
type Store interface {
	BatchForceUpdateTrans(batch *pgx.Batch, id string, name, catID **string)
	BatchInsertTransMapping(batch *pgx.Batch, transID, mappingID string, updatesName bool)
	BatchCheckpointsNew(batch *pgx.Batch, cardID string, date time.Time, amt float64)
	BatchMappedTransactionDeleteNoMappingID(batch *pgx.Batch, transID string, name bool)
	UserByName(ctx context.Context, username string) (*UserByNameRow, error)
	UserUpdatedAt(ctx context.Context, id string) (time.Time, error)
	MappedTransactionsInsert(ctx context.Context, arg []*MappedTransactionsInsertParams) (int64, error)
	TransactionsInsert(ctx context.Context, arg []*TransactionsInsertParams) (int64, error)
	CardsDelete(ctx context.Context, userID string, iD string) (int64, error)
	CardsUpdate(ctx context.Context, userID string, iD string, name string) (int64, error)
	CategoriesDelete(ctx context.Context, authorID string, iD string) (int64, error)
	CategoriesExists(ctx context.Context, iD string, authorID string) (bool, error)
	TransactionsDelete(ctx context.Context, authorID string, iD string) (int64, error)
	MappingsDeleteForCategoryDelete(ctx context.Context, resCategory *string) error
	MappingsDeleteKeepingOrphans(ctx context.Context, authorID string, iD string) (int64, error)
	MappingsExists(ctx context.Context, authorID string, iD string) (bool, error)
	MappingsRemapExistingCategoryID(ctx context.Context, mappingID string, resolvedCategory *string) error
	MappingsRemapExistingName(ctx context.Context, mappingID string, resolvedName *string) error
	MappingsTransactionCount(ctx context.Context, mappingID string) (int64, error)
	MappingsUnmapTransactions(ctx context.Context, mappingID string) ([]*MappingsUnmapTransactionsRow, error)
	TransactionsExists(ctx context.Context, iD string, authorID string) (bool, error)
	TransactionsExistsNoID(ctx context.Context, cardID string, authedAt time.Time, settledAt time.Time, description string, amount decimal.Decimal) (bool, error)
	SendBatch(ctx context.Context, b *pgx.Batch) error
	TxFunc(ctx context.Context, h func(s Store) error) error
	// Gets a raw DB conn from a store. Be careful using this.
	GetDB() db.DBQuerier
	MappingGetAll(ctx context.Context, authorID string) ([]*data.Mapping, error)
	MappingGetByID(ctx context.Context, authorID, mappingID string) (*data.Mapping, error)
	// Map existing transactions based on a mapping, inserting mapped_transactions values as well
	TransactionsMapsMapExisting(ctx context.Context, updateName bool, authorID string, m *data.Mapping) (int, error)
	CardsNew(ctx context.Context, userID string, name string) (string, error)
	CategoriesNew(ctx context.Context, authorID string, name string, icon string, color string) (string, error)
	MappingNew(ctx context.Context, authorID string, m *data.Mapping) (string, error)
	// Delete the mapped_transaction AND unset the needed column.
	TransactionsUnmapForMappingID(ctx context.Context, mappingID string, unmapName, unmapCat bool) ([]*MappingsUnmapTransactionsRow, error)
}
