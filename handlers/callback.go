package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/gorilla/mux"
	"github.com/plutov/paypal/v4"
)

// handlePaymentMessage allows us to mock the call to producePaymentMessage for unit tests
var handlePaymentMessage = producePaymentMessage

// HandleGovPayCallback handles the callback from Govpay and redirects the user
func HandleGovPayCallback(gp service.PaymentProviderService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Get the payment session
		vars := mux.Vars(req)
		id := vars["payment_id"]
		if id == "" {
			log.ErrorR(req, fmt.Errorf("payment id not supplied"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log.InfoR(req, "Callback received from Gov Pay", log.Data{"payment_id": id})

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

		if paymentSession.Status == service.Paid.String() {
			log.ErrorR(req, fmt.Errorf("payment session is already paid. id: %s", id))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Check if the payment session is expired
		isExpired, err := service.IsExpired(*paymentSession, &paymentService.Config)
		if err != nil {
			log.ErrorR(req, fmt.Errorf("error checking payment session expiry status: [%v]", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Get the state of a GovPay payment
		statusResponse, providerID, responseType, err := gp.CheckPaymentProviderStatus(paymentSession)

		if err != nil {
			log.ErrorR(req, fmt.Errorf("error getting payment status from govpay: [%v]", err), log.Data{"service_response_type": responseType.String()})
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if isExpired && responseType != service.Success {
			// Set the status of the payment
			paymentSession.Status = service.Expired.String()
			_, err := paymentService.PatchPaymentSession(req, id, *paymentSession)
			if err != nil {
				log.ErrorR(req, fmt.Errorf("error setting payment status of expired payment session: [%v]", err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			log.ErrorR(req, fmt.Errorf("payment session has expired"))
			w.WriteHeader(http.StatusForbidden)
			return
		}

		// Ensure payment method matches endpoint
		if paymentSession.PaymentMethod != "credit-card" {
			log.ErrorR(req, fmt.Errorf("payment method, [%s], for resource [%s] not recognised", paymentSession.PaymentMethod, id))
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}

		// Set the Provider ID provided by Gov Pay
		paymentSession.ProviderID = providerID
		// Set the status of the payment
		paymentSession.Status = statusResponse.Status
		// only update 'completed_at' if payment marked as successful in GovPay response
		if responseType == service.Success {
			// To match the format time is saved to mongo, e.g. "2018-11-22T08:39:16.782Z", truncate the time
			paymentSession.CompletedAt = time.Now().Truncate(time.Millisecond)
		}

		patchResponseType, err := paymentService.PatchPaymentSession(req, id, *paymentSession)
		if err != nil {
			log.ErrorR(req, fmt.Errorf("error setting payment status: [%v]", err), log.Data{"service_response_type": patchResponseType.String()})
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Prepare parameters needed for redirecting
		params := models.RedirectParams{
			State:  paymentSession.MetaData.State,
			Ref:    paymentSession.Reference,
			Status: paymentSession.Status,
		}

		// Onl generate Kafka message if payment marked as successful in GovPay response
		if responseType == service.Success {
			log.InfoR(req, "Successfully Closed payment session", log.Data{"payment_id": id, "status": paymentSession.Status})

			err = handlePaymentMessage(paymentSession.MetaData.ID)
			if err != nil {
				log.ErrorR(req, fmt.Errorf("error producing payment kafka message: [%v]", err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		redirectUser(w, req, paymentSession.MetaData.RedirectURI, params)
	})
}

// HandlePayPalCallback handles the callback from PayPal and redirects the user
func HandlePayPalCallback(externalPaymentSvc service.PaymentProviderService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		// Get the payment session
		vars := mux.Vars(req)
		paymentID := vars["payment_id"]
		if paymentID == "" {
			log.ErrorR(req, fmt.Errorf("payment id not supplied"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log.InfoR(req, "Callback received from PayPal", log.Data{"payment_id": paymentID})

		// The payment session must be retrieved directly to enable access to metadata outside the data block
		paymentSession, _, err := paymentService.GetPaymentSession(req, paymentID)
		if err != nil {
			log.ErrorR(req, fmt.Errorf("error getting payment session: [%v]", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if paymentSession == nil {
			log.ErrorR(req, fmt.Errorf("payment session not found. id: %s", paymentID))
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if paymentSession.Status == service.Paid.String() {
			log.ErrorR(req, fmt.Errorf("payment session is already paid. id: %s", paymentID))
			w.WriteHeader(http.StatusBadRequest)
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
			_, err := paymentService.PatchPaymentSession(req, paymentID, *paymentSession)
			if err != nil {
				log.ErrorR(req, fmt.Errorf("error setting payment status of expired payment session: [%v]", err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			log.ErrorR(req, fmt.Errorf("payment session has expired"))
			w.WriteHeader(http.StatusForbidden)
			return
		}

		// Ensure payment method matches endpoint
		if !strings.EqualFold(paymentSession.PaymentMethod, "paypal") {
			log.ErrorR(req, fmt.Errorf("payment method, [%s], for resource [%s] not recognised", paymentSession.PaymentMethod, paymentID))
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}

		statusResponse, _, responseType, err := externalPaymentSvc.CheckPaymentProviderStatus(paymentSession)
		if err != nil {
			log.ErrorR(req, fmt.Errorf("error getting payment status from PayPal: [%w]", err), log.Data{"service_response_type": responseType.String()})
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if statusResponse.Status != paypal.OrderStatusApproved && statusResponse.Status != paypal.OrderStatusCreated {
			log.ErrorR(req, fmt.Errorf("error - paypal payment status not approved, status is: [%s]", statusResponse.Status))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// If order has been approved, then proceed to capture payment
		if statusResponse.Status == paypal.OrderStatusApproved {
			response, err := externalPaymentSvc.CapturePayment(paymentSession.MetaData.ExternalPaymentStatusID)
			if err != nil {
				log.ErrorR(req, fmt.Errorf("error capturing payment: %v", err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			captureStatus := response.PurchaseUnits[0].Payments.Captures[0].Status
			log.InfoR(req, fmt.Sprintf("Status of paypal capture is: [%s]", captureStatus))
			switch captureStatus {
			case "COMPLETED":
				paymentSession.Status = service.Paid.String()
			case "DECLINED":
				paymentSession.Status = service.NoFunds.String()
			default:
				paymentSession.Status = service.Failed.String()
			}

			// Add external transaction ID to paymentSession metadata
			paymentSession.MetaData.ExternalPaymentTransactionID = response.PurchaseUnits[0].Payments.Captures[0].ID
		}

		// If order status is created, then the payment has been cancelled
		if statusResponse.Status == paypal.OrderStatusCreated {
			paymentSession.Status = service.Failed.String()
		}

		// To match the format time is saved to mongo, e.g. "2018-11-22T08:39:16.782Z", truncate the time
		paymentSession.CompletedAt = time.Now().Truncate(time.Millisecond)

		responseType, err = paymentService.PatchPaymentSession(req, paymentID, *paymentSession)
		if err != nil {
			log.ErrorR(req, fmt.Errorf("error setting payment status: [%v]", err), log.Data{"service_response_type": responseType.String()})
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Prepare parameters needed for redirecting
		params := models.RedirectParams{
			State:  paymentSession.MetaData.State,
			Ref:    paymentSession.Reference,
			Status: paymentSession.Status,
		}

		log.InfoR(req, "Successfully Closed payment session", log.Data{"payment_id": paymentID, "status": paymentSession.Status})

		err = handlePaymentMessage(paymentSession.MetaData.ID)
		if err != nil {
			log.ErrorR(req, fmt.Errorf("error producing payment kafka message: [%v]", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		redirectUser(w, req, paymentSession.MetaData.RedirectURI, params)
	})
}
