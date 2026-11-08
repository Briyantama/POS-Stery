package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Briyantama/POS-Stery/internal/config"
)

// TestRoutes builds the real wiring (the pool never connects) and checks that the routes register
// without a pattern conflict, liveness is public and each service's routes require a token.
func TestRoutes(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pubDER, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	dir := t.TempDir()
	cfg := config.Config{
		JWTPrivateKeyPath: filepath.Join(dir, "private.pem"),
		JWTPublicKeyPath:  filepath.Join(dir, "public.pem"),
	}
	writePEM(t, cfg.JWTPrivateKeyPath, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(key))
	writePEM(t, cfg.JWTPublicKeyPath, "PUBLIC KEY", pubDER)

	pool, err := pgxpool.New(context.Background(), "postgres://nobody@127.0.0.1:1/none")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	rdb := redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()})

	handlers, err := build(cfg, pool, rdb)
	if err != nil {
		t.Fatal(err)
	}
	mux := newMux(handlers...)

	rows := []struct {
		method, path string
		want         int
	}{
		{"GET", "/healthz", 200},
		{"GET", "/api/v1/auth/sessions", 401},
		{"GET", "/api/v1/products", 401},
		{"GET", "/api/v1/inventory", 401},
		{"POST", "/api/v1/sales", 401},
		{"GET", "/api/v1/suppliers", 401},
		{"GET", "/api/v1/customers", 401},
	}
	for _, r := range rows {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(r.method, r.path, nil))
		if rec.Code != r.want {
			t.Errorf("%s %s: got %d want %d", r.method, r.path, rec.Code, r.want)
		}
	}
}

func writePEM(t *testing.T, path, typ string, der []byte) {
	t.Helper()
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
}
