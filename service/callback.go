package service

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/gorilla/mux"

	"github.com/companieshouse/chs.go/log"
)

// HandleGovPayCallback handles the callback from Govpay and redirects the user
func (service *PaymentService) HandleGovPayCallback(w http.ResponseWriter, req *http.Request) {
	// Get the payment session
	vars := mux.Vars(req)
	id := vars["payment_id"]
	if id == "" {
		log.ErrorR(req, fmt.Errorf("payment id not supplied"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// The PaymentResource must be retrieved directly to enable access to metadata outside the data block
	paymentResource, err := service.DAO.GetPaymentResource(id)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error getting payment resource from db: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if paymentResource == nil {
		log.ErrorR(req, fmt.Errorf("payment session not found. id: %s", id))
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Ensure payment method matches endpoint
	if paymentResource.Data.PaymentMethod != "GovPay" {
		log.ErrorR(req, fmt.Errorf("payment method, [%s], for resource [%s] not recognised", paymentResource.Data.PaymentMethod, id))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get the state of a GovPay payment
	statusResponse, err := GovPayResponse.checkProvider(GovPayResponse{}, paymentResource)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error getting payment status from govpay: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Set the status of the payment
	err = service.UpdatePaymentStatus(*statusResponse, *paymentResource)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error setting payment status: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Prepare parameters needed for redirecting
	params := models.RedirectParams{State: paymentResource.State, Ref: paymentResource.Data.Reference, Status: paymentResource.Data.Status}

	log.InfoR(req, "Successfully Closed payment session", log.Data{"payment_id": id, "status": *statusResponse})

	produceKafkaMessage()
	redirectUser(w, req, paymentResource.RedirectURI, params)
}
