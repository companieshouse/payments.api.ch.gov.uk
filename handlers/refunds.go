package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/gorilla/mux"
	"net/http"
)

// handleRefundMessage allows us to mock the call to produceRefundMessage for unit tests
var handleRefundMessage = produceRefundMessage

// HandleCreateRefund initiates a refund from the external provider
func HandleCreateRefund(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		log.ErrorR(req, fmt.Errorf("request body empty"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id := mux.Vars(req)["paymentId"]

	if id == "" {
		log.ErrorR(req, fmt.Errorf("payment id not supplied"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	requestDecoder := json.NewDecoder(req.Body)
	var incomingRefundResourceRequest models.CreateRefundRequest
	err := requestDecoder.Decode(&incomingRefundResourceRequest)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("request body invalid: [%v]", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// once we've read and decoded request body call the refund service handle internal business logic
	paymentResource, refund, responseType, err := refundService.CreateRefund(req, id, incomingRefundResourceRequest)

	if err != nil {
		log.ErrorR(req, fmt.Errorf("error creating refund resource: [%v]", err), log.Data{"service_response_type": responseType.String()})
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(refund)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
		return
	}

	log.InfoR(req, "Successful POST request for new refund", log.Data{"refund_id": refund.RefundId, "status": http.StatusCreated})

	err = handleRefundMessage(paymentResource.MetaData.ID, refund.RefundId)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error producing refund kafka message: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
