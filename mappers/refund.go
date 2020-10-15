package mappers

import "github.com/companieshouse/payments.api.ch.gov.uk/models"

func MapToRefundRest(response models.CreateRefundGovPayResponse) models.RefundResourceRest {
	refundRest := models.RefundResourceRest{
		RefundId:    response.RefundId,
		CreatedDate: response.CreatedDate,
		Amount:      response.Amount,
		Links:       mapToRefundsLinksRest(response.Links),
		Status:      response.Status,
	}

	return refundRest
}

func mapToRefundsLinksRest(db models.GovPayRefundLinks) models.RefundLinksRest {
	refundLinksRest := models.RefundLinksRest{
		Self: models.RefundSelfRest{
			HREF:   db.Self.HREF,
			Method: db.Self.Method,
		},
		Payment: models.RefundPaymentRest{
			HREF:   db.Payment.HREF,
			Method: db.Payment.Method,
		},
	}

	return refundLinksRest
}
