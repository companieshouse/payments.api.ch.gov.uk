package models

// StatusResponse is the generic response
type StatusResponse struct {
	Status string
}

type response_service interface {
	checkProvider()
}
