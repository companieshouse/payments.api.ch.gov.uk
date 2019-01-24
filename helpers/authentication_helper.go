package helpers

import "net/http"

const (
	Oauth2IdentityType = "oauth2"

	ericIdentity       = "ERIC-Identity"
	ericIdentityType   = "ERIC-Identity-Type"
	ericAuthorisedUser = "ERIC-Authorised-User"
)

func GetAuthorisedIdentity(r *http.Request) string {
	return r.Header.Get(ericIdentity)
}

func GetAuthorisedIdentityType(r *http.Request) string {
	return r.Header.Get(ericIdentityType)
}

func GetAuthorisedUser(r *http.Request) string {
	return r.Header.Get(ericAuthorisedUser)
}
