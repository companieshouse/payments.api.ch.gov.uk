package service

import (
	"net/http"

	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/plutov/paypal/v4"
)

// PaymentProviderService is an Interface for all the requests to external payment providers
type PaymentProviderService interface {
	CheckPaymentProviderStatus(paymentResource *models.PaymentResourceRest) (*models.StatusResponse, string, ResponseType, error)
	CreatePaymentAndGenerateNextURL(req *http.Request, paymentResource *models.PaymentResourceRest) (string, ResponseType, error)
	GetPaymentDetails(paymentResource *models.PaymentResourceRest) (*models.PaymentDetails, ResponseType, error)
	CapturePayment(id string) (*paypal.CaptureOrderResponse, error)
	GetCapturedPaymentDetails(id string) (*paypal.CaptureDetailsResponse, error)
	RefundCapture(captureID string) (*paypal.RefundResponse, error)
	GetRefundSummary(req *http.Request, id string) (*models.PaymentResourceRest, *models.RefundSummary, ResponseType, error)
	GetRefundStatus(paymentResource *models.PaymentResourceRest, refundId string) (*models.CreateRefundGovPayResponse, ResponseType, error)
	CreateRefund(paymentResource *models.PaymentResourceRest, refundRequest *models.CreateRefundGovPayRequest) (*models.CreateRefundGovPayResponse, ResponseType, error)
}
