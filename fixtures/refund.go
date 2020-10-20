package fixtures

import "github.com/companieshouse/payments.api.ch.gov.uk/models"

func GetRefundRequest(amount int) models.CreateRefundRequest {
	return models.CreateRefundRequest{Amount: amount}
}

func GetRefundSummary(amount int) *models.RefundSummary {
	return &models.RefundSummary{
		Status:          "available",
		AmountAvailable: amount,
		AmountSubmitted: 0,
	}
}
