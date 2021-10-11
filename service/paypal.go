package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/plutov/paypal/v4"
)

var client *paypal.Client

func GetPayPalClient(cfg config.Config) (*paypal.Client, error) {
	if client != nil {
		return client, nil
	}

	paypalAPIBase := getPayPalAPIBase(cfg.PaypalEnv)
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

// PayPalSDK is an interface for all the PayPal client methods that will be used
// in this service
type PayPalSDK interface {
	GetAccessToken(ctx context.Context) (*paypal.TokenResponse, error)
	CreateOrder(ctx context.Context, intent string, purchaseUnits []paypal.PurchaseUnitRequest, payer *paypal.CreateOrderPayer, appContext *paypal.ApplicationContext) (*paypal.Order, error)
	GetOrder(ctx context.Context, orderID string) (*paypal.Order, error)
	CaptureOrder(ctx context.Context, orderID string, captureOrderRequest paypal.CaptureOrderRequest) (*paypal.CaptureOrderResponse, error)
}

// PayPalService handles the specific functionality of integrating PayPal into Payment Sessions
type PayPalService struct {
	Client         PayPalSDK
	PaymentService PaymentService
}

// CheckPaymentProviderStatus checks the status of the payment with PayPal
func (pp *PayPalService) CheckPaymentProviderStatus(paymentResource *models.PaymentResourceRest) (*models.StatusResponse, ResponseType, error) {

	res, err := pp.Client.GetOrder(
		context.Background(),
		paymentResource.MetaData.ExternalPaymentStatusID,
	)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error checking payment status with PayPal: [%w]", err)
	}

	return &models.StatusResponse{Status: res.Status}, Success, nil
}

// CreatePaymentAndGenerateNextURL creates a PayPal session linked to the given payment session
func (pp *PayPalService) CreatePaymentAndGenerateNextURL(req *http.Request, paymentResource *models.PaymentResourceRest) (string, ResponseType, error) {

	log.TraceR(req, "performing PayPal request", log.Data{"company_number": paymentResource.CompanyNumber})

	id := paymentResource.MetaData.ID

	order, err := pp.Client.CreateOrder(
		context.Background(),
		paypal.OrderIntentCapture,
		[]paypal.PurchaseUnitRequest{
			{
				ReferenceID: id,
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
		return "", Error, fmt.Errorf("failed to correctly create paypal order - status is not CREATED")
	}

	var nextURL string
	var externalStatusURI string

	for _, link := range order.Links {
		if link.Rel == "approve" {
			nextURL = link.Href
		}
		if link.Rel == "self" {
			externalStatusURI = link.Href
		}
	}

	err = pp.PaymentService.StoreExternalPaymentStatusDetails(paymentResource.MetaData.ID, externalStatusURI, order.ID)
	if err != nil {
		return "", Error, fmt.Errorf("error storing PayPal external payment details for payment session: [%s]", err)
	}

	return nextURL, Success, nil
}

// GetPaymentDetails gets the details of a PayPal payment
func (pp *PayPalService) GetPaymentDetails(_ *models.PaymentResourceRest) (*models.PaymentDetails, ResponseType, error) {

	//TODO: Check the payment details with PayPal

	return nil, Success, nil
}

// GetRefundSummary gets refund summary of a PayPal payment
func (pp *PayPalService) GetRefundSummary(_ *http.Request, _ string) (*models.PaymentResourceRest, *models.RefundSummary, ResponseType, error) {

	//TODO: Get refund summary with PayPal

	return nil, nil, Success, nil
}

// CreateRefund creates a refund in PayPal
func (pp *PayPalService) CreateRefund(_ *models.PaymentResourceRest, _ *models.CreateRefundGovPayRequest) (*models.CreateRefundGovPayResponse, ResponseType, error) {

	//TODO: Create a refund with PayPal

	return nil, Success, nil
}

// GetRefundStatus gets refund status from PayPal
func (pp *PayPalService) GetRefundStatus(_ *models.PaymentResourceRest, _ string) (*models.GetRefundStatusGovPayResponse, ResponseType, error) {

	//TODO: Get refund status

	return nil, Success, nil
}

// CapturePayment captures the payment in PayPal
func (pp *PayPalService) CapturePayment(orderId string) (*paypal.CaptureOrderResponse, error) {
	res, err := pp.Client.CaptureOrder(
		context.Background(),
		orderId,
		paypal.CaptureOrderRequest{},
	)
	return res, err
}

func getPayPalAPIBase(env string) string {
	switch env {
	case "live":
		return paypal.APIBaseLive
	case "test":
		return paypal.APIBaseSandBox
	default:
		return ""
	}
}
