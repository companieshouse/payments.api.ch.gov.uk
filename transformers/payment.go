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
		Amount:        rest.Amount,
		CompletedAt:   rest.CompletedAt,
		CreatedAt:     rest.CreatedAt,
		Description:   rest.Description,
		PaymentMethod: rest.PaymentMethod,
		Reference:     rest.Reference,
		Status:        rest.Status,
	}

	paymentResourceData.CreatedBy = models.CreatedByDB(rest.CreatedBy)
	paymentResourceData.Links = models.PaymentLinksDB(rest.Links)

	paymentResource := models.PaymentResourceDB{
		Data: paymentResourceData,
	}

	return paymentResource
}

// TransformToRest transforms payment resource database model into payment resource rest model
func (pt PaymentTransformer) TransformToRest(dbResource models.PaymentResourceDB) models.PaymentResourceRest {
	paymentResource := models.PaymentResourceRest{
		Amount:        dbResource.Data.Amount,
		CompletedAt:   dbResource.Data.CompletedAt,
		CreatedAt:     dbResource.Data.CreatedAt,
		CreatedBy:     models.CreatedByRest(dbResource.Data.CreatedBy),
		Description:   dbResource.Data.Description,
		PaymentMethod: dbResource.Data.PaymentMethod,
		Reference:     dbResource.Data.Reference,
		Status:        dbResource.Data.Status,
		Links:         models.PaymentLinksRest(dbResource.Data.Links),
	}
	return paymentResource
}
