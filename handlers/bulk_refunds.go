package handlers

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/chs.go/utils"
	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/go-playground/validator/v10"
)

const (
	paypalProvider = "paypal"
	govpayProvider = "govpay"
)

// HandleGovPayBulkRefund accepts a bulk refunds file and adds and updates
// refunds data to the DB
func HandleGovPayBulkRefund(w http.ResponseWriter, req *http.Request) {
	log.InfoR(req, "start POST request for gov pay bulk refunds")

	handleRefundFile(w, req, govpayProvider)
}

// HandlePayPalBulkRefund accepts a bulk refund file and adds and updates
// refunds data to the DB
func HandlePayPalBulkRefund(w http.ResponseWriter, req *http.Request) {
	log.InfoR(req, "start POST request for paypal bulk refunds")

	handleRefundFile(w, req, paypalProvider)
}

func handleRefundFile(w http.ResponseWriter, req *http.Request, paymentProvider string) {
	file, header, err := req.FormFile("file")
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error reading file from request: %w", err))
		m := utils.NewMessageResponse("error reading file from request")
		utils.WriteJSONWithStatus(w, req, m, http.StatusInternalServerError)
		return
	}
	defer closeFile(file)

	// Copy file to bytes buffer
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		log.ErrorR(req, fmt.Errorf("error opening file: %w", err))
		m := utils.NewMessageResponse("error opening file")
		utils.WriteJSONWithStatus(w, req, m, http.StatusInternalServerError)
		return
	}

	var batchRefund models.RefundBatch

	// Unmarshal file to RefundBatch struct
	err = xml.Unmarshal(buf.Bytes(), &batchRefund)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error parsing file: %w", err))
		m := utils.NewMessageResponse("error parsing file")
		utils.WriteJSONWithStatus(w, req, m, http.StatusInternalServerError)
		return
	}

	// Validate required fields in batch refund request
	v := validator.New()
	err = v.Struct(batchRefund)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error validating request: %w", err))
		m := utils.NewMessageResponse("error validating request")
		utils.WriteJSONWithStatus(w, req, m, http.StatusUnprocessableEntity)
		return
	}

	var validationErrors []string

	// Validate batch refund request data against data in DB
	switch paymentProvider {
	case paypalProvider:
	case govpayProvider:
		validationErrors, err = refundService.ValidateGovPayBatchRefund(req.Context(), batchRefund)
		if err != nil {
			log.ErrorR(req, err)
			m := utils.NewMessageResponse("error processing batch refund")
			utils.WriteJSONWithStatus(w, req, m, http.StatusInternalServerError)
			return
		}
		if len(validationErrors) > 0 {
			message := fmt.Sprintf("the batch refund has failed validation on the following: %s", strings.Join(validationErrors, ","))
			log.Debug(message)
			m := utils.NewMessageResponse(message)
			utils.WriteJSONWithStatus(w, req, m, http.StatusBadRequest)
			return
		}
	default:
		message := fmt.Sprintf("invalid payment provider: %s", paymentProvider)
		log.Debug(message)
		m := utils.NewMessageResponse(message)
		utils.WriteJSONWithStatus(w, req, m, http.StatusInternalServerError)
		return
	}

	userID, ok := req.Context().Value(helpers.ContextKeyUserID).(string)
	if !ok {
		log.ErrorR(req, fmt.Errorf("error user details not found in context"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = refundService.UpdateBatchRefund(req.Context(), batchRefund, header.Filename, userID)
	if err != nil {
		m := utils.NewMessageResponse("error updating request")
		utils.WriteJSONWithStatus(w, req, m, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
}

// HandleProcessPendingRefunds retrieves a list of payments in the DB with
// a refund-pending status and fires a request to the payment provider to
// issue a refund
func HandleProcessPendingRefunds(w http.ResponseWriter, req *http.Request) {
	log.InfoR(req, "Start POST request for processing pending refunds")

	errList := refundService.ProcessBatchRefund(req)

	var res []string

	for _, e := range errList {
		res = append(res, e.Error())
	}

	utils.WriteJSONWithStatus(w, req, strings.Join(res, ","), http.StatusAccepted)
}

func closeFile(file multipart.File) {
	err := file.Close()
	if err != nil {
		log.Error(fmt.Errorf("error closing file: %w", err))
	}
}

// HandleGetRefundStatuses retrieves payments that are pending refund
func HandleGetRefundStatuses(w http.ResponseWriter, req *http.Request) {

	log.InfoR(req, "start GET request for payments with pending refund statuses")
	pendingRefundPaymentSessions, err := refundService.GetPaymentsWithPendingRefundStatus()
	if err != nil {
		log.ErrorR(req, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(pendingRefundPaymentSessions)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.InfoR(req, "Successful GET request for payments with pending refund statuses")
}
