package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
)

// CreateExternalPaymentJourney creates an external payment session with a Payment Provider that is given, e.g: GovPay
func (service *PaymentService) CreateExternalPaymentJourney(w http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get(":payment_id")
	if id == "" {
		log.ErrorR(req, fmt.Errorf("payment id not supplied"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	paymentSession, httpStatus, err := service.getPaymentSession(id)
	if err != nil {
		w.WriteHeader(httpStatus)
		log.ErrorR(req, err)
		return
	}

	paymentJourney := &models.ExternalPaymentJourney{}

	switch paymentSession.PaymentMethod {
	case "GovPay":
		paymentJourney.NextURL, err = service.returnNextURLGovPay(paymentSession, id, &service.Config)
		if err != nil {
			log.ErrorR(req, fmt.Errorf("error communicating with GovPay: [%s]", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if paymentJourney.NextURL == "" {
			log.ErrorR(req, fmt.Errorf("no NextURL returned from GovPay"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = json.NewEncoder(w).Encode(paymentJourney)
		if err != nil {
			log.ErrorR(req, fmt.Errorf("error writing response: %s", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		log.InfoR(req, "Successfully started session with GovPay", log.Data{"payment_id": id, "status": http.StatusCreated})

		return

	default:
		log.ErrorR(req, fmt.Errorf("payment method, [%s], for resource [%s] not recognised", paymentSession.PaymentMethod, id))

		w.WriteHeader(http.StatusBadRequest)
	}
}

// decimalPayment will always be in the form XX.XX (e.g: 12.00) due to getTotalAmount converting to decimal with 2 fixed places right of decimal point.
func convertToPenceFromDecimal(decimalPayment string) (int, error) {
	pencePayment := strings.Replace(decimalPayment, ".", "", 1)
	return strconv.Atoi(pencePayment)
}
