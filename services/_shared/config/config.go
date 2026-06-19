package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	DB          DBConfig
	Redis       RedisConfig
	NATS        NATSConfig
	GRPC        GRPCConfig
	Observability ObservabilityConfig
	ServiceToken string
}

type DBConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string
	MaxConns int32
}

func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode,
	)
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type NATSConfig struct {
	URL      string
	User     string
	Password string
}

type GRPCConfig struct {
	Port        int
	MetricsPort int
}

type ObservabilityConfig struct {
	OTLPEndpoint string
	LogLevel     string
	Environment  string
	ServiceName  string
}

func Load(serviceName string) (*Config, error) {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("postgres_host", "localhost")
	v.SetDefault("postgres_port", 5432)
	v.SetDefault("postgres_db", "pos_db")
	v.SetDefault("postgres_admin_password", "changeme")
	v.SetDefault("postgres_app_password", "changeme")
	v.SetDefault("postgres_ssl_mode", "disable")
	v.SetDefault("postgres_max_conns", 20)

	v.SetDefault("redis_addr", "localhost:6379")
	v.SetDefault("redis_password", "redis")
	v.SetDefault("redis_db", 0)

	v.SetDefault("nats_url", "nats://localhost:4222")
	v.SetDefault("nats_user", "pos")
	v.SetDefault("nats_password", "changeme")

	v.SetDefault("grpc_port", 8080)
	v.SetDefault("metrics_port", 9100)

	v.SetDefault("otel_exporter_otlp_endpoint", "http://localhost:4317")
	v.SetDefault("log_level", "info")
	v.SetDefault("environment", "development")

	v.SetDefault("service_token_secret", "changeme")

	return &Config{
		DB: DBConfig{
			Host:     v.GetString("postgres_host"),
			Port:     v.GetInt("postgres_port"),
			Database: v.GetString("postgres_db"),
			User:     "pos_app",
			Password: v.GetString("postgres_app_password"),
			SSLMode:  v.GetString("postgres_ssl_mode"),
			MaxConns: int32(v.GetInt("postgres_max_conns")),
		},
		Redis: RedisConfig{
			Addr:     v.GetString("redis_addr"),
			Password: v.GetString("redis_password"),
			DB:       v.GetInt("redis_db"),
		},
		NATS: NATSConfig{
			URL:      v.GetString("nats_url"),
			User:     v.GetString("nats_user"),
			Password: v.GetString("nats_password"),
		},
		GRPC: GRPCConfig{
			Port:        v.GetInt("grpc_port"),
			MetricsPort: v.GetInt("metrics_port"),
		},
		Observability: ObservabilityConfig{
			OTLPEndpoint: v.GetString("otel_exporter_otlp_endpoint"),
			LogLevel:     v.GetString("log_level"),
			Environment:  v.GetString("environment"),
			ServiceName:  serviceName,
		},
		ServiceToken: v.GetString("service_token_secret"),
	}, nil
}
