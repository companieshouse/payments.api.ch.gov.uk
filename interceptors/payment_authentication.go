package interceptors

import (
	"context"
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/gorilla/mux"
)

// PaymentAuthenticationInterceptor contains the payment service used in the interceptor
type PaymentAuthenticationInterceptor struct {
	Service service.PaymentService
}

func (paymentAuthenticationInterceptor PaymentAuthenticationInterceptor) PaymentAuthenticationIntercept(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for a payment ID in request
		vars := mux.Vars(r)
		id := vars["payment_id"]
		if id == "" {
			log.ErrorR(r, fmt.Errorf("PaymentAuthenticationInterceptor error: no payment id"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Get user details from context, passed in by UserAuthenticationInterceptor
		userDetails, ok := r.Context().Value(helpers.UserDetailsKey).(models.AuthUserDetails)
		if !ok {
			log.ErrorR(r, fmt.Errorf("PaymentAuthenticationInterceptor error: invalid AuthUserDetails from UserAuthenticationInterceptor"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Get user details from request
		authorisedUser := userDetails.Id
		if authorisedUser == "" {
			log.Error(fmt.Errorf("PaymentAuthenticationInterceptor unauthorised: no authorised identity"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Get the payment session from the ID in request
		paymentSession, httpStatus, err := paymentAuthenticationInterceptor.Service.GetPaymentSession(id)
		if err != nil {
			log.Error(fmt.Errorf("PaymentAuthenticationInterceptor error when retrieving payment session: [%v]", err))
			w.WriteHeader(httpStatus)
			return
		}

		// Store paymentSession in context to use later in the handler
		ctx := context.WithValue(r.Context(), service.PaymentSessionKey, paymentSession)

		// Set up variables that are used to determine authorisation below
		isGetRequest := http.MethodGet == r.Method
		authUserIsPaymentCreator := authorisedUser == paymentSession.CreatedBy.ID
		authUserHasPaymentLookupRole := helpers.IsRoleAuthorised(r, helpers.AdminPaymentLookupRole)

		// Set up debug map for logging at each exit point
		debugMap := log.Data{
			"payment_id":                        id,
			"auth_user_is_payment_creator":      authUserIsPaymentCreator,
			"auth_user_has_payment_lookup_role": authUserHasPaymentLookupRole,
			"request_method":                    r.Method,
		}

		// Now that we have the payment data and authorized user there are
		// multiple cases that can be allowed through:
		if authUserIsPaymentCreator {
			// 1) Authorized user created the payment
			log.InfoR(r, "PaymentAuthenticationInterceptor authorised as creator", debugMap)
			// Call the next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		} else if authUserHasPaymentLookupRole && isGetRequest {
			// 2) Authorized user has permission to lookup any payment session and
			// request is a GET i.e. to see payment data but not modify/delete
			log.InfoR(r, "PaymentAuthenticationInterceptor authorised as payment lookup role on GET", debugMap)
			// Call the next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			// If none of the above conditions above are met then the request is
			// unauthorized
			w.WriteHeader(http.StatusUnauthorized)
			log.InfoR(r, "PaymentAuthenticationInterceptor unauthorised", debugMap)
		}
	})
}
