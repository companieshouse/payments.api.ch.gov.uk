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
	}

	paymentResourceData.CreatedBy = models.CreatedByDB(rest.CreatedBy)
	paymentResourceData.Links = models.PaymentLinksDB(rest.Links)
	paymentResourceData.Costs = pt.transformCostResourcesToDB(rest.Costs)

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
		Costs:                   pt.transformCostResourcesToRest(dbResourceData.Costs),
	}
	return paymentResource
}

func (pt PaymentTransformer) transformCostResourcesToDB(rest []models.CostResourceRest) []models.CostResourceDB {
	costResources := make([]models.CostResourceDB, len(rest))
	for i, restCost := range rest {
		costResources[i] = pt.transformCostResourceToDB(restCost)
	}
	return costResources
}

func (pt PaymentTransformer) transformCostResourcesToRest(dbCostResources []models.CostResourceDB) []models.CostResourceRest {
	costResources := make([]models.CostResourceRest, len(dbCostResources))
	for i, cost := range dbCostResources {
		costResources[i] = pt.transformCostResourceToRest(cost)
	}
	return costResources
}

func (pt PaymentTransformer) transformCostResourceToDB(rest models.CostResourceRest) models.CostResourceDB {
	return models.CostResourceDB{
		Amount:                  rest.Amount,
		AvailablePaymentMethods: rest.AvailablePaymentMethods,
		ClassOfPayment:          rest.ClassOfPayment,
		Description:             rest.Description,
		DescriptionIdentifier:   rest.DescriptionIdentifier,
		DescriptionValues:       rest.DescriptionValues,
		Links:                   models.CostLinksDB(rest.Links),
	}
}

func (pt PaymentTransformer) transformCostResourceToRest(dbCostResource models.CostResourceDB) models.CostResourceRest {
	return models.CostResourceRest{
		Amount:                  dbCostResource.Amount,
		AvailablePaymentMethods: dbCostResource.AvailablePaymentMethods,
		ClassOfPayment:          dbCostResource.ClassOfPayment,
		Description:             dbCostResource.Description,
		DescriptionIdentifier:   dbCostResource.DescriptionIdentifier,
		DescriptionValues:       dbCostResource.DescriptionValues,
		Links:                   models.CostLinksRest(dbCostResource.Links),
	}
}
