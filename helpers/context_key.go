package helpers

// ContextKey is a type for creating context keys
type ContextKey string

// ContextKeyPaymentSession is a specific key for identifying "payment_session" contexts added to the http request
var ContextKeyPaymentSession = ContextKey("payment_session")

// ContextKeyUserID is a specific key for identifying "user_id" contexts added to the http request
var ContextKeyUserID = ContextKey("user_id")
