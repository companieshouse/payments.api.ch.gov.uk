package interceptors

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
)

func AuthenticationInterceptor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Check headers for identity type and identityg
		identityType := r.Header.Get("Eric-Identity-Type")
		if identityType == "" || identityType != "oauth2" {
			log.Error(fmt.Errorf("Authentication interceptor unauthorised: not oauth2 identity type"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		identity := r.Header.Get("Eric-Identity")
		if identity == "" {
			log.Error(fmt.Errorf("Authentication interceptor unauthorised: no authorised identity"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}
