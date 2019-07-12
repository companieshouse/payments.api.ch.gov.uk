package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
)

// HandleCreateExternalPaymentJourney creates an external payment session with a Payment Provider that is given, e.g. GOV.UK Pay
func HandleCreateExternalPaymentJourney(w http.ResponseWriter, req *http.Request) {
	// get payment resource from context, put there by PaymentAuthenticationInterceptor
	paymentSession, ok := req.Context().Value(helpers.ContextKeyPaymentSession).(*models.PaymentResourceRest)
	if !ok {
		log.ErrorR(req, fmt.Errorf("invalid PaymentResourceRest in request context"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	externalPaymentJourney, responseType, err := paymentService.CreateExternalPaymentJourney(req, paymentSession)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error creating external payment journey: [%v]", err), log.Data{"service_response_type": responseType.String()})
		switch responseType {
		case service.Error:
			w.WriteHeader(http.StatusInternalServerError)
			return
		case service.InvalidData:
			w.WriteHeader(http.StatusBadRequest)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(externalPaymentJourney)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error writing response: %s", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.InfoR(req, "Successfully started session with GOV.UK Pay", log.Data{"payment_id": paymentSession.MetaData.ID, "status": http.StatusCreated})
}
