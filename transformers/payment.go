package transformers

import (
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
)

// Transformer is an interface for all transformer implementations to implement
type Transformer interface {
	TransformToDB(interface{}) interface{}
	TransformToRest(interface{}) interface{}
}

// PaymentTransformer transforms payment resource data between rest and database models
type PaymentTransformer struct{}

// TransformToDB transforms payment resource rest model into payment resource database model
func (pt PaymentTransformer) TransformToDB(rest models.PaymentResourceRest) models.PaymentResourceDB {
	paymentResourceData := models.PaymentResourceDataDB{
		Amount:                  rest.Amount,
		AvailablePaymentMethods: rest.AvailablePaymentMethods,
		CompletedAt:             rest.CompletedAt,
		CreatedAt:               rest.CreatedAt,
		Description:             rest.Description,
		PaymentMethod:           rest.PaymentMethod,
		Reference:               rest.Reference,
		Status:                  rest.Status,
		Etag:                    rest.Etag,
		Kind:                    rest.Kind,
	}

	paymentResourceData.CreatedBy = models.CreatedByDB(rest.CreatedBy)
	paymentResourceData.Links = models.PaymentLinksDB(rest.Links)

	paymentResource := models.PaymentResourceDB{
		Data: paymentResourceData,
	}

	return paymentResource
}

// TransformToRest transforms payment resource database model into payment resource rest model
func (pt PaymentTransformer) TransformToRest(dbResourceData models.PaymentResourceDataDB) models.PaymentResourceRest {
	paymentResource := models.PaymentResourceRest{
		Amount:                  dbResourceData.Amount,
		AvailablePaymentMethods: dbResourceData.AvailablePaymentMethods,
		CompletedAt:             dbResourceData.CompletedAt,
		CreatedAt:               dbResourceData.CreatedAt,
		CreatedBy:               models.CreatedByRest(dbResourceData.CreatedBy),
		Description:             dbResourceData.Description,
		PaymentMethod:           dbResourceData.PaymentMethod,
		Reference:               dbResourceData.Reference,
		Status:                  dbResourceData.Status,
		Links:                   models.PaymentLinksRest(dbResourceData.Links),
		Etag:                    dbResourceData.Etag,
		Kind:                    dbResourceData.Kind,
	}
	return paymentResource
}
