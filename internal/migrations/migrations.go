// Package migrations applies the goose migrations embedded in the binary.
package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// newProvider builds a goose provider over fsys, which must hold the .sql
// migrations at its root. The Postgres session locker (an advisory lock) makes
// a second concurrent Up wait instead of racing the first.
func newProvider(db *sql.DB, fsys fs.FS) (*goose.Provider, error) {
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, fmt.Errorf("creating session locker: %w", err)
	}
	return goose.NewProvider(goose.DialectPostgres, db, fsys, goose.WithSessionLocker(locker))
}

// Up applies every pending migration in fsys.
func Up(ctx context.Context, db *sql.DB, fsys fs.FS) error {
	provider, err := newProvider(db, fsys)
	if err != nil {
		return err
	}
	results, err := provider.Up(ctx)
	for _, r := range results {
		log.Printf("Applied migration %s in %s", r.Source.Path, r.Duration)
	}
	if err != nil {
		return err
	}
	if len(results) == 0 {
		log.Println("No pending migrations")
	}
	return nil
}

// CheckNoPending returns an error if the database is missing any migration in
// fsys. Migrations applied to the database but unknown to fsys (an older binary
// running after a rollback) are not an error, since migrations are kept
// backward-compatible.
func CheckNoPending(ctx context.Context, db *sql.DB, fsys fs.FS) error {
	provider, err := newProvider(db, fsys)
	if err != nil {
		return err
	}
	pending, err := provider.HasPending(ctx)
	if err != nil {
		return fmt.Errorf("checking for pending migrations: %w", err)
	}
	if pending {
		return fmt.Errorf("database has pending migrations, run \"app migrate\" first")
	}
	return nil
}
