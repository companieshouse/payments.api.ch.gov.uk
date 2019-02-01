// Package handlers defines the API endpoints.
package handlers

import (
	"github.com/companieshouse/chs.go/log"
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

	// Create subrouters. All routes except /callback need auth middleware, so router needs to be split up. This allows
	// per-subrouter middleware.

	// create-payment endpoint should not be intercepted by the paymentauth interceptor, so needs to be it's own subrouter
	rootPaymentRouter := mainRouter.PathPrefix("/payments").Subrouter()
	rootPaymentRouter.HandleFunc("", p.CreatePaymentSession).Methods("POST").Name("create-payment")

	// get-payment endpoint needs payment and user auth, so needs to be it's own subrouter
	getPaymentRouter := rootPaymentRouter.PathPrefix("/{payment_id}").Subrouter()
	getPaymentRouter.HandleFunc("", p.GetPaymentSessionFromRequest).Methods("GET").Name("get-payment")

	// All private endpoints need  payment and user auth, so needs to be it's own subrouter
	privateRouter := mainRouter.PathPrefix("/private").Subrouter()
	privateRouter.HandleFunc("/payments/{payment_id}", p.PatchPaymentSession).Methods("PATCH").Name("patch-payment")
	privateRouter.HandleFunc("/payments/{payment_id}/external-journey", p.CreateExternalPaymentJourney).Methods("POST").Name("create-external-payment-journey")

	// callback endpoints should not be intercepted by the paymentauth or userauth interceptors, so needs to be it's own subrouter
	callbackRouter := mainRouter.PathPrefix("/callback").Subrouter()
	callbackRouter.HandleFunc("/payments/govpay/{payment_id}", p.HandleGovPayCallback).Methods("GET").Name("handle-govpay-callback")

	// Set middleware for subrouters
	rootPaymentRouter.Use(interceptors.UserAuthenticationInterceptor, log.Handler)
	getPaymentRouter.Use(interceptors.PaymentAuthenticationInterceptor)
	privateRouter.Use(interceptors.UserAuthenticationInterceptor, interceptors.PaymentAuthenticationInterceptor, log.Handler)
	callbackRouter.Use(log.Handler)
}

func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
