package service

import (
	"fmt"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"net/http"
	"os"

	"github.com/companieshouse/chs.go/log"
)

// redirectUser redirects user to the provided redirect_uri with query params
func redirectUser(w http.ResponseWriter, r *http.Request, redirectURI string, state string, ref string, status string) {
	// Redirect the user to the redirect_uri, passing the state, ref and status as query params
	req, err := http.NewRequest("GET", redirectURI, nil)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error redirecting user: [%s]", err))
		os.Exit(1)
	}
	query := req.URL.Query()
	query.Add("state", state)
	query.Add("ref", ref)
	query.Add("status", status)

	generatedURL := fmt.Sprintf("%s?%s", redirectURI, query.Encode())
	log.InfoR(r, "Redirecting to:", log.Data{"generated_url": generatedURL})

	http.Redirect(w, r, generatedURL, http.StatusSeeOther)
}

func produceKafkaMessage() {
	// TODO: Produce message to payment-processed topic
}

func (service *PaymentService) UpdatePaymentStatus(s models.StatusResponse, p models.PaymentResource) error {
	p.Data.Status = s.Status
	_, err := service.patchPaymentSession(p.ID, p)

	if err != nil {
		return fmt.Errorf("error updating payment status: [%s]", err)
	}
	return nil
}
