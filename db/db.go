package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"time"
)

// DB is the management interface for the standard database handle
type DB struct {
	DB        *sql.DB
	Connector driver.Connector

	MaxOpenConns    int
	MaxIdelConns    int
	ConnMaxLifetime time.Duration
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
	r := db.DB.QueryRowContext(ctx, query, args...)
	return r
}

// Query executes a query that is expected to return rows.
func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return db.DB.QueryContext(ctx, query, args...)
}

// WithTransaction executes a function within a transaction
func (db *DB) WithTransaction(ctx context.Context, fn func(context.Context, *sql.Tx) error) error {
	var err error
	var tx *sql.Tx

	tx, err = db.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelDefault})
	if err != nil {
		return err
	}

	defer func() {
		p := recover()
		switch {
		case p != nil:
			// a panic occurred, rollback and repanic
			err = tx.Rollback()
			panic(p)
		case err != nil:
			// something went wrong, rollback
			err = tx.Rollback()
		default:
			// all good, commit
			err = tx.Commit()
		}
	}()

	err = fn(ctx, tx)
	return err
}

// Exec executes a query within a transaction that doesn't return rows
func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	var res sql.Result

	err := db.WithTransaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		var e error
		res, e = tx.ExecContext(ctx, query, args...)
		return e
	})

	return res, err
}
