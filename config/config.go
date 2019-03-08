// Package config defines the environment variable and command-line flags
// supported by this service and includes default values for particular
// fields.
package config

import (
	"sync"

	"github.com/companieshouse/gofigure"
)

var cfg *Config
var mtx sync.Mutex

// Config defines the configuration options for this service.
type Config struct {
	BindAddr                   string   `env:"BIND_ADDR"                       flag:"bind-addr"                         flagDesc:"Bind address"`
	Collection                 string   `env:"MONGODB_COLLECTION"              flag:"mongodb-collection"                flagDesc:"MongoDB collection for data"`
	Database                   string   `env:"MONGODB_DATABASE"                flag:"mongodb-database"                  flagDesc:"MongoDB database for data"`
	MongoDBURL                 string   `env:"MONGODB_URL"                     flag:"mongodb-url"                       flagDesc:"MongoDB server URL"`
	DomainWhitelist            string   `env:"DOMAIN_WHITELIST"                flag:"domain-whitelist"                  flagDesc:"List of Valid Domains"`
	PaymentsWebURL             string   `env:"PAYMENTS_WEB_URL"                flag:"payments-web-url"                  flagDesc:"Base URL for the Payment Service Web"`
	PaymentsAPIURL             string   `env:"PAYMENTS_API_URL"                flag:"payments-api-url"                  flagDesc:"Base URL for the Payment Service API"`
	GovPayURL                  string   `env:"GOV_PAY_URL"                     flag:"gov-pay-url"                       flagDesc:"URL used to make calls to GovPay"`
	GovPayBearerTokenTreasury  string   `env:"GOV_PAY_BEARER_TOKEN_TREASURY"   flag:"gov-pay-bearer-token-treasury"     flagDesc:"Bearer Token used to authenticate API calls with GovPay for treasury payments"`
	GovPayBearerTokenChAccount string   `env:"GOV_PAY_BEARER_TOKEN_CH_ACCOUNT" flag:"gov-pay-bearer-token-ch-account"   flagDesc:"Bearer Token used to authenticate API calls with GovPay for Companies House Payments"`
	ExpiryTimeInMinutes        string   `env:"EXPIRY_TIME_IN_MINUTES"          flag:"expiry-time-in-minsutes"           flagDesc:"The expiry time for the payment session in minutes"`
	BrokerAddr                 []string `env:"KAFKA_BROKER_ADDR"               flag:"broker-addr"                       flagDesc:"Kafka broker address"`
	SchemaRegistryURL          string   `env:"SCHEMA_REGISTRY_URL"             flag:"schema-registry-url"               flagDesc:"Schema registry url"`
}

// DefaultConfig returns a pointer to a Config instance that has been populated
// with default values.
func DefaultConfig() *Config {
	return &Config{
		Database:            "payments",
		Collection:          "payments",
		ExpiryTimeInMinutes: "90",
	}
}

// Get returns a pointer to a Config instance that has been populated with
// values provided by the environment or command-line flags, or with default
// values if none are provided.
func Get() (*Config, error) {
	mtx.Lock()
	defer mtx.Unlock()

	if cfg != nil {
		return cfg, nil
	}

	cfg = DefaultConfig()

	err := gofigure.Gofigure(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
