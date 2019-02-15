package service

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
)

// CreateExternalPaymentJourney creates an external payment session with a Payment Provider that is given, e.g: GovPay
func (service *PaymentService) CreateExternalPaymentJourney(req *http.Request, paymentSession *models.PaymentResourceRest) (*models.ExternalPaymentJourney, error) {

	switch paymentSession.PaymentMethod {
	case "GovPay":
		paymentJourney := &models.ExternalPaymentJourney{}

		gp := &GovPayService{PaymentService: *service}

		nextURL, err := gp.GenerateNextURLGovPay(req, paymentSession)
		if err != nil {
			err = fmt.Errorf("error communicating with GovPay: [%s]", err)
			log.ErrorR(req, err)
			return nil, err
		}
		if nextURL == "" {
			err = fmt.Errorf("no NextURL returned from GovPay")
			log.ErrorR(req, err)
			return nil, err
		}
		paymentJourney.NextURL = nextURL

		return paymentJourney, nil

	default:
		err := fmt.Errorf("payment method [%s] for resource [%s] not recognised", paymentSession.PaymentMethod, paymentSession.Links.Self)
		log.ErrorR(req, err)

		return nil, err
	}
}
