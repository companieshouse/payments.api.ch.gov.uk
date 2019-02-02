package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
)

// HandleCreateExternalPaymentJourney creates an external payment session with a Payment Provider that is given, e.g: GovPay
func HandleCreateExternalPaymentJourney(w http.ResponseWriter, req *http.Request) {
	// vars := mux.Vars(req)
	// id := vars["payment_id"]
	// if id == "" {
	// 	log.ErrorR(req, fmt.Errorf("payment id not supplied"))
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	return
	// }

	// paymentSession, httpStatus, err := service.GetPaymentSession(id)
	// if err != nil {
	// 	w.WriteHeader(httpStatus)
	// 	log.ErrorR(req, err)
	// 	return
	// }

	// get payment resource from context, put there by PaymentAuthenticationInterceptor
	paymentSession, ok := req.Context().Value(helpers.ContextKeyPaymentSession).(models.PaymentResourceRest)
	if !ok {
		log.ErrorR(req, fmt.Errorf("invalid PaymentResourceRest in request context"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	externalPaymentJourney, err := paymentService.CreateExternalPaymentJourney(req, &paymentSession)
	if err != nil {
		err = fmt.Errorf("error creating external payment journey: %s", err)
		log.ErrorR(req, err)
		return
	}

	err = json.NewEncoder(w).Encode(externalPaymentJourney)
	if err != nil {
		err = fmt.Errorf("error writing response: %s", err)
		log.ErrorR(req, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	log.InfoR(req, "Successfully started session with GovPay", log.Data{"payment_id": paymentSession.MetaData.ID, "status": http.StatusCreated})

}
