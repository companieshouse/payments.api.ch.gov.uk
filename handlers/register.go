// Package handlers defines the API endpoints.
package handlers

import (
	"net/http"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/gorilla/mux"
)

// Register defines the route mappings
func Register(r *mux.Router, cfg config.Config) {
	m := &dao.Mongo{
		URL: cfg.MongoDBURL,
	}
	p := &service.PaymentService{
		DAO:    m,
		Config: cfg,
	}

	r.HandleFunc("/healthcheck", healthCheck).Methods("GET").Name("get-healthcheck")
	r.HandleFunc("/payments", p.CreatePaymentSession).Methods("POST").Name("create-payment")
	r.HandleFunc("/payments/{payment_id}", p.GetPaymentSession).Methods("GET").Name("get-payment")
	r.HandleFunc("/private/payments/{payment_id}", p.PatchPaymentSession).Methods("PATCH").Name("patch-payment")
	r.HandleFunc("/private/payments/{payment_id}/external-journey", p.CreateExternalPaymentJourney).Methods("POST").Name("create-external-payment-journey")
	r.HandleFunc("/callback/payments/govpay/{payment_id}", p.HandleGovPayCallback).Methods("GET").Name("handle-govpay-callback")
}

func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
