package interceptors

import (
	"context"
	"fmt"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/gorilla/mux"
	"net/http"
	"strings"

	"github.com/companieshouse/chs.go/log"
)

func UserAuthenticationInterceptor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Check headers for identity type and identity
		identityType := helpers.GetAuthorisedIdentityType(r)
		if identityType != helpers.Oauth2IdentityType {
			log.Error(fmt.Errorf("Authentication interceptor unauthorised: not oauth2 identity type"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		identity := helpers.GetAuthorisedIdentity(r)
		if identity == "" {
			log.Error(fmt.Errorf("Authentication interceptor unauthorised: no authorised identity"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		authorisedUser := helpers.GetAuthorisedUser(r)
		if authorisedUser == "" {
			log.Error(fmt.Errorf("Authentication interceptor unauthorised: no authorised user"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Extract user details and add to context
		userDetails := strings.Split(authorisedUser, ";")
		authUserDetails := models.AuthUserDetails{}

		switch len(userDetails) {
		case 1:
			authUserDetails.User_email = strings.TrimSpace(userDetails[0])
		case 2:
			authUserDetails.User_email = strings.TrimSpace(userDetails[0])
			authUserDetails.User_forename = userDetails[1]
		case 3:
			authUserDetails.User_email = strings.TrimSpace(userDetails[0])
			authUserDetails.User_forename = userDetails[1]
			authUserDetails.User_surname = userDetails[2]
		}

		ctx := context.WithValue(r.Context(), "user_details", authUserDetails)

		// Call the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func PaymentAuthenticationInterceptor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["payment_id"]
		if id == "" {
			log.ErrorR(r, fmt.Errorf("PaymentAuthenticationInterceptor error: no payment id"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		cfg, err := config.Get()
		if err != nil {
			log.Error(fmt.Errorf("error configuring service: %s. Exiting", err), nil)
			return
		}

		m := &dao.Mongo{
			URL: cfg.MongoDBURL,
		}

		paymentSession, err := m.GetPaymentResource(id)
		if err != nil {
			log.Error(fmt.Errorf("PaymentAuthenticationInterceptor not found: payment session found"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		authorisedUser := helpers.GetAuthorisedIdentity(r)
		if authorisedUser == "" {
			log.Error(fmt.Errorf("PaymentAuthenticationInterceptor unauthorised: no authorised identity"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		isGetRequest := http.MethodGet == r.Method
		authUserIsPaymentCreator := authorisedUser == paymentSession.Data.CreatedBy.ID
		authUserHasPaymentLookupRole := helpers.IsRoleAuthorised(r, helpers.AdminPaymentLookupRole)

		// Now that we have the payment data and authorized user there are
		// multiple cases that can be allowed through:

		// 1) Authorized user created the payment and request is a GET i.e.
		// to see payment data but not modify/delete
		if (authUserIsPaymentCreator && isGetRequest){
			log.InfoR(r, "PaymentAuthenticationInterceptor authorised as creator on GET")
			// Call the next handler
			next.ServeHTTP(w, r)
		}

		// 2) Authorized user has permission to lookup any payment session and
		// request is a GET i.e. to see payment data but not modify/delete
		if (authUserHasPaymentLookupRole && isGetRequest){
			log.InfoR(r, "PaymentAuthenticationInterceptor authorised as payment lookup role on GET")
			// Call the next handler
			next.ServeHTTP(w, r)
		}

		// If none of the above conditions above are met then the request is
		// unauthorized
		log.InfoR(r, "PaymentAuthenticationInterceptor unauthorised")
	})
}
