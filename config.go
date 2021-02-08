package main

import (
	"code.cloudfoundry.org/go-envstruct"
)

// Config is the configuration for a LogCache.
type Config struct {
	Addr        string `env:"ADDR, required, report"`
	AppSelector string `env:"APP_SELECTOR, required, report"`
	Namespace   string `env:"NAMESPACE"`
	NodeCacheTTL string `env:"NODE_CACHE_TTL"`

	// QueryTimeout sets the maximum allowed runtime for a single PromQL query.
	// Smaller timeouts are recommended.
	QueryTimeout int64 `env:"QUERY_TIMEOUT, report"`
}

// LoadConfig creates Config object from environment variables
func LoadConfig() (*Config, error) {
	c := Config{
		//Addr:         ":8080",
		NodeCacheTTL: "30s",
		QueryTimeout: 10,
	}

	if err := envstruct.Load(&c); err != nil {
		return nil, err
	}

	return &c, nil
}
