package handlers

import (
	"bytes"
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

// HandleGovPayBulkRefund accepts a bulk refunds file and adds and updates
// refunds data to the DB
func HandleGovPayBulkRefund(w http.ResponseWriter, req *http.Request) {

	log.InfoR(req, "start POST request for bulk refunds")

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

	var batchRefund models.GovPayRefundBatch

	// Unmarshal file to GovPayRefundBatch struct
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

	// Validate batch refund request data against data in DB
	validationErrors, err := refundService.ValidateGovPayBatchRefund(req.Context(), batchRefund)
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

	userID, ok := req.Context().Value(helpers.ContextKeyUserID).(string)
	if !ok {
		log.ErrorR(req, fmt.Errorf("error user details not found in context"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = refundService.UpdateGovPayBatchRefund(req.Context(), batchRefund, header.Filename, userID)
	if err != nil {
		m := utils.NewMessageResponse("error updating request")
		utils.WriteJSONWithStatus(w, req, m, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
}

// HandleProcessPendingRefund retrieves a list of payments in the DB with
// a refund-pending status and fires a request to the payment provider to
// issue a refund
func HandleProcessPendingRefunds(w http.ResponseWriter, req *http.Request) {
	log.InfoR(req, "Start POST request for processing pending refunds")

	payments, err := refundService.DAO.GetPaymentsWithRefundStatus()
	if err != nil {
		log.ErrorR(req, err)
		m := utils.NewMessageResponse("error retrieving payments with refund-pending status")
		utils.WriteJSONWithStatus(w, req, m, http.StatusInternalServerError)
		return
	}
	if len(payments) == 0 {
		log.InfoR(req, "no payments with refund-pending status found")
		m := utils.NewMessageResponse("no payments with refund-pending status found")
		utils.WriteJSONWithStatus(w, req, m, http.StatusNotFound)
		return
	}

	utils.WriteJSONWithStatus(w, req, payments, http.StatusOK)
}

func closeFile(file multipart.File) {
	err := file.Close()
	if err != nil {
		log.Error(fmt.Errorf("error closing file: %w", err))
	}
}
