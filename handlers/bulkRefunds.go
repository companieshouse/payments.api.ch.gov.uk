package handlers

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
)

// HandleBulkRefund
func HandleBulkRefund(w http.ResponseWriter, req *http.Request) {

	log.InfoR(req, fmt.Sprintf("start POST request for bulk refunds"))

	fmt.Print("printing request body .........")
	fmt.Print(req.Body)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
}
