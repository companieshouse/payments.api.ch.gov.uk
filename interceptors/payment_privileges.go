package interceptors

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/authentication"
	"github.com/companieshouse/chs.go/log"
)

// ElevatedOrPaymentPrivilegesIntercept checks that the user is authenticated
func ElevatedOrPaymentPrivilegesIntercept(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers for identity type and identity
		identityType := authentication.GetAuthorisedIdentityType(r)
		if !(identityType == authentication.APIKeyIdentityType) {
			log.Error(fmt.Errorf("elevated privileges interceptor unauthorised: not API key identity type"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if authentication.IsKeyElevatedPrivilegesAuthorised(r) || authentication.CheckAuthorisedKeyHasPrivilege(r, "payment") {
			// Call the next handler
			next.ServeHTTP(w, r)
		} else {
			// If the request is not with an elevated privileges API key then the request is unauthorized
			w.WriteHeader(http.StatusUnauthorized)
			log.Error(fmt.Errorf("elevated privileges interceptor unauthorised: not elevated privileges API key"))
		}
	})
}

// Oauth2OrPaymentPrivilegesIntercept checks that the user is authenticated via Oauth2 or having the
// payment privilege
func Oauth2OrPaymentPrivilegesIntercept(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identityType := authentication.GetAuthorisedIdentityType(r)
		if !(identityType == authentication.Oauth2IdentityType || identityType == authentication.APIKeyIdentityType) {
			log.Error(fmt.Errorf("oauth2 privileges interceptor unauthorised: not oauth2 or API key identity type"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if identityType == authentication.Oauth2IdentityType || authentication.CheckAuthorisedKeyHasPrivilege(r, "payment") {
			// Call the next handler
			next.ServeHTTP(w, r)
		}

		// If the request is not with an elevated privileges API key or Oauth2 authorised then the request is unauthorized
		w.WriteHeader(http.StatusUnauthorized)
		log.Error(fmt.Errorf("oauth2 privileges interceptor unauthorised: not oauth2 or not payment privileges API key"))
	})
}