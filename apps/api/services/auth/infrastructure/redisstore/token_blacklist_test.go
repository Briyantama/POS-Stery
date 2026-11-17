package redisstore_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/Briyantama/POS-Stery/services/auth/infrastructure/redisstore"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// The key must match what the access-token verifier reads: prefix + hex sha256(jti), TTL = remaining token life.
func TestBlacklistKeyAndTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	if err := redisstore.NewTokenBlacklist(rdb).Blacklist(context.Background(), "jti-1", time.Now().Add(10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	key := fmt.Sprintf("%s%x", redisstore.BlacklistPrefix, sha256.Sum256([]byte("jti-1")))
	if !mr.Exists(key) {
		t.Fatalf("key %s not set", key)
	}
	if ttl := mr.TTL(key); ttl <= 9*time.Minute || ttl > 10*time.Minute {
		t.Fatalf("ttl = %s, want about 10m", ttl)
	}
}
