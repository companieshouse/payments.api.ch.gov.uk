package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
)

// HandleCreatePaymentSession creates a payment session and returns a journey URL for the calling app to redirect to
func HandleCreatePaymentSession(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		log.ErrorR(req, fmt.Errorf("request body empty"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	requestDecoder := json.NewDecoder(req.Body)
	var incomingPaymentResourceRequest models.IncomingPaymentResourceRequest
	err := requestDecoder.Decode(&incomingPaymentResourceRequest)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("request body invalid: [%v]", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// once we've read and decoded request body call the payment service handle internal business logic
	paymentResource, responseType, err := paymentService.CreatePaymentSession(req, incomingPaymentResourceRequest)

	if err != nil {
		log.ErrorR(req, fmt.Errorf("error creating payment resource: [%v]", err), log.Data{"service_response_type": responseType.String()})
		switch responseType {
		case service.InvalidData:
			w.WriteHeader(http.StatusBadRequest)
			return
		case service.Error:
			w.WriteHeader(http.StatusInternalServerError)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// response body contains fully decorated REST model
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", paymentResource.Links.Journey)
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(paymentResource)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
		return
	}

	log.InfoR(req, "Successful POST request for new payment resource", log.Data{"payment_id": paymentResource.MetaData.ID, "status": http.StatusCreated})
}

// HandleGetPaymentSession retrieves the payment session from request context
func HandleGetPaymentSession(w http.ResponseWriter, req *http.Request) {

	// get payment resource from context, put there by PaymentAuthenticationInterceptor
	paymentSession, ok := req.Context().Value(helpers.ContextKeyPaymentSession).(*models.PaymentResourceRest)

	if !ok {
		log.ErrorR(req, fmt.Errorf("invalid PaymentResourceRest in request context"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Check if the payment session is expired
	isExpired, err := service.IsExpired(*paymentSession, &paymentService.Config)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error checking payment session expiry status: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if isExpired && paymentSession.Status != service.Paid.String() {
		paymentSession.Status = service.Expired.String()
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(paymentSession)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.InfoR(req, "Successfully GET request for payment resource: ", log.Data{"payment_id": paymentSession.MetaData.ID})
}

// HandlePatchPaymentSession patches and updates the payment session
func HandlePatchPaymentSession(w http.ResponseWriter, req *http.Request) {
	// get payment resource from context, put there by PaymentAuthenticationInterceptor
	paymentSession, ok := req.Context().Value(helpers.ContextKeyPaymentSession).(*models.PaymentResourceRest)
	if !ok {
		log.ErrorR(req, fmt.Errorf("invalid PaymentResourceRest in request context"))
		w.WriteHeader(http.StatusInternalServerError)
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
		log.ErrorR(req, fmt.Errorf("payment session has expired"))
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if req.Body == nil {
		log.ErrorR(req, fmt.Errorf("request body empty"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	requestDecoder := json.NewDecoder(req.Body)
	var PaymentResourceUpdateData models.PaymentResourceRest
	err = requestDecoder.Decode(&PaymentResourceUpdateData)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("request body invalid: [%v]", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if PaymentResourceUpdateData.PaymentMethod == "" && PaymentResourceUpdateData.Status == "" {
		log.ErrorR(req, fmt.Errorf("no valid fields for the patch request has been supplied for resource [%s]", paymentSession.MetaData.ID))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if PaymentResourceUpdateData.PaymentMethod == "" {
		log.ErrorR(req, fmt.Errorf("no valid fields for the patch request have been supplied for resource [%s]", paymentSession.MetaData.ID))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	responseType, err := paymentService.PatchPaymentSession(req, paymentSession.MetaData.ID, PaymentResourceUpdateData)

	if err != nil {
		log.ErrorR(req, fmt.Errorf("error patching payment resource: [%v]", err), log.Data{"service_response_type": responseType.String()})
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.InfoR(req, "Successful PATCH request for payment resource", log.Data{"payment_id": paymentSession.MetaData.ID, "status": http.StatusOK})
}

// HandleGetPaymentDetails retrieves the payment details from the external provider
func HandleGetPaymentDetails(externalPaymentSvc *service.ExternalPaymentProvidersService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// The payment session must be retrieved directly to enable access to metadata outside the data block
		paymentSession, ok := req.Context().Value(helpers.ContextKeyPaymentSession).(*models.PaymentResourceRest)
		if !ok {
			log.ErrorR(req, fmt.Errorf("invalid PaymentResourceRest in request context"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Get the state of a payment
		var statusResponse *models.PaymentDetails
		var responseType service.ResponseType
		var err error
		switch paymentSession.PaymentMethod {
		case "credit-card":
			statusResponse, responseType, err = externalPaymentSvc.GovPayService.GetPaymentDetails(paymentSession)
		case "PayPal":
			statusResponse, responseType, err = externalPaymentSvc.PayPalService.GetPaymentDetails(paymentSession)
		default:
			err := fmt.Errorf("payment method [%s] for resource [%s] not recognised", paymentSession.PaymentMethod, paymentSession.Links.Self)
			log.ErrorR(req, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err != nil {
			log.ErrorR(req, fmt.Errorf("error getting payment details from external provider: [%v]", err), log.Data{"service_response_type": responseType.String()})
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		err = json.NewEncoder(w).Encode(statusResponse)
		if err != nil {
			log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.InfoR(req, "Successful GET request for payment details: ", log.Data{"payment_id": paymentSession.MetaData.ID})
	})
}

// HandleCheckPaymentStatus checks the status of incomplete payments and processes appropriately if paid
func HandleCheckPaymentStatus(w http.ResponseWriter, req *http.Request) {
	log.InfoR(req, "received request to check payment statuses")

	incompletePayments, err := paymentService.GetIncompletePayments(&paymentService.Config)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error getting in-progress payments: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	updatedPayments := make([]models.PaymentResourceRest, 0)

	if len(*incompletePayments) == 0 {
		log.InfoR(req, "no in-progress payments found")
		err = json.NewEncoder(w).Encode(updatedPayments)
		if err != nil {
			log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	log.InfoR(req, fmt.Sprintf("%d in-progress payments found", len(*incompletePayments)))

	// call GovPay for each payment to check status
	for _, pendingPayment := range *incompletePayments {

		// we need to get session to include costs
		paymentSession, _, err := paymentService.GetPaymentSession(req, pendingPayment.MetaData.ID)
		if err != nil {
			log.ErrorR(req, fmt.Errorf("error getting payment session for paymentID [%s]: [%w]", pendingPayment.MetaData.ID, err))
			continue
		}
		finished, status, providerID, err := externalPaymentService.GovPayService.GetPaymentStatus(paymentSession)
		if err != nil {
			log.ErrorR(req, fmt.Errorf("error getting status for paymentID [%s]: [%w]", pendingPayment.MetaData.ID, err))
			continue
		}

		if !finished {
			log.InfoR(req, fmt.Sprintf("Payment [%s] not finished, skipping.", pendingPayment.MetaData.ID))
			continue
		}

		completedAt := time.Now().Truncate(time.Millisecond)

		if status == "paid" {
			// payment has been successful, continue processing
			err = handlePaymentMessage(pendingPayment.MetaData.ID)
			if err != nil {
				log.ErrorR(req, fmt.Errorf("error producing payment kafka message for paymentID [%s]: [%w]", pendingPayment.MetaData.ID, err))
				continue
			}
			log.InfoR(req, fmt.Sprintf("kafka message successfully published for paymentID [%s]", pendingPayment.MetaData.ID))
			paymentSession.Status = status
			paymentSession.CompletedAt = completedAt
			updatedPayments = append(updatedPayments, *paymentSession)
		}

		// update payment status in DB
		paymentUpdate := models.PaymentResourceRest{
			Status:      status,
			ProviderID:  providerID,
			CompletedAt: completedAt,
		}

		_, err = paymentService.PatchPaymentSession(req, pendingPayment.MetaData.ID, paymentUpdate)
		if err != nil {
			log.ErrorR(req, fmt.Errorf("error patching DB for paymentID [%s]: [%w]", pendingPayment.MetaData.ID, err))
		}
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(updatedPayments)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.InfoR(req, "finished checking payment statuses")
}
