package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"time"
)

// DB is the management interface for the standard database handle
type DB struct {
	DB        *sql.DB
	Connector driver.Connector

	// MaxOpenConns is the maximum number of open connections to the database.
	MaxOpenConns int
	// MaxIdelConns is the maximum number of connections in the idle connection pool.
	MaxIdelConns int
	// ConnMaxLifetime is the maximum lifetime of a connection.
	ConnMaxLifetime time.Duration
	// ConnMaxIdelTime is the maximum lifetime of an idle connection.
	ConnMaxIdelTime time.Duration
}

// New creates a new database object
func New(connector driver.Connector, options ...func(*DB) error) (*DB, error) {
	db := &DB{
		// max 25 cuncurrently connections by default
		MaxOpenConns: 25,
		// max 2 concurrently idle connections by default
		MaxIdelConns: 2,
		// connection is reused forever by default
		ConnMaxLifetime: 0,
		// idle connection is closed after 5min by default
		ConnMaxIdelTime: 5 * time.Minute,
	}

	for _, option := range options {
		err := option(db)
		if err != nil {
			return nil, err
		}
	}

	db.Connector = connector
	db.DB = sql.OpenDB(db.Connector)

	// connection pool options
	db.DB.SetMaxOpenConns(db.MaxOpenConns)
	db.DB.SetMaxIdleConns(db.MaxIdelConns)
	db.DB.SetConnMaxLifetime(db.ConnMaxLifetime)
	db.DB.SetConnMaxIdleTime(db.ConnMaxIdelTime)

	return db, nil
}

// WithMaxOpenConns is a database option that sets the maximum number of open connections
func WithMaxOpenConns(conns int) func(*DB) error {
	return func(db *DB) error {
		db.MaxOpenConns = conns
		return nil
	}
}

// WithMaxIdelConns is a database option that sets the maximum number of idle connections
func WithMaxIdelConns(conns int) func(*DB) error {
	return func(db *DB) error {
		db.MaxIdelConns = conns
		return nil
	}
}

// WithConnMaxLifetime is a database option that sets the maximum lifttime of a connection
func WithConnMaxLifetime(d time.Duration) func(*DB) error {
	return func(db *DB) error {
		db.ConnMaxLifetime = d
		return nil
	}
}

// WithConnMaxIdleTime is a database option that sets the maximum idle time of a connection
func WithConnMaxIdleTime(d time.Duration) func(*DB) error {
	return func(db *DB) error {
		db.ConnMaxIdelTime = d
		return nil
	}
}

// Ping checks if the database is reachable
func (db *DB) Ping() error {
	return db.DB.Ping()
}

// Close closes the database connection
func (db *DB) Close() {
	db.DB.Close()
}

// QueryRow executes a query that is expected to return at most one row.
func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	// if transaction is already started then use it
	if tx, ok := ctx.Value(txKey).(*sql.Tx); ok {
		return tx.QueryRowContext(ctx, query, args...)
	}

	// otherwise query row without transaction
	r := db.DB.QueryRowContext(ctx, query, args...)
	return r
}

// Query executes a query that is expected to return rows.
func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	// if transaction is already started then use it
	if tx, ok := ctx.Value(txKey).(*sql.Tx); ok {
		return tx.QueryContext(ctx, query, args...)
	}

	// otherwise query row without transaction
	return db.DB.QueryContext(ctx, query, args...)
}

// WithTransaction executes a function within a transaction
func (db *DB) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	// if transaction is already started then use it
	if _, ok := ctx.Value(txKey).(*sql.Tx); ok {
		return fn(ctx)
	}

	// otherwise start a transaction
	var err error
	var tx *sql.Tx

	tx, err = db.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelDefault})
	if err != nil {
		return err
	}

	// create a new context with the transaction
	txCtx := context.WithValue(ctx, txKey, tx)

	// execute callback in background
	pch, ech := doTx(txCtx, fn)

	// wait for the callback to finish or context cancled
	select {
	case <-ctx.Done():
		// if the parent context is cancled
		// double check if the rollback is called
		if rerr := tx.Rollback(); rerr != nil {
			if errors.Is(rerr, sql.ErrTxDone) {
				// ignore this error
			} else {
				return fmt.Errorf("%s: %w", rerr, ctx.Err())
			}
		}

		return ctx.Err()
	case r, ok := <-pch:
		if ok {
			// if the callback has panic
			// rollback the transaction and repanic
			if rerr := tx.Rollback(); rerr != nil {
				if errors.Is(rerr, sql.ErrTxDone) {
					// ignore this error
				} else {
					panic(fmt.Sprintf("%s: %v", rerr, r))
				}
			}
			panic(r)
		}
	case ferr, ok := <-ech:
		switch {
		case ok && ferr != nil:
			// if the callback finished with error
			// rollback the transaction and return the error
			if rerr := tx.Rollback(); rerr != nil {
				if errors.Is(rerr, sql.ErrTxDone) {
					// ignore this error
				} else {
					return fmt.Errorf("%v: %w", ferr, rerr)
				}
			}
			return ferr
		case ok && ferr == nil:
			// if the callback finished without error
			// commit the transaction and return the error
			return tx.Commit()
		}
	}

	// something went wrong, we should never reach here
	// try to rollback the transaction, and panic
	if rerr := tx.Rollback(); rerr != nil {
		if errors.Is(rerr, sql.ErrTxDone) {
			// ignore this error
		} else {
			panic(fmt.Sprintf("%s: db(trx): something went wrong", rerr))
		}
	}
	panic("db(trx): something went wrong")
}

// Exec executes a query within a transaction that doesn't return rows
func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	var res sql.Result

	err := db.WithTransaction(ctx, func(ctx context.Context) error {
		// extract the transaction from the context
		if tx, ok := ctx.Value(txKey).(*sql.Tx); ok {
			var eerr error
			res, eerr = tx.ExecContext(ctx, query, args...)
			return eerr
		}

		// transaction should be found in the context
		panic("DB: transaction not found")
	})

	return res, err
}
