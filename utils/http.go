package utils

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
)

// ResponseResource is the object returned in an error case
type ResponseResource struct {
	Message string `json:"message"`
}

// NewMessageResponse - convenience function for creating a response resource
func NewMessageResponse(message string) *ResponseResource {
	return &ResponseResource{Message: message}
}

// WriteJSONWithStatus writes the interface as a json string with the supplied status.
func WriteJSONWithStatus(w http.ResponseWriter, r *http.Request, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		log.ErrorR(r, fmt.Errorf("error writing response: %v", err))
	}
}
