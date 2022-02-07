package interceptors

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/authentication"
	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
)

// PaymentAdminAuthenticationIntercept checks that the user is authenticated for payment admin priveleges
func PaymentAdminAuthenticationIntercept(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Get identity type from request
		identityType := authentication.GetAuthorisedIdentityType(r)
		if !(identityType == authentication.Oauth2IdentityType || identityType == authentication.APIKeyIdentityType) {
			log.Error(fmt.Errorf("authentication interceptor unauthorised: not oauth2 or API key identity type"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		authorisedUser := ""

		// Get user details from context, passed in by UserAuthenticationInterceptor
		userDetails, ok := r.Context().Value(authentication.ContextKeyUserDetails).(authentication.AuthUserDetails)
		if !ok {
			log.ErrorR(r, fmt.Errorf("PaymentAdminAuthenticationInterceptor error: invalid AuthUserDetails from UserAuthenticationInterceptor"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Get user details from request
		authorisedUser = userDetails.ID
		if authorisedUser == "" {
			log.Error(fmt.Errorf("PaymentAdminAuthenticationInterceptor unauthorised: no authorised identity"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		isPostRequest := http.MethodPost == r.Method
		authUserHasBulkRefundRole := authentication.IsRoleAuthorised(r, helpers.AdminBulkRefundRole)

		// Set up debug map for logging
		debugMap := log.Data{
			"auth_user_has_bulk_refund_role": authUserHasBulkRefundRole,
			"request_method":                 r.Method,
		}

		// Check if authorised user has bulk refund role present
		if authUserHasBulkRefundRole && isPostRequest {
			log.InfoR(r, "PaymentAdminAuthenticationInterceptor authorised as bulk refund admin role on POST", debugMap)
			next.ServeHTTP(w, r)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		log.InfoR(r, "PaymentAdminAuthenticationInterceptor unauthorised", debugMap)
	})
}
