package models

type CreateRefundRequest struct {
	Amount int `json:"amount"`
}

type CreateRefundResponse struct {
	RefundId    string `json:"refund_id"`
	CreatedDate string `json:"created_date"`
	Amount      int    `json:"amount"`
	Status      string `json:"status"`
}

type RefundResourceRest struct {
	RefundId    string          `json:"refund_id"`
	CreatedDate string          `json:"created_date"`
	Amount      int             `json:"amount"`
	Links       RefundLinksRest `json:"_links"`
	Status      string          `json:"status"`
}

type RefundLinksRest struct {
	Self    RefundSelfRest    `json:"self"`
	Payment RefundPaymentRest `json:"payment"`
}

// Self links to the payment
type RefundSelfRest struct {
	HREF   string `json:"href"`
	Method string `json:"method"`
}

// Payment links to the payment
type RefundPaymentRest struct {
	HREF   string `json:"href"`
	Method string `json:"method"`
}
