package service

import (
	"fmt"
	"net/http"
)

// redirectUser redirects user to the provided redirect_uri with query params
func redirectUser(w http.ResponseWriter, r *http.Request, redirectURI string, state string, ref string, status string) {
	// Redirect the user to the redirect_uri, passing the state, ref and status as query params
	generatedURL := fmt.Sprintf("%s?state=%s&ref=%s&status=%s", redirectURI, state, ref, status)
	http.Redirect(w, r, generatedURL, http.StatusSeeOther)
}

func produceKafkaMessage() {
	// TODO: Produce message to payment-processed topic
}
