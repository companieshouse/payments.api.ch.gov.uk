package models

type CreateRefundRequest struct {
	Amount int `json:"amount"`
}

type CreateRefundResponse struct {
	RefundId        string `json:"refund_id"`
	CreatedDateTime string `json:"created_date_time"`
	Amount          int    `json:"amount"`
	Status          string `json:"status"`
}

type RefundResourceRest struct {
	RefundId          string `json:"refund_id"`
	CreatedAt         string `json:"created_at"`
	Amount            int    `json:"amount"`
	Status            string `json:"status"`
	ExternalRefundUrl string `json:"external_refund_url"`
}
