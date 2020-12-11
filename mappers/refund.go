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

func MapGovPayToRefundResponse(gpResponse models.CreateRefundGovPayResponse) models.RefundResponse {
	return models.RefundResponse{
		RefundId:        gpResponse.RefundId,
		Amount:          gpResponse.Amount,
		CreatedDateTime: gpResponse.CreatedDate,
		Status:          gpResponse.Status,
	}
}

func MapRefundToRefundResponse(refund models.RefundResourceRest) models.RefundResponse {
	return models.RefundResponse{
		RefundId:        refund.RefundId,
		Amount:          refund.Amount,
		CreatedDateTime: refund.CreatedAt,
		Status:          refund.Status,
	}
}
