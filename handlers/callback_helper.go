package handlers

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
)

// redirectUser redirects user to the provided redirect_uri with query params
func redirectUser(w http.ResponseWriter, r *http.Request, redirectURI string, params models.RedirectParams) {
	// Redirect the user to the redirect_uri, passing the state, ref and status as query params
	req, err := http.NewRequest("GET", redirectURI, nil)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error redirecting user: [%s]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	query := req.URL.Query()
	query.Add("state", params.State)
	query.Add("ref", params.Ref)
	query.Add("status", params.Status)

	generatedURL := fmt.Sprintf("%s?%s", redirectURI, query.Encode())
	log.InfoR(r, "Redirecting to:", log.Data{"generated_url": generatedURL})

	http.Redirect(w, r, generatedURL, http.StatusSeeOther)
}

func produceKafkaMessage() {
	// TODO: Produce message to payment-processed topic
}
