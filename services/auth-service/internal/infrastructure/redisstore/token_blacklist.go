package redisstore

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const blacklistPrefix = "pos:token:blacklist:"
const blacklistTTL = 2 * time.Hour // slightly longer than JWT TTL

type TokenBlacklist struct {
	client *redis.Client
}

func NewTokenBlacklist(client *redis.Client) *TokenBlacklist {
	return &TokenBlacklist{client: client}
}

func (b *TokenBlacklist) Blacklist(token string) error {
	key := blacklistKey(token)
	return b.client.Set(context.Background(), key, "1", blacklistTTL).Err()
}

func (b *TokenBlacklist) IsBlacklisted(token string) (bool, error) {
	key := blacklistKey(token)
	err := b.client.Get(context.Background(), key).Err()
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
