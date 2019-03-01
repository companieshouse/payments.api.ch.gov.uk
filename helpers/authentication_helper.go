package helpers

import (
	"net/http"
	"strings"
)

const (
	// Oauth2IdentityType defines the identity type for OAuth2.
	Oauth2IdentityType = "oauth2"

	// AdminPaymentLookupRole defines the path to check whether a user is authorised to look up a payment.
	AdminPaymentLookupRole = "/admin/payment-lookup"

	ericIdentity        = "ERIC-Identity"
	ericIdentityType    = "ERIC-Identity-Type"
	ericAuthorisedUser  = "ERIC-Authorised-User"
	ericAuthorisedRoles = "ERIC-Authorised-Roles"
)

// GetAuthorisedIdentity gets the Identity from the Header.
func GetAuthorisedIdentity(r *http.Request) string {
	return r.Header.Get(ericIdentity)
}

// GetAuthorisedIdentityType gets the Identity Type from the Header.
func GetAuthorisedIdentityType(r *http.Request) string {
	return r.Header.Get(ericIdentityType)
}

// GetAuthorisedUser gets the User from the Header.
func GetAuthorisedUser(r *http.Request) string {
	return r.Header.Get(ericAuthorisedUser)
}

// GetAuthorisedRoles gets the Roles from the Header.
func GetAuthorisedRoles(r *http.Request) string {
	return r.Header.Get(ericAuthorisedRoles)
}

func getAuthorisedRolesArray(r *http.Request) []string {
	roles := r.Header.Get(ericAuthorisedRoles)
	if len(roles) == 0 {
		return nil
	}

	return strings.Split(roles, " ")
}

// IsRoleAuthorised checks whether a Role is Authorise
func IsRoleAuthorised(r *http.Request, role string) bool {
	if len(role) == 0 {
		return false
	}

	roles := getAuthorisedRolesArray(r)
	if roles == nil || len(roles) == 0 {
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
