package config_test

import (
	"strings"
	"testing"

	"github.com/Briyantama/POS-Stery/internal/config"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestLoad(t *testing.T) {
	valid := map[string]string{
		"DATABASE_URL":               "postgres://u:s3cret@h/db",
		"REDIS_ADDR":                 "localhost:6379",
		"REDIS_PASSWORD":             "redispw",
		"JWT_RS256_PRIVATE_KEY_PATH": "/k/priv.pem",
		"JWT_RS256_PUBLIC_KEY_PATH":  "/k/pub.pem",
	}
	without := func(name string) map[string]string {
		m := map[string]string{}
		for k, v := range valid {
			if k != name {
				m[k] = v
			}
		}
		return m
	}

	for _, name := range []string{"DATABASE_URL", "REDIS_ADDR", "JWT_RS256_PRIVATE_KEY_PATH", "JWT_RS256_PUBLIC_KEY_PATH"} {
		_, err := config.Load(env(without(name)))
		if err == nil || !strings.Contains(err.Error(), name+" is required") {
			t.Errorf("missing %s: err = %v", name, err)
		}
	}

	if _, err := config.Load(env(map[string]string{})); err == nil || !strings.HasPrefix(err.Error(), "invalid configuration: ") {
		t.Errorf("empty env: err = %v", err)
	}

	bad := without("")
	bad["PORT"], bad["REDIS_TLS"], bad["TRUSTED_PROXY_CIDRS"] = "0", "maybe", "nope"
	_, err := config.Load(env(bad))
	if err == nil {
		t.Fatal("bad values accepted")
	}
	for _, name := range []string{"PORT", "REDIS_TLS", "TRUSTED_PROXY_CIDRS"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error does not name %s: %v", name, err)
		}
	}
	if strings.Contains(err.Error(), "maybe") {
		t.Error("error leaks a value")
	}

	valid["TRUSTED_PROXY_CIDRS"] = "10.0.0.0/8, 192.168.1.1"
	cfg, err := config.Load(env(valid))
	if err != nil {
		t.Fatalf("valid: %v", err)
	}
	if cfg.HTTPPort != "8000" || len(cfg.TrustedProxies) != 2 {
		t.Errorf("defaults/proxies: %+v", cfg)
	}
	if s := cfg.String(); strings.Contains(s, "s3cret") || strings.Contains(s, "redispw") {
		t.Errorf("String leaks secrets: %s", s)
	}
}
