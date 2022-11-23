package models

// RefundResourceDB represents the database refund structure
type RefundResourceDB struct {
	RefundId          string `bson:"refund_id"`
	CreatedAt         string `bson:"created_at"`
	Amount            int    `bson:"amount"`
	Status            string `bson:"status"`
	Attempts          int    `bson:"attempts,omitempty"`
	ExternalRefundUrl string `bson:"external_refund_url"`
	RefundReference   string `bson:"refund_reference"`
}
