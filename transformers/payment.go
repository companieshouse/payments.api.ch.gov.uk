package transformers

import (
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
)

// Transformer is an interface for all transformer implementations to implement
type Transformer interface {
	Transform(interface{}) interface{}
}

// PaymentTransformer transforms payment resource data between rest and database models
type PaymentTransformer struct{}

// Transform transforms payment resource rest model into payment resource database model
func (pt PaymentTransformer) Transform(rest models.PaymentResourceRest) models.PaymentResource {
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

	paymentResourceData.CreatedBy = pt.transformCreatedBy(rest.CreatedBy)
	paymentResourceData.Links = pt.transformPaymentLinks(rest.Links)
	paymentResourceData.Costs = pt.transformCostResources(rest.Costs)

	paymentResource := models.PaymentResource{
		Data: paymentResourceData,
	}

	return paymentResource
}

func (pt PaymentTransformer) transformCreatedBy(rest models.CreatedByRest) models.CreatedBy {
	return models.CreatedBy(rest)
}

func (pt PaymentTransformer) transformPaymentLinks(rest models.PaymentLinksRest) models.PaymentLinks {
	return models.PaymentLinks(rest)
}

func (pt PaymentTransformer) transformCostResources(rest []models.CostResourceRest) []models.CostResource {
	costResources := make([]models.CostResource, len(rest))
	for i, restCost := range rest {
		costResources[i] = pt.transformCostResource(restCost)
	}
	return costResources
}

func (pt PaymentTransformer) transformCostResource(rest models.CostResourceRest) models.CostResource {
	costResource := models.CostResource{
		Amount:                  rest.Amount,
		AvailablePaymentMethods: rest.AvailablePaymentMethods,
		ClassOfPayment:          rest.ClassOfPayment,
		Description:             rest.Description,
		DescriptionIdentifier:   rest.DescriptionIdentifier,
		DescriptionValues:       rest.DescriptionValues,
	}

	costResource.Links = pt.transformCostLinks(rest.Links)

	return costResource
}

func (pt PaymentTransformer) transformCostLinks(rest models.CostLinksRest) models.CostLinks {
	return models.CostLinks(rest)
}
