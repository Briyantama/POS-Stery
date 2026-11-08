// Package config reads the process configuration from the environment.
package config

import (
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
)

const (
	defaultHTTPPort = "8000"
	maxPort         = 65535
)

// Config is the validated process configuration.
type Config struct {
	HTTPPort string

	DatabaseURL string

	RedisAddr     string
	RedisPassword string
	RedisTLS      bool

	// Paths to the PEM files holding the RS256 key pair.
	JWTPrivateKeyPath string
	JWTPublicKeyPath  string

	// ServiceEnv and LogLevel carry over from the old shared config (ENVIRONMENT, LOG_LEVEL).
	ServiceEnv string
	LogLevel   string

	TrustedProxies []netip.Prefix
}

// Load reads and validates the configuration, reporting every problem at once.
// Error messages name variables but never include their values.
func Load(getenv func(string) string) (Config, error) {
	var problems []string

	required := func(name string) string {
		value := getenv(name)
		if value == "" {
			problems = append(problems, name+" is required")
		}
		return value
	}
	fallback := func(name, def string) string {
		if value := getenv(name); value != "" {
			return value
		}
		return def
	}

	cfg := Config{
		HTTPPort:          fallback("PORT", defaultHTTPPort),
		DatabaseURL:       required("DATABASE_URL"),
		RedisAddr:         required("REDIS_ADDR"),
		RedisPassword:     getenv("REDIS_PASSWORD"),
		JWTPrivateKeyPath: required("JWT_RS256_PRIVATE_KEY_PATH"),
		JWTPublicKeyPath:  required("JWT_RS256_PUBLIC_KEY_PATH"),
		ServiceEnv:        fallback("SERVICE_ENV", "development"),
		LogLevel:          fallback("LOG_LEVEL", "info"),
	}

	if n, err := strconv.Atoi(cfg.HTTPPort); err != nil || n < 1 || n > maxPort {
		problems = append(problems, "PORT must be a port number between 1 and 65535")
	}
	if raw := getenv("REDIS_TLS"); raw != "" {
		useTLS, err := strconv.ParseBool(raw)
		if err != nil {
			problems = append(problems, "REDIS_TLS must be true or false")
		}
		cfg.RedisTLS = useTLS
	}
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		problems = append(problems, "LOG_LEVEL must be debug, info, warn or error")
	}
	if raw := getenv("TRUSTED_PROXY_CIDRS"); raw != "" {
		proxies, ok := parseProxies(raw)
		if !ok {
			problems = append(problems, "TRUSTED_PROXY_CIDRS must be a comma-separated list of CIDR ranges or addresses")
		}
		cfg.TrustedProxies = proxies
	}

	if len(problems) > 0 {
		return Config{}, errors.New("invalid configuration: " + strings.Join(problems, "; "))
	}
	return cfg, nil
}

// parseProxies reads a comma-separated list of CIDR ranges; a bare address is a single-host range.
func parseProxies(raw string) ([]netip.Prefix, bool) {
	var out []netip.Prefix
	for part := range strings.SplitSeq(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if prefix, err := netip.ParsePrefix(part); err == nil {
			out = append(out, prefix.Masked())
			continue
		}
		addr, err := netip.ParseAddr(part)
		if err != nil {
			return nil, false
		}
		out = append(out, netip.PrefixFrom(addr, addr.BitLen()))
	}
	return out, true
}

// String summarizes the configuration with every secret redacted.
func (c Config) String() string {
	return fmt.Sprintf(
		"Config{http=:%s env=%s logLevel=%s redis=%s redisTLS=%t jwtKeys=%s,%s trustedProxies=%d database=[redacted] redisPassword=[redacted]}",
		c.HTTPPort, c.ServiceEnv, c.LogLevel, c.RedisAddr, c.RedisTLS, c.JWTPrivateKeyPath, c.JWTPublicKeyPath, len(c.TrustedProxies),
	)
}
