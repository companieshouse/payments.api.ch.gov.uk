package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/plutov/paypal/v4"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/gorilla/mux"
)

// handlePaymentMessage allows us to mock the call to producePaymentMessage for unit tests
var handlePaymentMessage = producePaymentMessage

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
	if paymentSession.PaymentMethod != "credit-card" {
		log.ErrorR(req, fmt.Errorf("payment method, [%s], for resource [%s] not recognised", paymentSession.PaymentMethod, id))
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}

	// Get the state of a GovPay payment
	gp := &service.GovPayService{
		PaymentService: *paymentService,
	}
	statusResponse, responseType, err := gp.CheckPaymentProviderStatus(paymentSession)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error getting payment status from govpay: [%v]", err), log.Data{"service_response_type": responseType.String()})
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
	// To match the format time is saved to mongo, e.g. "2018-11-22T08:39:16.782Z", truncate the time
	paymentSession.CompletedAt = time.Now().Truncate(time.Millisecond)

	responseType, err = paymentService.PatchPaymentSession(req, id, *paymentSession)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error setting payment status: [%v]", err), log.Data{"service_response_type": responseType.String()})
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

	log.InfoR(req, "Successfully Closed payment session", log.Data{"payment_id": id, "status": paymentSession.Status})

	err = handlePaymentMessage(paymentSession.MetaData.ID)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error producing payment kafka message: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	redirectUser(w, req, paymentSession.MetaData.RedirectURI, params)
}

func HandlePayPalCallback(pp *service.PayPalService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		orderID := req.URL.Query().Get("token")

		order, err := pp.GetOrderDetails(orderID)
		if err != nil {
			log.ErrorR(req, fmt.Errorf("error getting order details: %v", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// The payment session must be retrieved directly to enable access to metadata outside the data block
		paymentID := order.PurchaseUnits[0].ReferenceID
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
			responseType, err := paymentService.PatchPaymentSession(req, paymentID, *paymentSession)
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
		if strings.ToLower(paymentSession.PaymentMethod) != "paypal" {
			log.ErrorR(req, fmt.Errorf("payment method, [%s], for resource [%s] not recognised", paymentSession.PaymentMethod, paymentID))
			w.WriteHeader(http.StatusPreconditionFailed)
			return
		}

		if order.Status != paypal.OrderStatusApproved {
			log.ErrorR(req, fmt.Errorf("error - paypal payment status not approved, status is: [%s]", order.Status))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = pp.CapturePayment(orderID)
		if err != nil {
			log.ErrorR(req, fmt.Errorf("error capturing payment: %v", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// TODO: uncomment when plutov accepts PR
		//captureStatus := response.PurchaseUnits[0].Payments.Captures[0].Status
		//if strings.ToLower(captureStatus) != "completed" {
		//	log.ErrorR(req, fmt.Errorf("error - paypal payment status not completed, status is: [%s]", captureStatus))
		//	w.WriteHeader(http.StatusInternalServerError)
		//	return
		//}
		// TODO handle unsuccessful capture statuses

		// Set the status of the payment
		paymentSession.Status = "completed"
		// To match the format time is saved to mongo, e.g. "2018-11-22T08:39:16.782Z", truncate the time
		paymentSession.CompletedAt = time.Now().Truncate(time.Millisecond)

		responseType, err := paymentService.PatchPaymentSession(req, paymentID, *paymentSession)
		if err != nil {
			log.ErrorR(req, fmt.Errorf("error setting payment status: [%v]", err), log.Data{"service_response_type": responseType.String()})
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
