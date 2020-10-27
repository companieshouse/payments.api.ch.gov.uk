package service

import (
	"errors"
	"fmt"
	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/mappers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/transformers"
	"net/http"
)

type RefundStatus string

const (
	RefundPending     = "pending"
	RefundUnavailable = "unavailable"
	RefundAvailable   = "available"
	RefundFull        = "full"
)

type RefundService struct {
	GovPayService PaymentProviderService
	DAO           dao.DAO
	Config        config.Config
}

// CreateRefund creates refund in GovPay and saves refund information to payment object in mongo
func (service *RefundService) CreateRefund(req *http.Request, id string, createRefundResource models.CreateRefundRequest) (*models.PaymentResourceRest, *models.CreateRefundResponse, ResponseType, error) {

	// Get RefundSummary from GovPay to check the available amount
	paymentSession, refundSummary, response, err := service.GovPayService.GetGovPayRefundSummary(req, id)
	if err != nil {
		err = fmt.Errorf("error getting refund summary from govpay: [%v]", err)
		log.ErrorR(req, err)
		return nil, nil, response, err
	}

	if refundSummary.AmountAvailable < createRefundResource.Amount {
		err = errors.New("refund amount is higher than available amount")
		return nil, nil, InvalidData, err
	}

	refundRequest := &models.CreateRefundGovPayRequest{
		Amount:                createRefundResource.Amount,
		RefundAmountAvailable: refundSummary.AmountAvailable,
	}

	// Call GovPay to initiate a Refund
	refund, response, err := service.GovPayService.CreateRefund(paymentSession, refundRequest)
	if err != nil {
		err = fmt.Errorf("error creating refund in govpay: [%v]", err)
		log.ErrorR(req, err)
		return nil, nil, response, err
	}

	refundResource := mappers.MapToRefundResponse(*refund)

	// Add refund information to payment session
	paymentSession.Refunds = append(paymentSession.Refunds, mappers.MapToRefundRest(*refund))
	paymentResourceUpdate := transformers.PaymentTransformer{}.TransformToDB(*paymentSession)

	// Save refund information to mongoDB
	err = service.DAO.PatchPaymentResource(id, &paymentResourceUpdate)
	if err != nil {
		err = fmt.Errorf("error patching payment session on database: [%v]", err)
		log.Error(err)
		return nil, nil, Error, err
	}

	return paymentSession, &refundResource, Success, nil
}
