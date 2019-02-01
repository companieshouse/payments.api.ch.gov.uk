package transformers

import (
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
)

// Transformer is an interface for all transformer implementations to implement
type Transformer interface {
	TransformToDB(interface{}) interface{}
	TransformFromDB(interface{}) interface{}
}

// PaymentTransformer transforms payment resource data between rest and database models
type PaymentTransformer struct{}

// TransformToDB transforms payment resource rest model into payment resource database model
func (pt PaymentTransformer) TransformToDB(rest models.PaymentResourceRest) models.PaymentResource {
	paymentResourceData := models.PaymentResourceData{
		Amount:                  rest.Amount,
		AvailablePaymentMethods: rest.AvailablePaymentMethods,
		CompletedAt:             rest.CompletedAt,
		CreatedAt:               rest.CreatedAt,
		Description:             rest.Description,
		PaymentMethod:           rest.PaymentMethod,
		Reference:               rest.Reference,
		Status:                  rest.Status,
	}

	paymentResourceData.CreatedBy = models.CreatedBy(rest.CreatedBy)
	paymentResourceData.Links = models.PaymentLinks(rest.Links)
	paymentResourceData.Costs = pt.transformCostResourcesToDB(rest.Costs)

	paymentResource := models.PaymentResource{
		Data: paymentResourceData,
	}

	return paymentResource
}

// TransformFromDB transforms payment resource database model into payment resource rest model
func (pt PaymentTransformer) TransformFromDB(dbResourceData models.PaymentResourceData) models.PaymentResourceRest {
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
		Costs:                   pt.transformCostResourcesFromDB(dbResourceData.Costs),
	}
	return paymentResource
}

func (pt PaymentTransformer) transformCostResourcesToDB(rest []models.CostResourceRest) []models.CostResource {
	costResources := make([]models.CostResource, len(rest))
	for i, restCost := range rest {
		costResources[i] = pt.transformCostResourceToDB(restCost)
	}
	return costResources
}

func (pt PaymentTransformer) transformCostResourcesFromDB(dbCostResources []models.CostResource) []models.CostResourceRest {
	costResources := make([]models.CostResourceRest, len(dbCostResources))
	for i, cost := range dbCostResources {
		costResources[i] = pt.transformCostResourceFromDB(cost)
	}
	return costResources
}

func (pt PaymentTransformer) transformCostResourceToDB(rest models.CostResourceRest) models.CostResource {
	return models.CostResource{
		Amount:                  rest.Amount,
		AvailablePaymentMethods: rest.AvailablePaymentMethods,
		ClassOfPayment:          rest.ClassOfPayment,
		Description:             rest.Description,
		DescriptionIdentifier:   rest.DescriptionIdentifier,
		DescriptionValues:       rest.DescriptionValues,
		Links:                   models.CostLinks(rest.Links),
	}
}

func (pt PaymentTransformer) transformCostResourceFromDB(dbCostResource models.CostResource) models.CostResourceRest {
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
