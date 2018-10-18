package dao

import "github.com/companieshouse/payments.api.ch.gov.uk/models"

// DAO is an interface for accessing dao from a backend store
type DAO interface {
	CreatePaymentResource(*models.PaymentResource) error
	GetPaymentResource(string) (models.PaymentResource, error)
	UpdatePaymentAmount(*models.PaymentResource, string) error
}
