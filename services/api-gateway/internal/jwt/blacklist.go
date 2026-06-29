package jwt

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const blacklistPrefix = "pos:token:blacklist:"

// Blacklist checks whether a JWT (identified by its JTI) has been revoked.
type Blacklist struct {
	client *redis.Client
}

// NewBlacklist returns a Blacklist backed by the given Redis client.
func NewBlacklist(rdb *redis.Client) *Blacklist {
	return &Blacklist{client: rdb}
}

// IsBlacklisted returns true if the given JTI has been added to the blacklist.
// The key pattern matches the auth-service redisstore exactly:
//
//	pos:token:blacklist:<hex(sha256(jti))>
func (b *Blacklist) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	err := b.client.Get(ctx, blacklistKey(jti)).Err()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check token blacklist: %w", err)
	}
	return true, nil
}

func blacklistKey(jti string) string {
	h := sha256.Sum256([]byte(jti))
	return fmt.Sprintf("%s%x", blacklistPrefix, h)
}
