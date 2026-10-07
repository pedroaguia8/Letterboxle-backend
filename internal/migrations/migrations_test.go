package migrations

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"testing"
	"testing/fstest"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// TestUpDownUp applies every real migration, rolls all of them back, and
// applies them again, so a broken Down block (or an Up that only works once)
// fails CI rather than a deploy.
func TestUpDownUp(t *testing.T) {
	db := createTempDatabase(t, testServerURL(t))
	if err := upDownUp(context.Background(), db, os.DirFS("../../sql/schema")); err != nil {
		t.Fatal(err)
	}
}

// TestUpDownUpCatchesBrokenDown feeds upDownUp small migration sets with a
// broken Down block, to prove each kind of breakage is reported.
func TestUpDownUpCatchesBrokenDown(t *testing.T) {
	serverURL := testServerURL(t)
	cases := []struct {
		name string
		up   string
		down string
	}{
		// Caught by the leftover-tables check.
		{"down leaves a table", "CREATE TABLE t (id INT);", "SELECT 1;"},
		// Caught by the Down itself.
		{"down errors", "CREATE TABLE t (id INT);", "DROP TABLE no_such_table;"},
		// Leaves no table behind, so only the second Up catches it.
		{"down leaves a type", "CREATE TYPE mood AS ENUM ('ok');", "SELECT 1;"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fsys := fstest.MapFS{"00001_test.sql": {Data: []byte(
				"-- +goose Up\n" + c.up + "\n-- +goose Down\n" + c.down + "\n")}}
			db := createTempDatabase(t, serverURL)
			err := upDownUp(context.Background(), db, fsys)
			if err == nil {
				t.Fatal("expected an error, got none")
			}
			t.Logf("got expected error: %v", err)
		})
	}
}

// upDownUp migrates db all the way up, all the way down, and up again, and
// checks that the down left no tables behind.
func upDownUp(ctx context.Context, db *sql.DB, fsys fs.FS) error {
	provider, err := newProvider(db, fsys)
	if err != nil {
		return err
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("first up: %w", err)
	}
	if _, err := provider.DownTo(ctx, 0); err != nil {
		return fmt.Errorf("down to 0: %w", err)
	}

	// Down blocks must undo everything, or the next Up would trip over leftovers
	// (or, worse, silently keep them).
	var leftover int
	err = db.QueryRowContext(ctx, `
		SELECT count(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name <> 'goose_db_version'`).Scan(&leftover)
	if err != nil {
		return err
	}
	if leftover != 0 {
		return fmt.Errorf("%d tables left after migrating down to 0", leftover)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("second up: %w", err)
	}
	return nil
}

// testServerURL returns TEST_DB_URL: a Postgres server where tests may create
// and drop their own throwaway databases (and touch nothing else). Locally it
// comes from .env; in CI the workflow sets it. A variable already set in the
// environment wins over .env. Missing means the test fails rather than skips,
// so the check can't silently stop running.
func testServerURL(t *testing.T) string {
	t.Helper()
	// No .env (as in CI) is fine; a broken one isn't.
	if err := godotenv.Load("../../.env"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("loading .env: %v", err)
	}
	serverURL := os.Getenv("TEST_DB_URL")
	if serverURL == "" {
		t.Fatal("TEST_DB_URL is not set. Locally, copy it from .env.example into .env.")
	}
	return serverURL
}

// createTempDatabase creates an empty database, named with a random suffix, on
// the server at serverURL and returns a connection to it. Cleanup closes it and
// drops the database. Nothing else on the server is touched.
func createTempDatabase(t *testing.T, serverURL string) *sql.DB {
	t.Helper()
	ctx := context.Background()

	u, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("parsing TEST_DB_URL: %v", err)
	}

	admin, err := sql.Open("postgres", serverURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })
	// Only the host is printed, to keep the password out of the output.
	if err := admin.PingContext(ctx); err != nil {
		t.Fatalf("can't reach Postgres at %s (TEST_DB_URL): %v", u.Host, err)
	}

	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatal(err)
	}
	name := "migrations_test_" + hex.EncodeToString(suffix)
	// The name is generated above, not user input, so it's safe to splice in.
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("creating test database: %v", err)
	}

	u.Path = "/" + name
	db, err := sql.Open("postgres", u.String())
	if err != nil {
		t.Fatal(err)
	}

	// Cleanups run last-in first-out: close this connection, drop the
	// database, then close the admin connection registered above.
	t.Cleanup(func() {
		db.Close()
		if _, err := admin.ExecContext(ctx, "DROP DATABASE "+name); err != nil {
			t.Errorf("dropping test database %s: %v", name, err)
		}
	})
	return db
}
