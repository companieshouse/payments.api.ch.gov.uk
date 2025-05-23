package handlers

import (
	"errors"
	"fmt"
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
var externalPaymentService *service.ExternalPaymentProvidersService

// Register defines the route mappings for the main router and it's subrouters
func Register(mainRouter *mux.Router, cfg config.Config, paymentsDao dao.DAO) {
	r, err := regexp.Compile(cfg.SecureAppCostsRegex)
	if err != nil {
		err = errors.New("secure app costs regex failed to compile")
		log.Error(err)
		os.Exit(1)
	}

	paymentService = &service.PaymentService{
		DAO:              paymentsDao,
		Config:           cfg,
		SecureCostsRegex: r,
	}

	govPayService := &service.GovPayService{PaymentService: *paymentService}

	payPalClient, err := service.GetPayPalClient(cfg)
	if err != nil {
		log.Error(fmt.Errorf("error getting PayPal client: %v", err))
		os.Exit(1)
	}

	payPalService := &service.PayPalService{Client: payPalClient, PaymentService: *paymentService}

	externalPaymentService = &service.ExternalPaymentProvidersService{
		GovPayService: *govPayService,
		PayPalService: *payPalService,
	}

	refundService = &service.RefundService{
		GovPayService:  govPayService,
		PayPalService:  payPalService,
		PaymentService: paymentService,
		DAO:            paymentsDao,
		Config:         cfg,
	}

	pa := &interceptors.PaymentAuthenticationInterceptor{
		Service: *paymentService,
	}

	mainRouter.HandleFunc("/healthcheck", healthCheck).Methods("GET").Name("get-healthcheck")

	// Create subrouters. All routes except /callback need auth middleware, so router needs to be split up. This allows
	// per-subrouter middleware.

	// create-payment endpoint should not be intercepted by the paymentauth interceptor, so needs to be it's own subrouter
	createPaymentRouter := mainRouter.PathPrefix("/payments").Subrouter()
	createPaymentRouter.HandleFunc("", HandleCreatePaymentSession).Methods("POST").Name("create-payment")

	// get-payment endpoint needs payment and user auth, so needs to be it's own subrouter
	getPaymentRouter := mainRouter.PathPrefix("/payments/{payment_id}").Subrouter()
	getPaymentRouter.HandleFunc("", HandleGetPaymentSession).Methods("GET").Name("get-payment")

	// payment-details endpoint needs it's own interceptor
	paymentDetailsRouter := mainRouter.PathPrefix("/private/payments/{payment_id}/payment-details").Subrouter()
	paymentDetailsRouter.Handle("", HandleGetPaymentDetails(externalPaymentService)).Methods("GET").Name("get-payment-details")

	paymentStatusRouter := mainRouter.PathPrefix("/private/payments/status-check").Subrouter()
	paymentStatusRouter.HandleFunc("", HandleCheckPaymentStatus).Methods("POST").Name("check-payment-status")

	// create-refund endpoint needs its own interceptor
	createRefundRouter := mainRouter.PathPrefix("/payments/{paymentId}/refunds").Subrouter()
	createRefundRouter.HandleFunc("", HandleCreateRefund).Methods("POST").Name("create-refund")

	// get-refunds endpoint needs its own interceptor
	getRefundRouter := mainRouter.PathPrefix("/payments/{paymentId}/refunds").Subrouter()
	getRefundRouter.HandleFunc("", HandleGetRefunds).Methods("GET").Name("get-refunds")

	// update-refund endpoint needs its own interceptor
	updateRefundRouter := mainRouter.PathPrefix("/payments/{paymentId}/refunds/{refundId}").Subrouter()
	updateRefundRouter.HandleFunc("", HandleUpdateRefund).Methods("PATCH").Name("update-refund")

	refundRouter := mainRouter.PathPrefix("/payments/refunds").Subrouter()
	refundRouter.HandleFunc("/process-pending", HandleProcessPendingRefunds).Methods("POST").Name("process-pending-refunds")

	// All private endpoints need  payment and user auth, and due to router limitations of applying interceptors, need their own subrouters
	privatePatchRouter := mainRouter.PathPrefix("/private/payments/{payment_id}").Subrouter()
	privatePatchRouter.HandleFunc("", HandlePatchPaymentSession).Methods("PATCH").Name("patch-payment")

	privateJourneyRouter := mainRouter.PathPrefix("/private/payments/{payment_id}/external-journey").Subrouter()
	privateJourneyRouter.Handle("", HandleCreateExternalPaymentJourney(externalPaymentService)).Methods("POST").Name("create-external-payment-journey")

	// Admin router will handle all the routes with an admin prefix
	// and will be intercepted to check for the admin role
	adminRouter := mainRouter.PathPrefix("/admin/payments/bulk-refunds").Subrouter()
	adminRouter.HandleFunc("", HandleGetRefundStatuses).Methods("GET").Name("get-refund-statuses")
	adminRouter.HandleFunc("/govpay", HandleGovPayBulkRefund).Methods("POST").Name("bulk-refund-govpay")
	adminRouter.HandleFunc("/paypal", HandlePayPalBulkRefund).Methods("POST").Name("bulk-refund-paypal")
	adminRouter.HandleFunc("/process-pending", HandleProcessBulkPendingRefunds).Methods("POST").Name("process-bulk-refund")

	// callback endpoints should not be intercepted by the paymentauth or userauth interceptors, so needs to be it's own subrouter
	callbackRouter := mainRouter.PathPrefix("/callback").Subrouter()
	callbackRouter.Handle("/payments/govpay/{payment_id}", HandleGovPayCallback(govPayService)).Methods("GET").Name("handle-govpay-callback")
	callbackRouter.Handle("/payments/paypal/orders/{payment_id}", HandlePayPalCallback(payPalService)).Methods("GET").Name("handle-paypal-callback")

	// Set middleware for subrouters
	createPaymentRouter.Use(log.Handler, interceptors.Oauth2OrPaymentPrivilegesIntercept, interceptors.UserPaymentAuthenticationIntercept)
	getPaymentRouter.Use(interceptors.UserPaymentAuthenticationIntercept, pa.PaymentAuthenticationIntercept)
	paymentDetailsRouter.Use(log.Handler, interceptors.UserPaymentAuthenticationIntercept, interceptors.InternalOrPaymentPrivilegesIntercept, pa.PaymentAuthenticationIntercept)
	paymentStatusRouter.Use(log.Handler, interceptors.InternalOrPaymentPrivilegesIntercept)
	createRefundRouter.Use(log.Handler, authentication.ElevatedPrivilegesInterceptor)
	updateRefundRouter.Use(log.Handler, authentication.ElevatedPrivilegesInterceptor)
	refundRouter.Use(log.Handler, interceptors.InternalOrPaymentPrivilegesIntercept)
	privatePatchRouter.Use(log.Handler, interceptors.UserPaymentAuthenticationIntercept, pa.PaymentAuthenticationIntercept)
	privateJourneyRouter.Use(log.Handler, interceptors.UserPaymentAuthenticationIntercept, pa.PaymentAuthenticationIntercept)
	adminRouter.Use(log.Handler, interceptors.UserPaymentAuthenticationIntercept, interceptors.PaymentAdminAuthenticationIntercept)
	callbackRouter.Use(log.Handler)
}

func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
