package helpers

import (
	"net/http"
	"strings"
)

const (
	Oauth2IdentityType     = "oauth2"
	AdminPaymentLookupRole = "/admin/payment-lookup"

	ericIdentity        = "ERIC-Identity"
	ericIdentityType    = "ERIC-Identity-Type"
	ericAuthorisedUser  = "ERIC-Authorised-User"
	ericAuthorisedRoles = "ERIC-Authorised-Roles"
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

func GetAuthorisedRoles(r *http.Request) string {
	return r.Header.Get(ericAuthorisedRoles)
}

func getAuthorisedRolesArray(r *http.Request) []string {
	roles := r.Header.Get(ericAuthorisedRoles)
	if roles == "" {
		return nil
	}

	return strings.Split(roles, " ")
}

func IsRoleAuthorised(r *http.Request, role string) bool {
	if role == "" {
		return false
	}

	roles := getAuthorisedRolesArray(r)
	if roles == nil {
		return false
	}

	return contains(roles, role)
}

// contains returns whether array contains string s.
func contains(array []string, s string) bool {
	for _, n := range array {
		if s == n {
			return true
		}
	}
	return false
}
