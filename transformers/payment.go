// Package transformers contains the transformation functions between the REST and DB models.
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
		CompanyNumber: rest.CompanyNumber,
		Status:        rest.Status,
		Etag:          rest.Etag,
		Kind:          rest.Kind,
		ProviderID:    rest.ProviderID,
	}

	paymentResourceData.CreatedBy = models.CreatedByDB(rest.CreatedBy)
	paymentResourceData.Links = models.PaymentLinksDB(rest.Links)

	paymentResource := models.PaymentResourceDB{
		Data:    paymentResourceData,
		Refunds: getRefundsDB(rest.Refunds),
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
		CompanyNumber: dbResource.Data.CompanyNumber,
		Status:        dbResource.Data.Status,
		Links:         models.PaymentLinksRest(dbResource.Data.Links),
		Etag:          dbResource.Data.Etag,
		Kind:          dbResource.Data.Kind,
		Refunds:       getRefundsRest(dbResource.Refunds),
		ProviderID:    dbResource.Data.ProviderID,
	}

	// One-way transformation of DB metadata: related to, but not part of the payment rest data json spec
	paymentResource.MetaData = models.PaymentResourceMetaDataRest{
		ID:                       dbResource.ID,
		RedirectURI:              dbResource.RedirectURI,
		State:                    dbResource.State,
		ExternalPaymentStatusURI: dbResource.ExternalPaymentStatusURI,
		ExternalPaymentStatusID:  dbResource.ExternalPaymentStatusID,
	}

	return paymentResource
}

func getRefundsDB(refunds []models.RefundResourceRest) []models.RefundResourceDB {
	var refundsDB []models.RefundResourceDB

	for i := 0; i < len(refunds); i++ {
		refundsDB = append(refundsDB, getRefundDB(refunds[i]))
	}

	return refundsDB
}

func getRefundDB(refund models.RefundResourceRest) models.RefundResourceDB {
	return models.RefundResourceDB{
		RefundId:          refund.RefundId,
		CreatedAt:         refund.CreatedAt,
		Amount:            refund.Amount,
		Status:            refund.Status,
		ExternalRefundUrl: refund.ExternalRefundUrl,
	}
}

func getRefundsRest(refunds []models.RefundResourceDB) []models.RefundResourceRest {
	var refundsRest []models.RefundResourceRest

	for i := 0; i < len(refunds); i++ {
		refundsRest = append(refundsRest, getRefundRest(refunds[i]))
	}

	return refundsRest
}

func getRefundRest(refund models.RefundResourceDB) models.RefundResourceRest {
	return models.RefundResourceRest{
		RefundId:          refund.RefundId,
		CreatedAt:         refund.CreatedAt,
		Amount:            refund.Amount,
		Status:            refund.Status,
		ExternalRefundUrl: refund.ExternalRefundUrl,
	}
}
