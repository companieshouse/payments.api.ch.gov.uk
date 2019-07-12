package models

// StatusResponse is the generic response
type StatusResponse struct {
	Status string
}

type response_service interface {
	checkProvider()
}

// RedirectParams contains parameters for redirecting.
type RedirectParams struct {
	State  string
	Ref    string
	Status string
}
