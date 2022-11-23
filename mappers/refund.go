package mappers

import (
	"fmt"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
)

// MapToRefundRest maps a GOV.UK Pay refund response to a rest resource
func MapToRefundRest(response models.CreateRefundGovPayResponse, refundReference string) models.RefundResourceRest {
	return models.RefundResourceRest{
		RefundId:          response.RefundId,
		CreatedAt:         response.CreatedDate,
		Amount:            response.Amount,
		Status:            mapGovPayStatusToInternal(response.Status),
		ExternalRefundUrl: response.Links.Self.HREF,
		RefundReference:   refundReference,
	}
}

// MapGovPayToRefundResponse maps a GOV.UK refund response to a refund response
func MapGovPayToRefundResponse(gpResponse models.CreateRefundGovPayResponse) models.RefundResponse {
	return models.RefundResponse{
		RefundId:        gpResponse.RefundId,
		Amount:          gpResponse.Amount,
		CreatedDateTime: gpResponse.CreatedDate,
		Status:          mapGovPayStatusToInternal(gpResponse.Status),
	}
}

// mapGovPayStatusToInternal maps all possible GOV.UK Pay status to internal values
// See GOV.UK Pay spec for possible values:
// https://docs.payments.service.gov.uk/refunding_payments/#checking-the-status-of-a-refund-status
// Important note - GovPay sandbox will return `success` immediately upon refund request,
// whereas Live will initially return `submitted`. GovPay then needs to be called again to check
// if the status is updated to `success`.
func mapGovPayStatusToInternal(status string) string {

	govPayStatusMap := map[string]string{
		"submitted": "refund-requested",
		"success":   "refund-success",
		"error":     "refund-error",
	}

	if govPayStatusMap[status] != "" {
		return govPayStatusMap[status]
	}

	// shouldn't end up here
	log.Error(fmt.Errorf("unexpected refund status returned from Gov Pay: %s", status))
	return status

}
