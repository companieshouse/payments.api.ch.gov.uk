package interceptors

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
)

// ElevatedPrivilegesInterceptor checks that the user is authenticated
func ElevatedPrivilegesInterceptor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers for identity type and identity
		identityType := helpers.GetAuthorisedIdentityType(r)
		if !(identityType == helpers.APIKeyIdentityType) {
			log.Error(fmt.Errorf("elevated privileges interceptor unauthorised: not API key identity type"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		isApiKeyRequest := identityType == helpers.APIKeyIdentityType
		apiKeyHasElevatedPrivileges := helpers.IsKeyElevatedPrivilegesAuthorised(r)
		if isApiKeyRequest && apiKeyHasElevatedPrivileges {
			// Call the next handler
			next.ServeHTTP(w, r)
		} else {
			// If the request is not with an elevated privileges API key then the request is
			// unauthorized
			w.WriteHeader(http.StatusUnauthorized)
			log.Error(fmt.Errorf("elevated privileges interceptor unauthorised: not elevated privileges API key"))
		}
	})
}
