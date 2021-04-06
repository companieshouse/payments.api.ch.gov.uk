package service

// ResponseType enumerates the types of authentication supported
type ResponseType int

const (
	// InvalidData response
	InvalidData ResponseType = iota

	// Error response
	Error

	// Forbidden response
	Forbidden

	// NotFound response
	NotFound

	// Success response
	Success

	// Cost Resource Not Found response
	CostsNotFound

	// Costs Gone response
	CostsGone
)

var vals = [...]string{
	"invalid-data",
	"error",
	"forbidden",
	"not-found",
	"success",
	"costs-not-found",
	"costs-gone",
}

// String representation of `ResponseType`
func (a ResponseType) String() string {
	return vals[a]
}
