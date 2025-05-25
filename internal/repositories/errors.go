package repositories

import (
	"database/sql"
	"errors"
)

var (
	// ErrNotFound is returned when a specific record is not found.
	ErrNotFound = errors.New("requested record not found")

	// ErrDatabaseError is returned for unexpected database errors.
	// It can be used to wrap more specific driver errors.
	ErrDatabaseError = errors.New("database error")

	// ErrDuplicateKey is returned when an insert/update violates a unique constraint.
	ErrDuplicateKey = errors.New("duplicate key value violates unique constraint")
)

// SQLExecutor defines an interface that can be satisfied by *sql.DB or *sql.Tx
// This allows repository methods to be used within transactions or with a direct DB connection.
type SQLExecutor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

// scanner is an interface satisfied by *sql.Row and *sql.Rows.
// This allows for generic scanning helpers.
type scanner interface {
	Scan(dest ...interface{}) error
}
