package handlers

import (
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
	r.Post("/private/paymentjourney", service.CreateExternalPaymentJourney).Name("create-paymentjourney")
}
