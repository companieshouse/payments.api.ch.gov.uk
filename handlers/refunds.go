package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/gorilla/mux"
)

const (
	contentType                 = "Content-Type"
	applicationJsonResponseType = "application/json"
	writingErrorResponse        = "error writing response: %v"
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
		case service.NotFound:
			w.WriteHeader(http.StatusNotFound)
			return
		case service.Forbidden:
			w.WriteHeader(http.StatusForbidden)
			return
		case service.Conflict:
			w.WriteHeader(http.StatusConflict)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set(contentType, applicationJsonResponseType)
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(refund)
	if err != nil {
		log.ErrorR(req, fmt.Errorf(writingErrorResponse, err))
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

// HandleUpdateRefund fetches refund from the external provider and updates the status in Payments MongoDB
func HandleUpdateRefund(w http.ResponseWriter, req *http.Request) {
	paymentId := mux.Vars(req)["paymentId"]
	refundId := mux.Vars(req)["refundId"]

	if paymentId == "" {
		log.ErrorR(req, fmt.Errorf("payment id not supplied"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if refundId == "" {
		log.ErrorR(req, fmt.Errorf("refund id not supplied"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	refund, responseType, err := refundService.UpdateRefund(req, paymentId, refundId)

	if err != nil {
		log.ErrorR(req, fmt.Errorf("error updating refund resource: [%v]", err), log.Data{"service_response_type": responseType.String()})
		switch responseType {
		case service.InvalidData:
			w.WriteHeader(http.StatusBadRequest)
			return
		case service.NotFound:
			w.WriteHeader(http.StatusNotFound)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set(contentType, applicationJsonResponseType)
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(refund)
	if err != nil {
		log.ErrorR(req, fmt.Errorf(writingErrorResponse, err))
		return
	}

	log.InfoR(req, "Successful PATCH request for refund", log.Data{"refund_id": refundId, "status": http.StatusOK})
}

// HandleProcessPendingRefunds checks the status of each pending refund and updates the status for any completed refunds.
func HandleProcessPendingRefunds(w http.ResponseWriter, req *http.Request) {
	payments, responseType, errList := refundService.ProcessPendingRefunds(req)

	if errList != nil {
		log.ErrorR(req, fmt.Errorf("error retrieving payments with pending refunds: %v", errList))
		switch responseType {
		case service.Error:
			w.WriteHeader(http.StatusBadRequest)
			return
		case service.Success:
			w.WriteHeader(http.StatusOK)
			return
		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set(contentType, applicationJsonResponseType)
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(payments)
	if err != nil {
		log.ErrorR(req, fmt.Errorf(writingErrorResponse, err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.InfoR(req, "Successful processed pending refunds request", log.Data{"status": http.StatusOK})

	processKafkaSendMessage(req, payments)

}

func processKafkaSendMessage(req *http.Request, payments []models.PaymentResourceDB) {
	for _, i := range payments {
		x := i
		refund := x.Refunds[0]
		err := handleRefundMessage(x.ID, refund.RefundId)
		if err != nil {
			log.ErrorR(req, fmt.Errorf("error producing refund kafka message: [%v]", err))
		}
	}

}
