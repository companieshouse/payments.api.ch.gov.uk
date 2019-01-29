// Package handlers defines the API endpoints.
package handlers

import (
	"github.com/companieshouse/payments.api.ch.gov.uk/interceptors"
	"net/http"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/gorilla/mux"
)

// Register defines the route mappings for the main router and it's subrouters
func Register(mainRouter *mux.Router, cfg config.Config) {
	m := &dao.Mongo{
		URL: cfg.MongoDBURL,
	}
	p := &service.PaymentService{
		DAO:    m,
		Config: cfg,
	}

	mainRouter.HandleFunc("/healthcheck", healthCheck).Methods("GET").Name("get-healthcheck")

	// Create subrouters. All routes except /callback need auth middleware, so router needs to be split up.
	paymentsRouter := mainRouter.PathPrefix("/payments").Subrouter()
	paymentsRouter.HandleFunc("", p.CreatePaymentSession).Methods("POST").Name("create-payment")
	paymentsRouter.HandleFunc("/{payment_id}", p.GetPaymentSession).Methods("GET").Name("get-payment")

	privateRouter := mainRouter.PathPrefix("/private").Subrouter()
	privateRouter.HandleFunc("/payments/{payment_id}", p.PatchPaymentSession).Methods("PATCH").Name("patch-payment")
	privateRouter.HandleFunc("/payments/{payment_id}/external-journey", p.CreateExternalPaymentJourney).Methods("POST").Name("create-external-payment-journey")

	// Set middleware for subrouters
	paymentsRouter.Use(interceptors.UserAuthenticationInterceptor, interceptors.PaymentAuthenticationInterceptor)
	privateRouter.Use(interceptors.UserAuthenticationInterceptor, interceptors.PaymentAuthenticationInterceptor)
}

func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
