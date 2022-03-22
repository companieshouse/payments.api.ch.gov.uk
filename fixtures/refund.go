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

func GetPendingRefundPayments() []models.PaymentResourceDB {
	data1 := models.PaymentResourceDataDB{
		Amount:        "150.00",
		Description:   "Late Filing Penalty",
		PaymentMethod: "GovPay",
		Reference:     "late_filing_penalty_OR04238448",
		CompanyNumber: "10000025",
		Status:        "refund-pending",
		Etag:          "63174d4d675c75d458fe192ca805e76873eb46611e137e572398f33b",
		Kind:          "payment-session#payment-session",
	}
	data2 := models.PaymentResourceDataDB{
		Amount:        "300.00",
		Description:   "Late Filing Penalty",
		PaymentMethod: "GovPay",
		Reference:     "late_filing_penalty_OR04238453",
		CompanyNumber: "10000030",
		Status:        "refund-pending",
		Etag:          "63174d4d675c75d458fe192ca805e76873eb46611e137e572398d33a",
		Kind:          "payment-session#payment-session",
	}
	paymentResource1 := models.PaymentResourceDB{ID: "1234", Data: data1}
	paymentResource2 := models.PaymentResourceDB{ID: "1234", Data: data2}
	paymentResources := []models.PaymentResourceDB{paymentResource1, paymentResource2}

	return paymentResources
}

func GetPendingRefundsResponse() *models.PendingRefundPaymentsResourceRest {
	paymentResource1 := models.PaymentResourceRest{
		Amount:        "150.00",
		Description:   "Late Filing Penalty",
		PaymentMethod: "GovPay",
		Reference:     "late_filing_penalty_OR04238448",
		CompanyNumber: "10000025",
		Status:        "refund-pending",
		Etag:          "63174d4d675c75d458fe192ca805e76873eb46611e137e572398f33b",
		Kind:          "payment-session#payment-session",
	}
	paymentResource2 := models.PaymentResourceRest{
		Amount:        "300.00",
		Description:   "Late Filing Penalty",
		PaymentMethod: "GovPay",
		Reference:     "late_filing_penalty_OR04238453",
		CompanyNumber: "10000030",
		Status:        "refund-pending",
		Etag:          "63174d4d675c75d458fe192ca805e76873eb46611e137e572398d33a",
		Kind:          "payment-session#payment-session",
	}
	paymentResources := []models.PaymentResourceRest{paymentResource1, paymentResource2}

	return &models.PendingRefundPaymentsResourceRest{
		Total:    2,
		Payments: paymentResources,
	}
}
