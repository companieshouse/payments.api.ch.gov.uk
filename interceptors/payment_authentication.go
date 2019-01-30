package interceptors

import (
	"fmt"
	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/gorilla/mux"
	"net/http"
)

func PaymentAuthenticationInterceptor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for a payment ID in request
		vars := mux.Vars(r)
		id := vars["payment_id"]
		if id == "" {
			log.ErrorR(r, fmt.Errorf("PaymentAuthenticationInterceptor error: no payment id"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		cfg, err := config.Get()
		if err != nil {
			log.Error(fmt.Errorf("error retreiving config: %s. Exiting", err), nil)
			return
		}

		m := &dao.Mongo{
			URL: cfg.MongoDBURL,
		}

		p := service.PaymentService{
			DAO:    m,
			Config: *cfg,
		}

		// Get the payment session form the ID in request
		paymentSession, httpStatus, err := p.GetPaymentSession(id)
		if err != nil {
			log.Error(fmt.Errorf("PaymentAuthenticationInterceptor not found: payment session found"))
			w.WriteHeader(httpStatus)
			return
		}

		// Get user details from request
		authorisedUser := helpers.GetAuthorisedIdentity(r)
		if authorisedUser == "" {
			log.Error(fmt.Errorf("PaymentAuthenticationInterceptor unauthorised: no authorised identity"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Set up variables that are used to determine authorisation below
		isGetRequest := http.MethodGet == r.Method
		authUserIsPaymentCreator := authorisedUser == paymentSession.CreatedBy.ID
		authUserHasPaymentLookupRole := helpers.IsRoleAuthorised(r, helpers.AdminPaymentLookupRole)

		// Now that we have the payment data and authorized user there are
		// multiple cases that can be allowed through:

		if authUserIsPaymentCreator && isGetRequest {
			// 1) Authorized user created the payment and request is a GET i.e.
			// to see payment data but not modify/delete
			log.InfoR(r, "PaymentAuthenticationInterceptor authorised as creator on GET")
			// Call the next handler
			next.ServeHTTP(w, r)
		} else if authUserHasPaymentLookupRole && isGetRequest {
			// 2) Authorized user has permission to lookup any payment session and
			// request is a GET i.e. to see payment data but not modify/delete
			log.InfoR(r, "PaymentAuthenticationInterceptor authorised as payment lookup role on GET")
			// Call the next handler
			next.ServeHTTP(w, r)
		} else {
			// If none of the above conditions above are met then the request is
			// unauthorized
			w.WriteHeader(http.StatusUnauthorized)
			log.InfoR(r, "PaymentAuthenticationInterceptor unauthorised")
		}
	})
}
