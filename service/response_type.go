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

	// CostsNotFound response
	CostsNotFound

	// CostsGone response
	CostsGone

	// Conflict response
	Conflict
)

var vals = [...]string{
	"invalid-data",
	"error",
	"forbidden",
	"not-found",
	"success",
	"costs-not-found",
	"costs-gone",
	"conflict",
}

// String representation of `ResponseType`
func (a ResponseType) String() string {
	return vals[a]
}
