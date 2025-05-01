package models

import "time"

// CreateRefundRequest contains the request data to create a refund
type CreateRefundRequest struct {
	Amount          int    `json:"amount"`
	RefundReference string `json:"refund_reference,omitempty"`
}

// RefundResponse is the data contained in a refund response
type RefundResponse struct {
	RefundId        string `json:"refund_id"`
	CreatedDateTime string `json:"created_date_time"`
	Amount          int    `json:"amount"`
	Status          string `json:"status"`
}

// RefundResourceRest is the data contained in a refund resource
type RefundResourceRest struct {
	RefundId          string     `json:"refund_id"`
	RefundedAt        *time.Time `json:"refunded_at,omitempty"`
	CreatedAt         string     `json:"created_at"`
	Amount            int        `json:"amount"`
	Status            string     `json:"status"`
	ExternalRefundUrl string     `json:"external_refund_url"`
	RefundReference   string     `json:"refund_reference"`
}
