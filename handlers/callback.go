package handlers

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/gorilla/mux"
)

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

	// The payment session must be retrieved directly to enable access to metadata outside the data block
	paymentSession, _, err := paymentService.GetPaymentSession(req, id)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error getting payment session: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if paymentSession == nil {
		log.ErrorR(req, fmt.Errorf("payment session not found. id: %s", id))
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Check if the payment session is expired
	isExpired, err := service.IsExpired(*paymentSession, &paymentService.Config)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error checking payment session expiry status: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if isExpired {
		// Set the status of the payment
		paymentSession.Status = service.Expired.String()
		responseType, err := paymentService.PatchPaymentSession(req, id, *paymentSession)
		if err != nil {
			log.ErrorR(req, fmt.Errorf("error setting payment status of expired payment session: [%v]", err))
			switch responseType {
			case service.Error:
				w.WriteHeader(http.StatusInternalServerError)
				return
			default:
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		log.ErrorR(req, fmt.Errorf("payment session has expired"))
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Ensure payment method matches endpoint
	if paymentSession.PaymentMethod != "GovPay" {
		log.ErrorR(req, fmt.Errorf("payment method, [%s], for resource [%s] not recognised", paymentSession.PaymentMethod, id))
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}

	// Get the state of a GovPay payment
	gp := &service.GovPayService{
		PaymentService: *paymentService,
	}
	responseType, statusResponse, err := gp.CheckProvider(paymentSession)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error getting payment status from govpay: [%v]", err))
		switch responseType {
		case service.Error:
			w.WriteHeader(http.StatusInternalServerError)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// Set the status of the payment
	paymentSession.Status = statusResponse.Status
	responseType, err = paymentService.PatchPaymentSession(req, id, *paymentSession)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error setting payment status: [%v]", err))
		switch responseType {
		case service.Error:
			w.WriteHeader(http.StatusInternalServerError)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// Prepare parameters needed for redirecting
	params := models.RedirectParams{
		State:  paymentSession.MetaData.State,
		Ref:    paymentSession.Reference,
		Status: paymentSession.Status,
	}

	log.InfoR(req, "Successfully Closed payment session", log.Data{"payment_id": id, "status": *statusResponse})

	produceKafkaMessage()
	redirectUser(w, req, paymentSession.MetaData.RedirectURI, params)
}
