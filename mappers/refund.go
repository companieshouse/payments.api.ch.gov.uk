package mappers

import "github.com/companieshouse/payments.api.ch.gov.uk/models"

func MapToRefundRest(response models.CreateRefundGovPayResponse) models.RefundResourceRest {
	return models.RefundResourceRest{
		RefundId:          response.RefundId,
		CreatedAt:         response.CreatedDate,
		Amount:            response.Amount,
		Status:            response.Status,
		ExternalRefundUrl: response.Links.Self.HREF,
	}
}

func MapToRefundResponse(gpResponse models.CreateRefundGovPayResponse) models.CreateRefundResponse {
	return models.CreateRefundResponse{
		RefundId:        gpResponse.RefundId,
		Amount:          gpResponse.Amount,
		CreatedDateTime: gpResponse.CreatedDate,
		Status:          gpResponse.Status,
	}
}
