package service

import (
	"errors"
	"fmt"
	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
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
	GovPayService GovPayService
	DAO           dao.DAO
	Config        config.Config
}

func (service *RefundService) CreateRefund(req *http.Request, id string, createRefundResource models.CreateRefundRequest) (*models.CreateRefundGovPayResponse, ResponseType, error) {

	// Get RefundSummary from GovPay to check the available amount
	paymentSession, refundSummary, response, err := service.GovPayService.GetGovPayRefundSummary(req, id)
	if err != nil {
		err = fmt.Errorf("error getting refund summary from govpay: [%v]", err)
		log.ErrorR(req, err)
		return nil, response, err
	}

	if refundSummary.AmountAvailable < createRefundResource.Amount {
		err = errors.New("refund amount is higher than available amount")
		return nil, InvalidData, err
	}

	refundRequest := models.CreateRefundGovPayRequest{}
	refundRequest.Amount = createRefundResource.Amount
	refundRequest.RefundAmountAvailable = refundSummary.AmountAvailable

	// Call GovPay to initiate a Refund
	refund, response, err := service.GovPayService.CreateRefund(paymentSession, refundRequest)
	if err != nil {
		err = fmt.Errorf("error creating refund in govpay: [%v]", err)
		log.ErrorR(req, err)
		return nil, response, err
	}

	return refund, Success, nil
}
