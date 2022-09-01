package interceptors

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/companieshouse/chs.go/authentication"
	"github.com/companieshouse/chs.go/log"
)

// InternalOrPaymentPrivilegesIntercept checks that the user is authenticated
func InternalOrPaymentPrivilegesIntercept(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers for identity type and identity
		identityType := authentication.GetAuthorisedIdentityType(r)
		if identityType != authentication.APIKeyIdentityType {
			log.Error(fmt.Errorf("internal or payment privileges interceptor unauthorised: not API key identity type"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if authentication.IsKeyElevatedPrivilegesAuthorised(r) || authentication.CheckAuthorisedKeyHasPrivilege(r, "payment") {
			// Call the next handler
			next.ServeHTTP(w, r)
		} else {
			// If the request is not with an internal or payment privileges API key then the request is unauthorized
			w.WriteHeader(http.StatusUnauthorized)
			log.Error(fmt.Errorf("internal or payment privileges interceptor unauthorised: not elevated privileges API key"))
		}
	})
}

// Oauth2OrPaymentPrivilegesIntercept checks that the user is authenticated via Oauth2 or having the
// payment privilege
func Oauth2OrPaymentPrivilegesIntercept(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identityType := authentication.GetAuthorisedIdentityType(r)
		if !(identityType == authentication.Oauth2IdentityType || identityType == authentication.APIKeyIdentityType) {
			log.Error(fmt.Errorf("oauth2 or payment privileges interceptor unauthorised: not oauth2 or API key identity type"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if identityType == authentication.Oauth2IdentityType || authentication.CheckAuthorisedKeyHasPrivilege(r, "payment") {
			// Call the next handler
			next.ServeHTTP(w, r)
			return
		}

		// If the request is not with a payment privileges API key or Oauth2 authorised then the request is unauthorized
		w.WriteHeader(http.StatusUnauthorized)
		log.Error(fmt.Errorf("oauth2 or payment privileges interceptor unauthorised: not oauth2 or not payment privileges API key"))
	})
}

// UserPaymentAuthenticationIntercept checks that the user is authenticated
func UserPaymentAuthenticationIntercept(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers for identity type and identity
		identityType := authentication.GetAuthorisedIdentityType(r)
		if !(identityType == authentication.Oauth2IdentityType || identityType == authentication.APIKeyIdentityType) {
			log.ErrorR(r, fmt.Errorf("authentication interceptor unauthorised: not oauth2 or API key identity type"), log.Data{"identity_type_used": identityType})
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		identity := authentication.GetAuthorisedIdentity(r)
		if identity == "" {
			log.ErrorR(r, fmt.Errorf("authentication interceptor unauthorised: no authorised identity"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if identityType == authentication.Oauth2IdentityType {
			authorisedUser := authentication.GetAuthorisedUser(r)
			if authorisedUser == "" {
				log.ErrorR(r, fmt.Errorf("authentication interceptor unauthorised: no authorised user"))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// Extract user details and add to context
			userDetails := strings.Split(authorisedUser, ";")
			authUserDetails := authentication.AuthUserDetails{ID: identity}

			switch len(userDetails) {
			case 1:
				authUserDetails.Email = strings.TrimSpace(userDetails[0])
			case 2:
				authUserDetails.Email = strings.TrimSpace(userDetails[0])
				authUserDetails.Forename = userDetails[1]
			case 3:
				authUserDetails.Email = strings.TrimSpace(userDetails[0])
				authUserDetails.Forename = userDetails[1]
				authUserDetails.Surname = userDetails[2]
			}

			ctx := context.WithValue(r.Context(), authentication.ContextKeyUserDetails, authUserDetails)
			log.DebugR(r, "UserAuthenticationInterceptor proceeding with OAuth2 user details in context", log.Data{"user_details": authUserDetails})

			// Call the next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		} else if identityType == authentication.APIKeyIdentityType {
			authUserDetails := authentication.AuthUserDetails{ID: identity}
			// Checks regarding 1) payment privileges and 2) the api key user being the owner of the
			// payment resource are handled in payment_authentication.go
			ctx := context.WithValue(r.Context(), authentication.ContextKeyUserDetails, authUserDetails)
			log.DebugR(r, "UserAuthenticationInterceptor proceeding with API key user", log.Data{"user_details": authUserDetails})

			// Call the next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return
	})
}
