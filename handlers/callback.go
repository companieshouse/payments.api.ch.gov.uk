package handlers

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/gorilla/mux"

	"github.com/companieshouse/chs.go/log"
)

// moved this functionality entirely into a callback handler fun rather than being part of the payment service
// it uses several calls into payment service functionality but should not be part of the payment service itself

// HandleGovPayCallback handles the callback from Govpay and redirects the user
func HandleGovPayCallback(w http.ResponseWriter, req *http.Request) {
	// Get the payment session
	vars := mux.Vars(req)
	id := vars["payment_id"]
	if id == "" {
		log.ErrorR(req, fmt.Errorf("payment id not supplied"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// The PaymentResource must be retrieved directly to enable access to metadata outside the data block
	paymentResource, _, err := paymentService.GetPaymentSession(req, id)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error getting payment session: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if paymentResource == nil {
		log.ErrorR(req, fmt.Errorf("payment session not found. id: %s", id))
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Ensure payment method matches endpoint
	if paymentResource.PaymentMethod != "GovPay" {
		log.ErrorR(req, fmt.Errorf("payment method, [%s], for resource [%s] not recognised", paymentResource.PaymentMethod, id))
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}

	// Get the state of a GovPay payment
	gp := &service.GovPayService{
		PaymentService: *paymentService,
	}
	statusResponse, err := gp.CheckProvider(paymentResource)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error getting payment status from govpay: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Set the status of the payment
	// err = service.UpdatePaymentStatus(*statusResponse, *paymentResource)
	// changed to use PatchPaymentSession with specific data change rather than unnecessary service fun for updating status
	paymentResource.Status = statusResponse.Status
	err = paymentService.PatchPaymentSession(req, id, *paymentResource)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error setting payment status: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Prepare parameters needed for redirecting
	params := models.RedirectParams{
		State:  paymentResource.MetaData.State,
		Ref:    paymentResource.Reference,
		Status: paymentResource.Status,
	}

	produceKafkaMessage()
	redirectUser(w, req, paymentResource.MetaData.RedirectURI, params)
}
