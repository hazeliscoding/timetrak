// Package db wraps the PostgreSQL connection pool and transactional helpers.
package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is the process-wide connection pool.
type Pool struct{ *pgxpool.Pool }

// Open establishes the connection pool and verifies connectivity.
func Open(ctx context.Context, dsn string) (*Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := p.Ping(ctx); err != nil {
		p.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Pool{p}, nil
}

// TxFn is a function run inside a transaction. Returning an error rolls back.
type TxFn func(ctx context.Context, tx pgx.Tx) error

// InTx runs fn within a single transaction. Commits on nil error, rolls back on error or panic.
func (p *Pool) InTx(ctx context.Context, fn TxFn) error {
	tx, err := p.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback(ctx)
			panic(r)
		}
	}()
	if err := fn(ctx, tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

// IsUniqueViolation reports whether err is a PostgreSQL unique_violation (SQLSTATE 23505).
// Callers use this to detect duplicate-email and "second running timer" conflicts.
func IsUniqueViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}

// IsCheckViolation reports whether err is a check_constraint violation (SQLSTATE 23514).
func IsCheckViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23514"
}

// IsForeignKeyViolation reports whether err is a foreign_key_violation (SQLSTATE 23503).
func IsForeignKeyViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23503"
}
