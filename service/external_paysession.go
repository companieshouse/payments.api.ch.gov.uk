package service

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/davecgh/go-spew/spew"
)

// CreateExternalPaymentJourney creates an external payment session with a Payment Provider that is given, e.g: GovPay
func (service *PaymentService) CreateExternalPaymentJourney(req *http.Request, paymentSession *models.PaymentResourceRest) (*models.ExternalPaymentJourney, ResponseType, error) {
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

	switch paymentSession.PaymentMethod {
	case "GovPay":
		paymentJourney := &models.ExternalPaymentJourney{}

		gp := &GovPayService{PaymentService: *service}

		nextURL, responseType, err := gp.GenerateNextURLGovPay(req, paymentSession)
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
		paymentJourney.NextURL = nextURL

		return paymentJourney, responseType, nil

	case "PayPal":
		// TODO
		// paymentJourney := &models.ExternalPaymentJourney{}

		pp := &PayPalService{PaymentService: *service}

		// TODO get bearer token
		accessToken, err := pp.GetBearerToken()
		if err != nil {
			return nil, Error, err // TODO
		}
		spew.Dump(accessToken)
		return nil, Success, nil // TODO

	default:
		err := fmt.Errorf("payment method [%s] for resource [%s] not recognised", paymentSession.PaymentMethod, paymentSession.Links.Self)
		log.ErrorR(req, err)

		return nil, Error, err
	}
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
