package models

type responseType int

// Response statuses to be passed to handlers.
const (
	InvalidData responseType = iota
	Error
	Forbidden
	NotFound
	Success
)

// ServiceResponseType defines the functions which must be supported by a Service Response
type ServiceResponseType interface {
	GetResponseType() responseType
}

// GetResponseType gets a valid response type
func (rt responseType) GetResponseType() responseType {
	return rt
}
