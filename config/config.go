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
	BindAddr        string `env:"BIND_ADDR"            flag:"bind-addr"              flagDesc:"Bind address"`
	Collection      string `env:"MONGODB_COLLECTION"   flag:"mongodb-collection"     flagDesc:"MongoDB collection for data"`
	Database        string `env:"MONGODB_DATABASE"     flag:"mongodb-database"       flagDesc:"MongoDB database for data"`
	MongoDBURL      string `env:"MONGODB_URL"          flag:"mongodb-url"            flagDesc:"MongoDB server URL"`
	DomainWhitelist string `env:"DOMAIN_WHITELIST"     flag:"domain-whitelist"       flagDesc:"List of Valid Domains"`
	PaymentsWebURL  string `env:"PAYMENTS_WEB_URL"     flag:"payments-web-url"       flagDesc:"Base URL for the Payment Service Web"`
	GovPayURL       string `env:"GOV_PAY_URL"          flag:"gov-pay-url"            flagDesc:"URL used to make calls to GovPay"`
	// GovPayURL and GovPayBearerToken:
	// can we use a single gov pay integration for multiple bank accounts?
	// we know that we have to take LFP payments into different account than other payments
	// so we should design this in from the start
	// Will we end up with different env vars when we add second integration
	//   - GovPayURL with GovPayBearerToken and GovPayURLTreasury & GovPayBearerTokenTreasury ?
	//     - being more explicit with Treasury integration/account as the default will then be CH bank account as most people would expect
	//   - will need to duplicate for any integration we add going forwards - barclays, paypal, CH account
	// Or is there a nicer way we can handle this? maybe config could contain a list of integrations?
	//  - export GOV_PAY_INTEGRATIONS=CH_ACCOUNT|https://govpayurl/blah|B34r3rT0k3n TREASURY_ACCOUNT|https://govpayurl/blah2|B34r3rT0k3n2
	//  - maybe overkill as we might obly ever have 2 integrations I'd assume adding a 3rd would require code changes anyway
	GovPayBearerToken string `env:"GOV_PAY_BEARER_TOKEN" flag:"gov-pay-bearer-token"   flagDesc:"Bearer Token used to authenticate API calls with GovPay"`
}

// DefaultConfig returns a pointer to a Config instance that has been populated
// with default values.
func DefaultConfig() *Config {
	return &Config{
		Database:   "payments",
		Collection: "payments",
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
