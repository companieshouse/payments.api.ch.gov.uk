package models

// IncomingExternalPaymentJourneyRequest is the data received in the body of the incoming request
type IncomingExternalPaymentJourneyRequest struct {
	PaymentMethod string `json:"payment_method"`
	Resource      string `json:"resource"`
}

// ExternalPaymentJourney returns the URL required to access external payment provider session
type ExternalPaymentJourney struct {
	NextURL string `json:"NextURL"`
}
