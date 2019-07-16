package models

// OutgoingGovPayRequest is the request sent to GovPay to initiate a payment session
type OutgoingGovPayRequest struct {
	Amount      int    `json:"amount"`
	Reference   string `json:"reference"`
	ReturnURL   string `json:"return_url"`
	Description string `json:"description"`
}

// IncomingGovPayResponse is the response expected back from GovPay after a payment session has been successfully initiated
type IncomingGovPayResponse struct {
	Amount            int               `json:"amount"`
	State             State             `json:"state"`
	Description       string            `json:"description"`
	Reference         string            `json:"reference"`
	Language          string            `json:"language"`
	PaymentID         string            `json:"payment_id"`
	PaymentProvider   string            `json:"payment_provider"`
	ReturnURL         string            `json:"return_url"`
	CreatedDate       string            `json:"created_date"`
	RefundSummary     RefundSummary     `json:"refund_summary"`
	SettlementSummary SettlementSummary `json:"settlement_summary"`
	DelayedCapture    bool              `json:"delayed_capture"`
	GovPayLinks       GovPayLinks       `json:"_links"`
	CardBrand         string            `json:"card_brand"`
}

// State is the current state of the payment
type State struct {
	Status   string `json:"status"`
	Finished bool   `json:"finished"`
	Code     string `json:"code"`
}

// RefundSummary is the refund status of the payment
type RefundSummary struct {
	Status          string `json:"status"`
	AmountAvailable int    `json:"amount_available"`
	AmountSubmitted int    `json:"amount_submitted"`
}

// SettlementSummary is the settlement status of the payment
type SettlementSummary struct {
	CaptureSubmitTime string `json:"capture_submit_time"`
	CapturedDate      string `json:"captured_date"`
}

// GovPayLinks contains links for this payment, including the next_url to continue the journey, and the link to check the status.
type GovPayLinks struct {
	Self        Self        `json:"self"`
	NextURL     NextURL     `json:"next_url"`
	NextURLPost NextURLPost `json:"next_url_post"`
	Events      Events      `json:"events"`
	Refunds     Refunds     `json:"refunds"`
	Cancel      Cancel      `json:"cancel"`
}

// Self links to the payment
type Self struct {
	HREF   string `json:"href"`
	Method string `json:"method"`
}

// NextURL contains where to navigate the user next as a GET
type NextURL struct {
	HREF   string `json:"href"`
	Method string `json:"method"`
}

// NextURLPost contains where to navigate the user next as a POST
type NextURLPost struct {
	PostType string `json:"type"`
	Params   Params `json:"params"`
	HREF     string `json:"href"`
	Method   string `json:"method"`
}

// Params contains the charge token for the payment
type Params struct {
	ChargeTokenID string `json:"chargeTokenId"`
}

// Events contains a link to payment events
type Events struct {
	HREF   string `json:"href"`
	Method string `json:"method"`
}

// Refunds contains a link to payment refunds
type Refunds struct {
	HREF   string `json:"href"`
	Method string `json:"method"`
}

// Cancel contains a link to cancel the payment
type Cancel struct {
	HREF   string `json:"href"`
	Method string `json:"method"`
}

// PaymentDetails is used by the payment-details endpoint to return card type and an auth number which is the payment id
type PaymentDetails struct {
	CardType  string `json:"card_type"`
	PaymentID string `json:"payment_id"`
}
