package service

import (
	"context"
	"fmt"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"

	"github.com/companieshouse/chs.go/log"
	"github.com/plutov/paypal/v4"
)

var client *paypal.Client

func getPaypalClient(cfg config.Config) (*paypal.Client, error) {
	if client != nil {
		return client, nil
	}

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

	return c, nil
}

// PaypalPaymentProviderService is an Interface to enable mocking
type PaypalSDK interface {
	GetAccessToken(ctx context.Context) (*paypal.TokenResponse, error)
	CreateOrder(ctx context.Context, intent string, purchaseUnits []paypal.PurchaseUnitRequest, payer *paypal.CreateOrderPayer, appContext *paypal.ApplicationContext) (*paypal.Order, error)
}

type PaypalPaymentProvider interface {
	CreatePaypalOrder(paymentResource *models.PaymentResourceRest) (string, ResponseType, error)
}

// PayPalService handles the specific functionality of integrating PayPal into Payment Sessions
type PaypalService struct {
	Client         PaypalSDK
	PaymentService PaymentService
}

func NewPayPalService(cfg config.Config, paymentSvc PaymentService) (*PaypalService, error) {
	c, err := getPaypalClient(cfg)
	if err != nil {
		return nil, err
	}
	return &PaypalService{Client: c, PaymentService: paymentSvc}, nil
}

// CreatePaypalOrder creates a PayPal order to accept a payment
func (pp PaypalService) CreatePaypalOrder(paymentResource *models.PaymentResourceRest) (string, ResponseType, error) {

	order, err := pp.Client.CreateOrder(
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
