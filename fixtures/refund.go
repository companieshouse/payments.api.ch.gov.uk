package fixtures

import "github.com/companieshouse/payments.api.ch.gov.uk/models"

var lateFilingPenalty = "Late Filing Penalty"
var refundPending = "refund-pending"
var paymentSessionKind = "payment-session#payment-session"

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

// GetPendingRefundPayments returns an array of PaymentResourceDB with refund-pending status
func GetPendingRefundPayments() []models.PaymentResourceDB {
	data1 := models.PaymentResourceDataDB{
		Amount:        "150.00",
		Description:   lateFilingPenalty,
		PaymentMethod: "GovPay",
		Reference:     "late_filing_penalty_OR04238448",
		CompanyNumber: "10000025",
		Status:        refundPending,
		Etag:          "63174d4d675c75d458fe192ca805e76873eb46611e137e572398f33b",
		Kind:          paymentSessionKind,
	}
	data2 := models.PaymentResourceDataDB{
		Amount:        "300.00",
		Description:   lateFilingPenalty,
		PaymentMethod: "GovPay",
		Reference:     "late_filing_penalty_OR04238453",
		CompanyNumber: "10000030",
		Status:        refundPending,
		Etag:          "63174d4d675c75d458fe192ca805e76873eb46611e137e572398d33a",
		Kind:          paymentSessionKind,
	}
	paymentResource1 := models.PaymentResourceDB{ID: "1234", Data: data1}
	paymentResource2 := models.PaymentResourceDB{ID: "1234", Data: data2}
	paymentResources := []models.PaymentResourceDB{paymentResource1, paymentResource2}

	return paymentResources
}

// GetPendingRefundPayments returns an array of PendingRefundPaymentsResourceRest with refund-pending status
func GetPendingRefundsResponse() *models.PendingRefundPaymentsResourceRest {
	paymentResource1 := models.PaymentResourceRest{
		Amount:        "150.00",
		Description:   lateFilingPenalty,
		PaymentMethod: "GovPay",
		Reference:     "late_filing_penalty_OR04238448",
		CompanyNumber: "10000025",
		Status:        refundPending,
		Etag:          "63174d4d675c75d458fe192ca805e76873eb46611e137e572398f33b",
		Kind:          paymentSessionKind,
	}
	paymentResource2 := models.PaymentResourceRest{
		Amount:        "300.00",
		Description:   lateFilingPenalty,
		PaymentMethod: "GovPay",
		Reference:     "late_filing_penalty_OR04238453",
		CompanyNumber: "10000030",
		Status:        refundPending,
		Etag:          "63174d4d675c75d458fe192ca805e76873eb46611e137e572398d33a",
		Kind:          paymentSessionKind,
	}
	paymentResources := []models.PaymentResourceRest{paymentResource1, paymentResource2}

	return &models.PendingRefundPaymentsResourceRest{
		Total:    2,
		Payments: paymentResources,
	}
}
