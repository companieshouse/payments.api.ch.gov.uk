package service

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/companieshouse/payments.api.ch.gov.uk/models"

	"github.com/companieshouse/chs.go/log"
)

// CreateExternalPaymentJourney creates an external payment session with a Payment Provider that is given, e.g: GovPay
func (service *PaymentService) CreateExternalPaymentJourney(w http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get(":payment_id")
	if id == "" {
		log.ErrorR(req, fmt.Errorf("payment id not supplied"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	paymentSession, httpStatus, err := (*PaymentService).getPaymentSession(service, id)
	if err != nil {
		w.WriteHeader(httpStatus)
		log.ErrorR(req, err)
		return
	}

	paymentJourney := &models.ExternalPaymentJourney{}

	if paymentSession.PaymentMethod == "GovPay" {
		paymentJourney.NextURL, err = returnNextURLGovPay(paymentSession, id, &service.Config)
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
		return
	}

	log.ErrorR(req, fmt.Errorf("payment method, [%s], for resource [%s] not recognised", paymentSession.PaymentMethod, id))
	w.WriteHeader(http.StatusBadRequest)
	return

}
