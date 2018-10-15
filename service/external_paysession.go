package service

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/companieshouse/payments.api.ch.gov.uk/models"

	"github.com/companieshouse/chs.go/log"
)

// CreateExternalPaymentJourney creates an external payment session with a Payment Provider that is given, e.g: GovPay
func CreateExternalPaymentJourney(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		log.ErrorR(req, fmt.Errorf("request body empty"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	requestDecoder := json.NewDecoder(req.Body)
	var incomingExternalPaymentJourneyRequest models.IncomingExternalPaymentJourneyRequest
	if requestDecoder.Decode(&incomingExternalPaymentJourneyRequest) != nil {
		log.ErrorR(req, fmt.Errorf("request body invalid: %v", incomingExternalPaymentJourneyRequest))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if incomingExternalPaymentJourneyRequest.PaymentMethod != "GovPay" {
		log.ErrorR(req, fmt.Errorf("payment method not recognised: %v", incomingExternalPaymentJourneyRequest.PaymentMethod))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	paymentJourney := &models.ExternalPaymentJourney{}
	//TODO: Return next_url from GovPay, hardcoded at the moment
	paymentJourney.NextURL = "http://gov.uk/paymentjourney"
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(paymentJourney)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
