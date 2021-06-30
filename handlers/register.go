package handlers

import (
	"errors"
	"net/http"
	"os"
	"regexp"

	"github.com/companieshouse/chs.go/authentication"
	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/interceptors"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/gorilla/mux"
)

var paymentService *service.PaymentService
var refundService *service.RefundService

// Register defines the route mappings for the main router and it's subrouters
func Register(mainRouter *mux.Router, cfg config.Config) {
	m := &dao.Mongo{
		URL: cfg.MongoDBURL,
	}

	r, err := regexp.Compile(cfg.SecureAppCostsRegex)
	if err != nil {
		err = errors.New("secure app costs regex failed to compile")
		log.Error(err)
		os.Exit(1)
	}

	paymentService = &service.PaymentService{
		DAO:              m,
		Config:           cfg,
		SecureCostsRegex: r,
	}

	govPayService := &service.GovPayService{PaymentService: *paymentService}

	refundService = &service.RefundService{
		GovPayService:  govPayService,
		PaymentService: paymentService,
		DAO:            m,
		Config:         cfg,
	}

	pa := &interceptors.PaymentAuthenticationInterceptor{
		Service: *paymentService,
	}

	userAuthInterceptor := &authentication.UserAuthenticationInterceptor{
		AllowAPIKeyUser:                true,
		RequireElevatedAPIKeyPrivilege: true,
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

	// payment-details endpoint needs it's own interceptor
	paymentDetailsRouter := mainRouter.PathPrefix("/private/payments/{payment_id}/payment-details").Subrouter()
	paymentDetailsRouter.HandleFunc("", HandleGetPaymentDetails).Methods("GET").Name("get-payment-details")

	// create-refund endpoint needs its own interceptor
	createRefundRouter := mainRouter.PathPrefix("/payments/{paymentId}/refunds").Subrouter()
	createRefundRouter.HandleFunc("", HandleCreateRefund).Methods("POST").Name("create-refund")

	// update-refund endpoint needs its own interceptor
	updateRefundRouter := mainRouter.PathPrefix("/payments/{paymentId}/refunds/{refundId}").Subrouter()
	updateRefundRouter.HandleFunc("", HandleUpdateRefund).Methods("PATCH").Name("update-refund")

	// All private endpoints need  payment and user auth, and due to router limitations of applying interceptors, need their own subrouters
	privatePatchRouter := mainRouter.PathPrefix("/private/payments/{payment_id}").Subrouter()
	privatePatchRouter.HandleFunc("", HandlePatchPaymentSession).Methods("PATCH").Name("patch-payment")

	privateJourneyRouter := mainRouter.PathPrefix("/private/payments/{payment_id}/external-journey").Subrouter()
	privateJourneyRouter.HandleFunc("", HandleCreateExternalPaymentJourney).Methods("POST").Name("create-external-payment-journey")

	// callback endpoints should not be intercepted by the paymentauth or userauth interceptors, so needs to be it's own subrouter
	callbackRouter := mainRouter.PathPrefix("/callback").Subrouter()
	callbackRouter.HandleFunc("/payments/govpay/{payment_id}", HandleGovPayCallback).Methods("GET").Name("handle-govpay-callback")

	// Set middleware for subrouters
	rootPaymentRouter.Use(log.Handler, interceptors.Oauth2OrPaymentPrivilegesIntercept, interceptors.UserPaymentAuthenticationIntercept)
	getPaymentRouter.Use(pa.PaymentAuthenticationIntercept)
	paymentDetailsRouter.Use(log.Handler, interceptors.InternalOrPaymentPrivilegesIntercept, pa.PaymentAuthenticationIntercept)
	createRefundRouter.Use(log.Handler, authentication.ElevatedPrivilegesInterceptor)
	updateRefundRouter.Use(log.Handler, authentication.ElevatedPrivilegesInterceptor)
	privatePatchRouter.Use(log.Handler, interceptors.UserPaymentAuthenticationIntercept, pa.PaymentAuthenticationIntercept)
	privateJourneyRouter.Use(log.Handler, userAuthInterceptor.UserAuthenticationIntercept, pa.PaymentAuthenticationIntercept)
	callbackRouter.Use(log.Handler)
}

func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
