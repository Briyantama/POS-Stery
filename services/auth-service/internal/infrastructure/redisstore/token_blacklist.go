package redisstore

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const blacklistPrefix = "pos:token:blacklist:"
const blacklistTTL = 15 * time.Minute // access tokens expire in 10min; 5min buffer for clock drift

type TokenBlacklist struct {
	client *redis.Client
}

func NewTokenBlacklist(client *redis.Client) *TokenBlacklist {
	return &TokenBlacklist{client: client}
}

func (b *TokenBlacklist) Blacklist(ctx context.Context, jti string) error {
	return b.client.Set(ctx, blacklistKey(jti), "1", blacklistTTL).Err()
}

func (b *TokenBlacklist) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	err := b.client.Get(ctx, blacklistKey(jti)).Err()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check blacklist: %w", err)
	}
	return true, nil
}

func blacklistKey(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%s%x", blacklistPrefix, h)
}
