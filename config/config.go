// Package config defines the environment variable and command-line flags
// supported by this service and includes default values for particular
// fields.
package config

import (
	"os"
	"sync"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/gofigure"
)

var cfg *Config
var mtx sync.Mutex

// Config defines the configuration options for this service.
type Config struct {
	BindAddr   string `env:"BIND_ADDR"                  flag:"bind-addr"                  flagDesc:"Bind address"`
	EricAddr   string `env:"ERIC_LOCAL_URL"             flag:"eric-addr"                  flagDesc:"Eric address"`
	Collection string `env:"MONGODB_COLLECTION"         flag:"mongodb-collection"         flagDesc:"MongoDB collection for data"`
	Database   string `env:"MONGODB_DATABASE"           flag:"mongodb-database"           flagDesc:"MongoDB database for data"`
	MongoDBURL string `env:"MONGODB_URL"                flag:"mongodb-url"                flagDesc:"MongoDB server URL"`
	APIKey     string `env:"PAYMENTS_API_KEY"           flag:"api-key"                    flagDesc:"API key used to authenticate for internal API calls"`
}

// DefaultConfig returns a pointer to a Config instance that has been populated
// with default values.
func DefaultConfig() *Config {
	return &Config{
		EricAddr:   ":4001",
		Database:   "payments",
		Collection: "payments",
	}
}

// Get returns a pointer to a Config instance that has been populated with
// values provided by the environment or command-line flags, or with default
// values if none are provided.
func Get() *Config {
	mtx.Lock()
	defer mtx.Unlock()

	if cfg != nil {
		return cfg
	}

	cfg = DefaultConfig()

	err := gofigure.Gofigure(cfg)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	return cfg
}
