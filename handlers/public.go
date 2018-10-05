package handlers

import (
	"net/http"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/gorilla/pat"
)

// Register defines the route mappings
func Register(r *pat.Router, cfg config.Config) {
	m := &dao.Mongo{
		URL: cfg.MongoDBURL,
	}
	p := &service.PaymentService{
		DAO: m,
	}
	r.Get("/healthcheck", healthCheck).Name("get-healthcheck")
	r.Post("/payments", p.CreatePaymentSession).Name("create-payment")
}

// Return a 200 response if service is running
func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
