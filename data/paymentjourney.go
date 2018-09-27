package data

// IncomingPaymentResourceRequest is the data received in the body of the incoming request
type IncomingExternalPaymentJourneyRequest struct {
	PaymentMethod string `json:"payment_method"`
	Resource      string `json:"resource"`
}

// PaymentJourney returns the URL required to access external payment provider session
type ExternalPaymentJourney struct {
	NextUrl string `json:"next_url"`
}
