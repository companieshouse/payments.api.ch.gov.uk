package handlers

import (
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/interceptors"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/gorilla/mux"
)

var paymentService *service.PaymentService

// Register defines the route mappings for the main router and it's subrouters
func Register(mainRouter *mux.Router, cfg config.Config) {
	m := &dao.Mongo{
		URL: cfg.MongoDBURL,
	}

	paymentService = &service.PaymentService{
		DAO:    m,
		Config: cfg,
	}

	pa := &interceptors.PaymentAuthenticationInterceptor{
		Service: *paymentService,
	}
	mainRouter.HandleFunc("/healthcheck", healthCheck).Methods("GET").Name("get-healthcheck")

	// Create subrouters. All routes except /callback need auth middleware, so router needs to be split up. This allows
	// per-subrouter middleware.

	// create-payment endpoint should not be intercepted by the paymentauth interceptor, so needs to be it's own subrouter
	rootPaymentRouter := mainRouter.PathPrefix("/payments").Subrouter()
	rootPaymentRouter.HandleFunc("", HandleCreatePaymentSession).Methods("POST").Name("create-payment")

	// get-payment endpoint needs payment and user auth, so needs to be it's own subrouter
	getPaymentRouter := rootPaymentRouter.PathPrefix("/{payment_id}").Subrouter()
	getPaymentRouter.HandleFunc("", HandleGetPaymentSession).Methods("GET").Name("get-payment")

	// All private endpoints need  payment and user auth, so needs to be it's own subrouter
	privateRouter := mainRouter.PathPrefix("/private").Subrouter()
	privateRouter.HandleFunc("/payments/{payment_id}", HandlePatchPaymentSession).Methods("PATCH").Name("patch-payment")
	privateRouter.HandleFunc("/payments/{payment_id}/external-journey", HandleCreateExternalPaymentJourney).Methods("POST").Name("create-external-payment-journey")

	// callback endpoints should not be intercepted by the paymentauth or userauth interceptors, so needs to be it's own subrouter
	callbackRouter := mainRouter.PathPrefix("/callback").Subrouter()
	callbackRouter.HandleFunc("/payments/govpay/{payment_id}", HandleGovPayCallback).Methods("GET").Name("handle-govpay-callback")

	// Set middleware for subrouters
	rootPaymentRouter.Use(log.Handler, interceptors.UserAuthenticationInterceptor)
	getPaymentRouter.Use(pa.PaymentAuthenticationIntercept)
	privateRouter.Use(log.Handler, interceptors.UserAuthenticationInterceptor, pa.PaymentAuthenticationIntercept)
	callbackRouter.Use(log.Handler)
}

func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
