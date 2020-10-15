package fixtures

import "github.com/companieshouse/payments.api.ch.gov.uk/models"

func GetCreateRefundGovPayResponse() *models.CreateRefundGovPayResponse {
	return &models.CreateRefundGovPayResponse{
		RefundId:    "ABC",
		CreatedDate: "",
		Amount:      8,
		Links:       models.GovPayRefundLinks{},
		Status:      "success",
	}
}

func GetCreateRefundGovPayRequest(amount int, amountAvailable int) *models.CreateRefundGovPayRequest {
	return &models.CreateRefundGovPayRequest{
		Amount:                amount,
		RefundAmountAvailable: amountAvailable,
	}
}
