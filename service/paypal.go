package service

import (
	"context"
	"fmt"
	"os"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/davecgh/go-spew/spew"
	"github.com/plutov/paypal/v4"
)

// PayPalService handles the specific functionality of integrating PayPal into Payment Sessions
type PayPalService struct {
	PaymentService PaymentService
}

func (pp PayPalService) GetBearerToken() (*paypal.TokenResponse, error) {
	cfg, err := config.Get()
	if err != nil {
		return nil, fmt.Errorf("error getting config: %s", err)
	}
	paypalAPIBase := getPaypalAPIBase(cfg.PaypalEnv)
	if paypalAPIBase == "" {
		return nil, fmt.Errorf("invalid paypal env in config: %s", cfg.PaypalEnv)
	}

	c, err := paypal.NewClient(cfg.PaypalClientID, cfg.PaypalSecret, paypalAPIBase)
	if err != nil {
		log.Error(fmt.Errorf("error creating new paypal client: %v", err))
		// TODO
		return nil, err
	}

	spew.Dump(c)

	c.SetLog(os.Stdout) // Set log to terminal stdout

	accessToken, err := c.GetAccessToken(context.Background())
	if err != nil {
		log.Error(fmt.Errorf("error getting access token: %v", err))
		return nil, err // TODO
	}
	return accessToken, nil

}

func getPaypalAPIBase(env string) string {
	switch env {
	case "live":
		return paypal.APIBaseLive
	case "test":
		return paypal.APIBaseSandBox
	default:
		return ""
	}
}
