package interceptors

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/authentication"
	"github.com/companieshouse/chs.go/log"
)

// ElevatedPrivilegesInterceptor checks that the user is authenticated
func ElevatedPrivilegesInterceptor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers for identity type and identity
		identityType := authentication.GetAuthorisedIdentityType(r)
		if !(identityType == authentication.APIKeyIdentityType) {
			log.Error(fmt.Errorf("elevated privileges interceptor unauthorised: not API key identity type"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if authentication.IsKeyElevatedPrivilegesAuthorised(r) {
			// Call the next handler
			next.ServeHTTP(w, r)
		} else {
			// If the request is not with an elevated privileges API key then the request is unauthorized
			w.WriteHeader(http.StatusUnauthorized)
			log.Error(fmt.Errorf("elevated privileges interceptor unauthorised: not elevated privileges API key"))
		}
	})
}
