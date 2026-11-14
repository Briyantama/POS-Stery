package db_test

import (
	"testing"

	"github.com/Briyantama/stery-kit/go/migrate"

	"github.com/Briyantama/POS-Stery/internal/db"
)

func TestMigrations_versionsAreUnique(t *testing.T) {
	if err := migrate.CheckUniqueVersions(db.Migrations); err != nil {
		t.Fatal(err)
	}
}
