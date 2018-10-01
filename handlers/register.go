package handlers

import "github.com/gorilla/pat"

// Register defines the route mappings
func Register(r *pat.Router) {
	r.Post("/private/paymentjourney", createExternalPaymentJourney).Name("create-paymentjourney")
	r.Get("/healthcheck", getHealthCheck).Name("get-healthcheck")
	r.Post("/payments", createPaymentSession).Name("create-payment")
}
