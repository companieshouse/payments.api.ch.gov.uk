package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/gorilla/mux"
	"gopkg.in/go-playground/validator.v9"
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

	// Ideally all validation would be done in the service layer but due to different response status code here this is handled outside of service for now
	if err = validatePaymentCreate(incomingPaymentResourceRequest); err != nil {
		log.ErrorR(req, fmt.Errorf("invalid POST request to create payment session: [%v]", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// once we've read and decoded request body call the payment service handle internal business logic
	paymentResource, status, err := paymentService.CreatePaymentSession(req, incomingPaymentResourceRequest)

	if err != nil {
		log.ErrorR(req, fmt.Errorf("error creating payment resource: [%v]", err))
		w.WriteHeader(status)
		return
	}

	// response body contains fully decorated REST model
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", paymentResource.Links.Journey)
	w.WriteHeader(status)

	err = json.NewEncoder(w).Encode(paymentResource)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.InfoR(req, "Successful POST request for new payment resource", log.Data{"payment_id": paymentResource.MetaData.ID, "status": http.StatusCreated})
}

func validatePaymentCreate(incomingPaymentResourceRequest models.IncomingPaymentResourceRequest) error {
	validate := validator.New()
	err := validate.Struct(incomingPaymentResourceRequest)
	if err != nil {
		return err
	}

	// TODO ??? Feels like this func should be the place where we validate that the resource to be paid for lives on a whitelisted domain
	return nil
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

	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(paymentSession)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.InfoR(req, "Successfully GET request for payment resource: ", log.Data{"payment_id": paymentSession.MetaData.ID})
}

// HandlePatchPaymentSession patches and updates the payment session
func HandlePatchPaymentSession(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["payment_id"]
	if id == "" {
		log.ErrorR(req, fmt.Errorf("payment id not supplied"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.Body == nil {
		log.ErrorR(req, fmt.Errorf("request body empty"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	requestDecoder := json.NewDecoder(req.Body)
	var PaymentResourceUpdateData models.PaymentResourceRest
	err := requestDecoder.Decode(&PaymentResourceUpdateData)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("request body invalid: [%v]", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if PaymentResourceUpdateData.PaymentMethod == "" && PaymentResourceUpdateData.Status == "" {
		log.ErrorR(req, fmt.Errorf("no valid fields for the patch request has been supplied for resource [%s]", id))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if PaymentResourceUpdateData.PaymentMethod == "" {
		log.ErrorR(req, fmt.Errorf("no valid fields for the patch request have been supplied for resource [%s]", id))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = paymentService.PatchPaymentSession(id, PaymentResourceUpdateData)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error patching payment session: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.InfoR(req, "Successful PATCH request for payment resource", log.Data{"payment_id": id, "status": http.StatusOK})
}
