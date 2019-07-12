package helpers

// ContextKey is a type for creating context keys
type ContextKey string

// ContextKeyPaymentSession is a specific key for identifying "payment_session" contexts added to the http request
var ContextKeyPaymentSession = ContextKey("payment_session")
