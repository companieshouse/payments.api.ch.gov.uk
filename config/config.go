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
	BindAddr                          string   `env:"BIND_ADDR"                       flag:"bind-addr"                         flagDesc:"Bind address"`
	Collection                        string   `env:"MONGODB_COLLECTION"              flag:"mongodb-collection"                flagDesc:"MongoDB collection for data"`
	Database                          string   `env:"MONGODB_DATABASE"                flag:"mongodb-database"                  flagDesc:"MongoDB database for data"`
	MongoDBURL                        string   `env:"MONGODB_URL"                     flag:"mongodb-url"                       flagDesc:"MongoDB server URL"`
	DomainAllowList                   string   `env:"DOMAIN_ALLOW_LIST"               flag:"domain-allow-list"                 flagDesc:"List of Valid Domains"`
	PaymentsWebURL                    string   `env:"PAYMENTS_WEB_URL"                flag:"payments-web-url"                  flagDesc:"Base URL for the Payment Service Web"`
	PaymentsAPIURL                    string   `env:"PAYMENTS_API_URL"                flag:"payments-api-url"                  flagDesc:"Base URL for the Payment Service API"`
	GovPayURL                         string   `env:"GOV_PAY_URL"                     flag:"gov-pay-url"                       flagDesc:"URL used to make calls to GovPay"`
	GovPayBearerTokenTreasury         string   `env:"GOV_PAY_BEARER_TOKEN_TREASURY"   flag:"gov-pay-bearer-token-treasury"     flagDesc:"Bearer Token used to authenticate API calls with GovPay for treasury payments"`
	GovPayBearerTokenChAccount        string   `env:"GOV_PAY_BEARER_TOKEN_CH_ACCOUNT" flag:"gov-pay-bearer-token-ch-account"   flagDesc:"Bearer Token used to authenticate API calls with GovPay for Companies House payments"`
	GovPayBearerTokenSanctionsAccount string   `env:"GOV_PAY_BEARER_TOKEN_SANCTIONS_ACCOUNT" flag:"gov-pay-bearer-token-sanctions-account"   flagDesc:"Bearer Token used to authenticate API calls with GovPay for sanctions penalty payments"`
	GovPayBearerTokenLegacy           string   `env:"GOV_PAY_BEARER_TOKEN_LEGACY"     flag:"gov-pay-bearer-token-legacy"       flagDesc:"Bearer Token used to authenticate API calls with GovPay for payments on legacy Companies House services"`
	GovPaySandbox                     bool     `env:"GOV_PAY_SANDBOX"                 flag:"gov-pay-sandbox"                   flagDesc:"Gov Pay Sandbox - returns different refund status values"`
	GovPayExpiryTime                  int      `env:"GOV_PAY_EXPIRY_TIME"             flag:"gov-pay-expiry_time"               flagDesc:"Gov Pay Expiry Time in minutes"`
	GovPayMaxCheckingDays             int      `env:"GOV_PAY_MAX_CHECKING_DAYS"       flag:"gov-pay-max-checking-days"         flagDesc:"Gov Pay Max Allowed Days for rechecking payment"`
	ExpiryTimeInMinutes               string   `env:"EXPIRY_TIME_IN_MINUTES"          flag:"expiry-time-in-minutes"            flagDesc:"The expiry time for the payment session in minutes"`
	BrokerAddr                        []string `env:"KAFKA_BROKER_ADDR"               flag:"broker-addr"                       flagDesc:"Kafka broker address"`
	SchemaRegistryURL                 string   `env:"SCHEMA_REGISTRY_URL"             flag:"schema-registry-url"               flagDesc:"Schema registry url"`
	ChsAPIKey                         string   `env:"CHS_API_KEY"                     flag:"chs-api-key"                       flagDesc:"API access key"`
	SecureAppCostsRegex               string   `env:"SECURE_APP_COSTS_REGEX"          flag:"secure-app-costs-regex"            flagDesc:"Regex to match secure app costs resource"`
	PaypalEnv                         string   `env:"PAYPAL_ENV"                      flag:"paypal-env"                        flagDesc:"live or test"`
	PaypalClientID                    string   `env:"PAYPAL_CLIENT_ID"                flag:"paypal-client-id"                  flagDesc:"PayPal Client ID"`
	PaypalSecret                      string   `env:"PAYPAL_SECRET"                   flag:"paypal-secret"                     flagDesc:"PayPal Secret"`
	RefundBatchSize                   int      `env:"REFUND_BATCH_SIZE"               flag:"refund-batch-size"                 flagDesc:"Refund batch size"`
}

// DefaultConfig returns a pointer to a Config instance that has been populated
// with default values.
func DefaultConfig() *Config {
	return &Config{
		Database:              "payments",
		Collection:            "payments",
		ExpiryTimeInMinutes:   "90",
		GovPayExpiryTime:      90,
		GovPayMaxCheckingDays: 30,
		RefundBatchSize:       20,
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
