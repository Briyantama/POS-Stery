package config

import (
	"os"
	"strings"

	sharedconfig "github.com/pos-stery/pos-stery/services/_shared/config"
)

// Config extends the shared config with gateway-specific fields.
type Config struct {
	Shared            *sharedconfig.Config
	HTTPPort          string
	JWTPublicKeyPath  string
	AuthServiceURL    string
	ProductServiceURL string
	InventoryServiceURL string
	SalesServiceURL   string
	SupplierServiceURL string
	CustomerServiceURL string
}

// Load reads shared config and overlays gateway-specific env vars.
func Load() (*Config, error) {
	shared, err := sharedconfig.Load("api-gateway")
	if err != nil {
		return nil, err
	}

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8000"
	}

	jwtPublicKeyPath := os.Getenv("JWT_RS256_PUBLIC_KEY_PATH")
	if jwtPublicKeyPath == "" {
		jwtPublicKeyPath = "/etc/pos/jwt.pub"
	}

	return &Config{
		Shared:              shared,
		HTTPPort:            httpPort,
		JWTPublicKeyPath:    jwtPublicKeyPath,
		AuthServiceURL:      stripScheme(envOrDefault("AUTH_SERVICE_URL", "localhost:8081")),
		ProductServiceURL:   stripScheme(envOrDefault("PRODUCT_SERVICE_URL", "localhost:8082")),
		InventoryServiceURL: stripScheme(envOrDefault("INVENTORY_SERVICE_URL", "localhost:8083")),
		SalesServiceURL:     stripScheme(envOrDefault("SALES_SERVICE_URL", "localhost:8084")),
		SupplierServiceURL:  stripScheme(envOrDefault("SUPPLIER_SERVICE_URL", "localhost:8085")),
		CustomerServiceURL:  stripScheme(envOrDefault("CUSTOMER_SERVICE_URL", "localhost:8086")),
	}, nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// stripScheme removes http:// or https:// prefix from a URL so that gRPC
// can use a bare host:port address.
func stripScheme(url string) string {
	if after, ok := strings.CutPrefix(url, "https://"); ok {
		return after
	}
	if after, ok := strings.CutPrefix(url, "http://"); ok {
		return after
	}
	return url
}
