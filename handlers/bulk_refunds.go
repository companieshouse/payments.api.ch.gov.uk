package handlers

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
)

// HandleBulkRefund accepts a bulk refunds file and adds and updates
// refunds data to the DB
func HandleBulkRefund(w http.ResponseWriter, req *http.Request) {

	log.InfoR(req, fmt.Sprintf("start POST request for bulk refunds"))

	_, _, err := req.FormFile("file")
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error reading form from request: %s", err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
}
