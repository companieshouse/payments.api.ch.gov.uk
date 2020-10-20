package models

type RefundResourceDB struct {
	RefundId          string `bson:"refund_id"`
	CreatedAt         string `bson:"created_at"`
	Amount            int    `bson:"amount"`
	Status            string `bson:"status"`
	ExternalRefundUrl string `bson:"external_refund_url"`
}
