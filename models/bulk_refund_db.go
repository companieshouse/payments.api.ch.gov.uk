package models

// BulkRefundDB contains all the details for a bulk refund
// Bulk refunds come specifically from an XML file generated by
// E5 and process a batch of refunds at once
type BulkRefundDB struct {
	Status            string `bson:"status"`
	UploadedFilename  string `bson:"uploaded_filename"`
	UploadedAt        string `bson:"uploaded_at"`
	UploadedBy        string `bson:"uploaded_by"`
	Amount            string `bson:"amount"`
	RefundID          string `bson:"refund_id"`
	ProcessedAt       string `bson:"processed_at"`
	ExternalRefundURL string `bson:"external_refund_url"`
}