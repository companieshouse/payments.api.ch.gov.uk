package helpers

// ContextKey is a type for creating context keys
type ContextKey string

// ContextKeyUserDetails is a specific key for identifying "user_details" contexts added to the http request
var ContextKeyUserDetails = ContextKey("user_details")

// ContextKeyPaymentSession is a specific key for identifying "payment_session" contexts added to the http request
var ContextKeyPaymentSession = ContextKey("payment_session")
