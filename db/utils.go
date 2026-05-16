package db

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func NoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// Reports if err is a unique constraint violation
func UniqueConstraint(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgerrcode.UniqueViolation
	}

	return false
}

func AscKey(asc bool) string {
	if asc {
		return "ASC"
	}

	return "DESC"
}
