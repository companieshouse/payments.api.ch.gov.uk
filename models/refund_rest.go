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
