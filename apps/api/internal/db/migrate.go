// Package db owns this product's PostgreSQL schema: the embedded goose migrations, applied by stery-kit/migrate.
package db

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"

	"github.com/Briyantama/stery-kit/go/migrate"
)

//go:embed migrations/*.sql
var files embed.FS

// Migrations holds the migration files at its root, the layout stery-kit/migrate expects.
var Migrations = mustSub(files, "migrations")

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err) // dir is a constant that matches the embed pattern
	}
	return sub
}

// Fingerprint identifies the migration set; tests name their template database after it.
func Fingerprint() string { return migrate.Fingerprint(Migrations) }

// Up applies every pending migration. It is safe to call repeatedly.
func Up(ctx context.Context, conn *sql.DB) error { return migrate.Up(ctx, conn, Migrations) }

// Down reverts every applied migration. Intended for tests and local resets.
func Down(ctx context.Context, conn *sql.DB) error { return migrate.Down(ctx, conn, Migrations) }
