// Package testdb binds stery-kit/testdb to this product's migrations and database-name prefix.
package testdb

import (
	"database/sql"
	"net/url"
	"testing"

	kit "github.com/Briyantama/stery-kit/go/testdb"

	"github.com/Briyantama/POS-Stery/internal/db"
)

const prefix = "pos"

// New returns a fresh, empty database dropped when the test ends; skips without TEST_DATABASE_URL.
func New(t *testing.T) *sql.DB { t.Helper(); return kit.New(t, prefix) }

// NewMigrated returns a fresh database with every migration applied.
func NewMigrated(t *testing.T) *sql.DB { t.Helper(); return kit.NewMigrated(t, prefix, db.Migrations) }

// NewMigratedURL is NewMigrated returning the connection URL, for pgxpool.New.
func NewMigratedURL(t *testing.T) string {
	t.Helper()
	return kit.NewMigratedURL(t, prefix, db.Migrations)
}

// AsApp returns dbURL with every session switched to pos_app, which is subject to RLS. The admin
// URL from TEST_DATABASE_URL is a superuser or BYPASSRLS, so tenant isolation only shows through this one.
func AsApp(t *testing.T, dbURL string) string {
	t.Helper()
	u, err := url.Parse(dbURL)
	if err != nil {
		t.Fatalf("parse database url: %v", err)
	}
	q := u.Query()
	q.Set("role", "pos_app")
	u.RawQuery = q.Encode()
	return u.String()
}
