package db_test

import (
	"testing"

	"github.com/Briyantama/POS-Stery/internal/testdb"
)

const idDefaultCount = `SELECT count(*)
	FROM pg_attrdef d
	JOIN pg_class cl ON cl.oid = d.adrelid
	JOIN pg_namespace n ON n.oid = cl.relnamespace
	JOIN pg_attribute a ON a.attrelid = d.adrelid AND a.attnum = d.adnum
	WHERE a.attname = 'id'
	  AND pg_get_expr(d.adbin, d.adrelid) = $1
	  AND cl.relkind IN ('r', 'p')
	  AND n.nspname NOT IN ('pg_catalog', 'information_schema')`

func TestPrimaryKeys_noIDStillDefaultsToRandomUUID(t *testing.T) {
	// Arrange
	conn := testdb.NewMigrated(t)

	// Act
	var n int
	err := conn.QueryRow(idDefaultCount, "gen_random_uuid()").Scan(&n)

	// Assert
	if err != nil {
		t.Fatalf("count defaults: %v", err)
	}
	if n != 0 {
		t.Fatalf("%d id columns still default to gen_random_uuid(), want 0", n)
	}
}

func TestPrimaryKeys_defaultToUUIDv7(t *testing.T) {
	// Arrange
	conn := testdb.NewMigrated(t)

	// Act
	var n, version int
	err := conn.QueryRow(idDefaultCount, "uuidv7()").Scan(&n)
	if err == nil {
		err = conn.QueryRow(`SELECT uuid_extract_version(uuidv7())`).Scan(&version)
	}

	// Assert
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if n == 0 {
		t.Fatal("no id column defaults to uuidv7()")
	}
	if version != 7 {
		t.Fatalf("uuidv7() version = %d, want 7", version)
	}
}
