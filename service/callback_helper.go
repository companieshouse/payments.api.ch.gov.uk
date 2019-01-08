package service

import (
	"fmt"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/davecgh/go-spew/spew"
	"log"
	"net/http"
	"os"
)

// redirectUser redirects user to the provided redirect_uri with query params
func redirectUser(w http.ResponseWriter, r *http.Request, redirectURI string, state string, ref string, status string) {
	// Redirect the user to the redirect_uri, passing the state, ref and status as query params
	req, err := http.NewRequest("GET", redirectURI, nil)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	query := req.URL.Query()
	query.Add("state", state)
	query.Add("ref", ref)
	query.Add("status", status)
	//req.URL.RawQuery = query.Encode()

	generatedURL := fmt.Sprintf("%s?%s", redirectURI, query.Encode())
	spew.Dump(generatedURL)
	http.Redirect(w, r, generatedURL, http.StatusSeeOther)
}

func produceKafkaMessage() {
	// TODO: Produce message to payment-processed topic
}

func (service *PaymentService) UpdatePaymentStatus(s models.StatusResponse, p models.PaymentResource) {
	
}
