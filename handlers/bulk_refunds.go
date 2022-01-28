package handlers

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/utils"

	"github.com/companieshouse/chs.go/log"
)

// HandleGovPayBulkRefund accepts a bulk refunds file and adds and updates
// refunds data to the DB
func HandleGovPayBulkRefund(w http.ResponseWriter, req *http.Request) {

	log.InfoR(req, "start POST request for bulk refunds")

	_, _, err := req.FormFile("file")
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error reading file from request: %w", err))
		m := utils.NewMessageResponse("error reading file from request")
		utils.WriteJSONWithStatus(w, req, m, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
}
