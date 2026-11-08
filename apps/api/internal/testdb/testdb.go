// Package testdb binds stery-kit/testdb to this product's migrations and database-name prefix.
package testdb

import (
	"database/sql"
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
