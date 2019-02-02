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
func (pt PaymentTransformer) TransformToRest(dbResource models.PaymentResourceDB) models.PaymentResourceRest {
	paymentResource := models.PaymentResourceRest{
		Amount:                  dbResource.Data.Amount,
		AvailablePaymentMethods: dbResource.Data.AvailablePaymentMethods,
		CompletedAt:             dbResource.Data.CompletedAt,
		CreatedAt:               dbResource.Data.CreatedAt,
		CreatedBy:               models.CreatedByRest(dbResource.Data.CreatedBy),
		Description:             dbResource.Data.Description,
		PaymentMethod:           dbResource.Data.PaymentMethod,
		Reference:               dbResource.Data.Reference,
		Status:                  dbResource.Data.Status,
		Links:                   models.PaymentLinksRest(dbResource.Data.Links),
		Costs:                   pt.transformCostResourcesToRest(dbResource.Data.Costs),
	}

	// one way transformation of DB metadata related to but not part of the payment rest data json spec
	paymentResource.MetaData = models.PaymentResourceMetaDataRest{
		ID:          dbResource.ID,
		RedirectURI: dbResource.RedirectURI,
		State:       dbResource.State,
		ExternalPaymentStatusURI: dbResource.ExternalPaymentStatusURI,
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
