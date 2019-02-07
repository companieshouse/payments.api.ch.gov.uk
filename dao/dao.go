package dao

import "github.com/companieshouse/payments.api.ch.gov.uk/models"

// DAO is an interface for accessing dao from a backend store
type DAO interface {
	CreatePaymentResource(paymentResource *models.PaymentResourceDB) error
	GetPaymentResource(string) (*models.PaymentResourceDB, error)
	PatchPaymentResource(id string, paymentUpdate *models.PaymentResourceDB) error
}
