package models

// OutgoingPayPalOrderRequest is the request sent to PayPal to initiate a payment session
type OutgoingPayPalOrderRequest struct {
	Intent             string             `json:"intent"`
	PurchaseUnits      []PurchaseUnit     `json:"purchase_units"`
	ApplicationContext ApplicationContext `json:"application_context"`
}

// PurchaseUnit contains an amount for a PayPal order
type PurchaseUnit struct {
	Amount Amount `json:"amount"`
}

// Amount is the amount object for a PayPal order
type Amount struct {
	CurrencyCode string `json:"currency_code"`
	Value        string `json:"value"`
}

// ApplicationContext is needed to supply PayPal with a return url
type ApplicationContext struct {
	ReturnUrl string `json:"return_url"`
}
