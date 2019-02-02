package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	validator "gopkg.in/bluesuncorp/validator.v5"
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
	paymentResource, err := paymentService.CreatePaymentSession(req, incomingPaymentResourceRequest)

	if err != nil {
		log.ErrorR(req, fmt.Errorf("error creating payment resource: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// response body contains fully decorated REST model
	err = json.NewEncoder(w).Encode(paymentResource)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Add data to response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", paymentResource.Links.Journey)
	w.WriteHeader(http.StatusCreated)

	log.InfoR(req, "Successful POST request for new payment resource", log.Data{"payment_id": paymentResourceEntity.ID, "status": http.StatusCreated})
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
	// no need to check for id in path - this is already done by PaymentAuthenticationInterceptor
	// 	vars := mux.Vars(req)
	// 	id := vars["payment_id"]
	// 	if id == "" {
	// 		log.ErrorR(req, fmt.Errorf("payment id not supplied"))
	// 		w.WriteHeader(http.StatusBadRequest)
	// 		return
	// 	}

	// get payment resource from context, put there by PaymentAuthenticationInterceptor
	paymentSession, ok := req.Context().Value(service.PaymentSessionKey).(models.PaymentResourceRest)
	if !ok {
		log.ErrorR(req, fmt.Errorf("invalid PaymentResourceRest in request context"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// no need for a service call as the PaymentAuthenticationInterceptor has done the call, got the reosurce and stored it in request context already
	// paymentSession, httpStatus, err := (*PaymentService).GetPaymentSession(service, id)
	// if err != nil {
	// 	w.WriteHeader(httpStatus)
	// 	log.ErrorR(req, err)
	// 	return
	// }

	// paymentSessionResponse := transformers.PaymentTransformer{}.TransformToRest(*paymentSession)

	err := json.NewEncoder(w).Encode(paymentSession)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	log.InfoR(req, "Successfully GET request for payment resource: ", log.Data{"payment_id": id})
}
