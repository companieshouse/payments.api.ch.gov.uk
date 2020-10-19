package models

type RefundResourceDB struct {
	RefundId  string        `bson:"refund_id"`
	CreatedAt string        `bson:"created_at"`
	Amount    int           `bson:"amount"`
	Links     RefundLinksDB `bson:"_links"`
	Status    string        `bson:"status"`
}

type RefundLinksDB struct {
	Self    RefundSelfDB    `bson:"self"`
	Payment RefundPaymentDB `bson:"payment"`
}

// Self links to the payment
type RefundSelfDB struct {
	HREF   string `bson:"href"`
	Method string `bson:"method"`
}

// Payment links to the payment
type RefundPaymentDB struct {
	HREF   string `bson:"href"`
	Method string `bson:"method"`
}
