package interceptors

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"

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
		authUserDetails := models.AuthUserDetails{Id: identity}

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

		ctx := context.WithValue(r.Context(), helpers.ContextKeyUserDetails, authUserDetails)

		// Call the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
