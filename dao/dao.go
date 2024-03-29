package dao

import (
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
)

// DAO is an interface for accessing dao from a backend store
type DAO interface {
	CreatePaymentResource(paymentResource *models.PaymentResourceDB) error
	GetPaymentResource(string) (*models.PaymentResourceDB, error)
	PatchPaymentResource(id string, paymentUpdate *models.PaymentResourceDB) error
	GetPaymentResourceByProviderID(providerID string) (*models.PaymentResourceDB, error)
	GetPaymentResourceByExternalPaymentTransactionID(providerID string) (*models.PaymentResourceDB, error)
	GetIncompleteGovPayPayments(*config.Config) ([]models.PaymentResourceDB, error)
	CreateBulkRefundByProviderID(bulkRefunds map[string]models.BulkRefundDB) error
	CreateBulkRefundByExternalPaymentTransactionID(bulkRefunds map[string]models.BulkRefundDB) error
	GetPaymentsWithRefundStatus() ([]models.PaymentResourceDB, error)
	GetPaymentsWithRefundPendingStatus() ([]models.PaymentResourceDB, error)
	GetPaymentRefunds(string) ([]models.RefundResourceDB, error)
	PatchRefundSuccessStatus(id string, isPaid bool, paymentUpdate *models.PaymentResourceDB) (models.PaymentResourceDB, error)
	PatchRefundStatus(id string, isRefunded bool, isFailed bool, refundStatus string, paymentUpdate *models.PaymentResourceDB) (models.PaymentResourceDB, error)
	IncrementRefundAttempts(paymentID string, paymentUpdate *models.PaymentResourceDB) error
}

// NewDAO will create a new instance of the DAO interface.
// All details about its implementation and the
// database driver will be hidden from outside this package
func NewDAO(cfg *config.Config) DAO {
	database := getMongoDatabase(cfg.MongoDBURL, cfg.Database)

	return &MongoService{
		db:              database,
		CollectionName:  cfg.Collection,
		RefundBatchSize: cfg.RefundBatchSize,
	}
}
