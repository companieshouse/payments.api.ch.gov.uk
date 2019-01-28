package models

// StatusResponse is the generic response
type StatusResponse struct {
	Status string
}

type response_service interface {
	checkProvider()
}

type RedirectParams struct {
	State  string
	Ref    string
	Status string
}
