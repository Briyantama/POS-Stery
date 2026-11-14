package redisstore

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// BlacklistPrefix is the key prefix the access-token verifier reads
// (key = prefix + hex sha256(jti)).
const BlacklistPrefix = "pos:token:blacklist:"

// minTTL keeps an already-expiring token revoked past clock drift.
const minTTL = 5 * time.Second

type TokenBlacklist struct {
	client *redis.Client
}

func NewTokenBlacklist(client *redis.Client) *TokenBlacklist {
	return &TokenBlacklist{client: client}
}

// Blacklist revokes jti until expiresAt (the token's own expiry).
func (b *TokenBlacklist) Blacklist(ctx context.Context, jti string, expiresAt time.Time) error {
	ttl := max(time.Until(expiresAt), minTTL)
	return b.client.Set(ctx, blacklistKey(jti), "1", ttl).Err()
}

func blacklistKey(jti string) string {
	h := sha256.Sum256([]byte(jti))
	return fmt.Sprintf("%s%x", BlacklistPrefix, h)
}
