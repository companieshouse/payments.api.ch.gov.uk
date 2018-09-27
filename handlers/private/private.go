package private

import (
	"encoding/json"
	"fmt"
	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/data"
	"github.com/gorilla/pat"
	"net/http"
)

// Register defines the route mappings
func Register(r *pat.Router) {
	r.Post("/private/paymentjourney", createExternalPaymentJourney).Name("create-paymentjourney")
}

func createExternalPaymentJourney(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		log.ErrorR(req, fmt.Errorf("Request Body Empty"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	requestDecoder := json.NewDecoder(req.Body)
	var incomingExternalPaymentJourneyRequest data.IncomingExternalPaymentJourneyRequest
	if requestDecoder.Decode(&incomingExternalPaymentJourneyRequest) != nil {
		log.ErrorR(req, fmt.Errorf("Request Body Invalid"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if incomingExternalPaymentJourneyRequest.PaymentMethod != "GovPay" {
		log.ErrorR(req, fmt.Errorf("Payment Method not Recognised"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	paymentJourney := &data.ExternalPaymentJourney{}
	//TODO: Return next_url from GovPay, hardcoded at the moment
	paymentJourney.NextUrl = "http://gov.uk/paymentjourney"
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(paymentJourney)
	return

}
