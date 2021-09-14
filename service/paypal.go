package service

import (
	"context"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/plutov/paypal/v4"
)

// PayPalService handles the specific functionality of integrating PayPal into Payment Sessions
type PayPalService struct {
	PaymentService PaymentService
}

func (pp PayPalService) GetBearerToken() (*paypal.TokenResponse, error) {
	c, err := paypal.NewClient("clientID", "secretID", paypal.APIBaseSandBox) // FIXME use proper credentials
	if err != nil {
		// TODO
		return nil, err
	}

	spew.Dump(c)

	c.SetLog(os.Stdout) // Set log to terminal stdout

	accessToken, err := c.GetAccessToken(context.Background())
	if err != nil {
		return nil, err // TODO
	}
	return accessToken, nil

}
