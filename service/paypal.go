package service

import (
	"context"
	"fmt"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"

	"github.com/companieshouse/chs.go/log"
	"github.com/plutov/paypal/v4"
)

// PayPalService handles the specific functionality of integrating PayPal into Payment Sessions
type PayPalService struct {
	PaymentService PaymentService
	PayPalClient   *paypal.Client
}

func NewPayPalService(cfg *config.Config) (*PayPalService, error) {

	paypalAPIBase := getPaypalAPIBase(cfg.PaypalEnv)
	if paypalAPIBase == "" {
		return nil, fmt.Errorf("invalid paypal env in config: %s", cfg.PaypalEnv)
	}

	c, err := paypal.NewClient(cfg.PaypalClientID, cfg.PaypalSecret, paypalAPIBase)
	if err != nil {
		return nil, fmt.Errorf("error creating paypal client: [%v]", err)
	}
	_, err = c.GetAccessToken(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error getting access token: [%v]", err)
	}
	return &PayPalService{PayPalClient: c}, nil
}

// CreateOrder creates a PayPal order to accept a payment
func (pp PayPalService) CreateOrder(paymentResource *models.PaymentResourceRest) (string, ResponseType, error) {

	order, err := pp.PayPalClient.CreateOrder(
		context.Background(),
		paypal.OrderIntentCapture,
		[]paypal.PurchaseUnitRequest{
			{
				Amount: &paypal.PurchaseUnitAmount{
					Value:    paymentResource.Amount,
					Currency: "GBP",
				},
			},
		},
		nil,
		&paypal.ApplicationContext{
			ReturnURL: fmt.Sprintf("%s/callback/payments/paypal/orders/%s",
				pp.PaymentService.Config.PaymentsAPIURL, paymentResource.MetaData.ID),
		},
	)
	if err != nil {
		return "", Error, fmt.Errorf("error creating order: [%v]", err)
	}

	if order.Status != paypal.OrderStatusCreated {
		log.Debug(fmt.Sprintf("paypal order response status: %s", order.Status))
		return "", Error, fmt.Errorf("failed to correctly create paypal order")
	}

	var nextURL string
	for _, link := range order.Links {
		if link.Rel == "approve" {
			nextURL = link.Href
		}
	}

	return nextURL, Success, nil
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
