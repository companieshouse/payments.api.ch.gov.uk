package service

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
)

// CreateExternalPaymentJourney creates an external payment session with a Payment Provider that is given, e.g: GovPay
func (service *PaymentService) CreateExternalPaymentJourney(req *http.Request, paymentSession *models.PaymentResourceRest, paypalService PaypalPaymentProvider) (*models.ExternalPaymentJourney, ResponseType, error) {
	if paymentSession.Status != InProgress.String() {
		err := fmt.Errorf("payment session is not in progress")
		log.ErrorR(req, err)
		return nil, InvalidData, err
	}

	// Check that class of payment of each Cost Resource is equal else error
	err := validateClassOfPayment(&paymentSession.Costs)
	if err != nil {
		log.ErrorR(req, err)
		return nil, InvalidData, err
	}

	paymentJourney := &models.ExternalPaymentJourney{}
	var responseType ResponseType
	var nextURL string

	switch paymentSession.PaymentMethod {
	case "GovPay":

		gp := &GovPayService{PaymentService: *service}

		nextURL, responseType, err = gp.GenerateNextURLGovPay(req, paymentSession)
		if err != nil {
			err = fmt.Errorf("error communicating with GovPay: [%s]", err)
			log.ErrorR(req, err)
			return nil, Error, err
		}
		if nextURL == "" {
			err = fmt.Errorf("no NextURL returned from GovPay")
			log.ErrorR(req, err)
			return nil, Error, err
		}

	case "PayPal":
		nextURL, responseType, err = paypalService.CreatePaypalOrder(paymentSession)
		if err != nil {
			err = fmt.Errorf("error communicating with PayPal API: [%v]", err)
			log.ErrorR(req, err)
			return nil, Error, err
		}
		if nextURL == "" {
			err = fmt.Errorf("approve link not returned in paypal order response")
			log.ErrorR(req, err)
			return nil, Error, err
		}
	default:
		err := fmt.Errorf("payment method [%s] for resource [%s] not recognised", paymentSession.PaymentMethod, paymentSession.Links.Self)
		log.ErrorR(req, err)

		return nil, Error, err
	}

	paymentJourney.NextURL = nextURL

	return paymentJourney, responseType, nil

}

func validateClassOfPayment(costs *[]models.CostResourceRest) error {

	for i, cost := range *costs {
		//Loop through Class Of Payments on a single resource to check they're the same.
		for j, classOfPayment := range cost.ClassOfPayment {
			if classOfPayment[j] != classOfPayment[0] {
				return fmt.Errorf("Two or more class of payments are different on the same cost resource: [%v] ", cost.Description)
			}
		}

		//Check the Class Of Payments on separate resources are the same.
		if (*costs)[i].ClassOfPayment[0] != (*costs)[0].ClassOfPayment[0] {
			return fmt.Errorf("Two or more class of payments are different on the same transaction: [%v] and [%v] ", (*costs)[0].Description, (*costs)[i].Description)
		}
	}
	return nil
}
