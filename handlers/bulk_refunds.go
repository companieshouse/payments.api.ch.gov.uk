package handlers

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"gopkg.in/go-playground/validator.v9"

	"github.com/companieshouse/chs.go/utils"

	"github.com/companieshouse/chs.go/log"
)

// HandleGovPayBulkRefund accepts a bulk refunds file and adds and updates
// refunds data to the DB
func HandleGovPayBulkRefund(w http.ResponseWriter, req *http.Request) {

	log.InfoR(req, "start POST request for bulk refunds")

	file, _, err := req.FormFile("file")
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

	var batchRefund models.BatchService

	// Unmarshal file to BatchService struct
	err = xml.Unmarshal(buf.Bytes(), &batchRefund)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error parsing file: %w", err))
		m := utils.NewMessageResponse("error parsing file")
		utils.WriteJSONWithStatus(w, req, m, http.StatusInternalServerError)
		return
	}

	v := validator.New()
	err = v.Struct(batchRefund)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error validating request: %w", err))
		m := utils.NewMessageResponse("error validating request")
		utils.WriteJSONWithStatus(w, req, m, http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
}

func closeFile(file multipart.File) {
	err := file.Close()
	if err != nil {
		log.Error(fmt.Errorf("error closing file: %w", err))
	}
}
