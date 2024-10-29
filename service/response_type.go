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

	// Success (terminal) response
	Success

	// CostsNotFound response
	CostsNotFound

	// CostsGone response
	CostsGone

	// Conflict response
	Conflict

	// Payment Created (non-terminal) response
	Created
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
	"created",
}

// String representation of `ResponseType`
func (a ResponseType) String() string {
	return vals[a]
}
