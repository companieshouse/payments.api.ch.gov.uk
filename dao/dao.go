package dao

import "github.com/companieshouse/payments.api.ch.gov.uk/models"

// DAO is an interface for accessing dao from a backend store
type DAO interface {
	CreatePaymentResource(paymentResource *models.PaymentResource) error
	GetPaymentResource(string) (*models.PaymentResourceData, error)
}
