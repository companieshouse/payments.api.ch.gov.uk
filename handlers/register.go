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
		DAO:    m,
		Config: cfg,
	}

	r.Get("/healthcheck", healthCheck).Name("get-healthcheck")
	r.Post("/payments", p.CreatePaymentSession).Name("create-payment")
	r.Get("/payments/{payment_id}", p.GetPaymentSession).Name("get-payment")
	r.Patch("/private/payments/{payment_id}", p.PatchPaymentSession).Name("patch-payment")
	r.Post("/private/payments/{payment_id}/external-journey", p.CreateExternalPaymentJourney).Name("create-external-payment-journey")
	//r.Get("/private/govpay/{payment_id}/external-journey-status", p.FinishGovPayJourney).Name("get-external-payment-journey-status")
	//r.Get("/private/payments/{payment_id}/external-journey-status", p.FinishGovPayJourney).Name("get-external-payment-journey-status")
	r.Get("/private/callback/govpay/{payment_id}", p.FinishGovPayJourney).Name("finish-govpay-journey")
}

func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
