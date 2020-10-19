package mappers

import "github.com/companieshouse/payments.api.ch.gov.uk/models"

func MapToRefundRest(response models.CreateRefundGovPayResponse) models.RefundResourceRest {
	return models.RefundResourceRest{
		RefundId:  response.RefundId,
		CreatedAt: response.CreatedDate,
		Amount:    response.Amount,
		Links:     mapToRefundsLinksRest(response.Links),
		Status:    response.Status,
	}
}

func MapToRefundResponse(gpResponse models.CreateRefundGovPayResponse) models.CreateRefundResponse {
	return models.CreateRefundResponse{
		RefundId:    gpResponse.RefundId,
		Amount:      gpResponse.Amount,
		CreatedDate: gpResponse.CreatedDate,
		Status:      gpResponse.Status,
	}
}

func mapToRefundsLinksRest(db models.GovPayRefundLinks) models.RefundLinksRest {
	return models.RefundLinksRest{
		Self: models.RefundSelfRest{
			HREF:   db.Self.HREF,
			Method: db.Self.Method,
		},
		Payment: models.RefundPaymentRest{
			HREF:   db.Payment.HREF,
			Method: db.Payment.Method,
		},
	}
}
