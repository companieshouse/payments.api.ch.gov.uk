package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/mappers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/transformers"
)

const (
	RefundPending          = "pending"
	RefundUnavailable      = "unavailable"
	RefundAvailable        = "available"
	RefundFull             = "full"
	RefundsStatusSuccess   = "success"
	RefundsStatusSubmitted = "submitted"
	RefundsStatusError     = "error"
)

type RefundService struct {
	GovPayService  PaymentProviderService
	PaymentService *PaymentService
	DAO            dao.DAO
	Config         config.Config
}

// CreateRefund creates refund in GovPay and saves refund information to payment object in mongo
func (service *RefundService) CreateRefund(req *http.Request, id string, createRefundResource models.CreateRefundRequest) (*models.PaymentResourceRest, *models.RefundResponse, ResponseType, error) {

	// Get RefundSummary from GovPay to check the available amount
	paymentSession, refundSummary, response, err := service.GovPayService.GetRefundSummary(req, id)
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

	refundResource := mappers.MapGovPayToRefundResponse(*refund)

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

// UpdateRefund checks refund status in GovPay and if status is successful saves it to payment object in mongo
func (service *RefundService) UpdateRefund(req *http.Request, paymentId string, refundId string) (*models.RefundResourceRest, ResponseType, error) {
	paymentSession, response, err := service.PaymentService.GetPaymentSession(req, paymentId)
	if err != nil {
		err = fmt.Errorf("error getting payment resource: [%v]", err)
		log.ErrorR(req, err)
		return nil, response, err
	}

	if response == NotFound {
		err = fmt.Errorf("error getting payment resource")
		log.ErrorR(req, err)

		return nil, NotFound, err
	}

	index, err := getRefundIndex(paymentSession.Refunds, refundId)

	if err != nil {
		log.ErrorR(req, err)
		return nil, NotFound, err
	}
	// Get RefundStatus from GovPay to check the status of the refund
	govPayStatusResponse, response, err := service.GovPayService.GetRefundStatus(paymentSession, refundId)
	if err != nil {
		err = fmt.Errorf("error getting refund status from govpay: [%v]", err)
		log.ErrorR(req, err)
		return nil, response, err
	}

	paymentSession.Refunds[index].Status = govPayStatusResponse.Status

	paymentResourceUpdate := transformers.PaymentTransformer{}.TransformToDB(*paymentSession)

	err = service.DAO.PatchPaymentResource(paymentId, &paymentResourceUpdate)
	if err != nil {
		err = fmt.Errorf("error patching payment session to database: [%v]", err)
		log.Error(err)
		return nil, Error, err
	}

	return &paymentSession.Refunds[index], Success, nil
}

func getRefundIndex(refunds []models.RefundResourceRest, refundId string) (int, error) {
	for i, ref := range refunds {
		if ref.RefundId == refundId {
			return i, nil
		}
	}
	return -1, errors.New("refund id not found in payment refunds")
}

// ProcessGovPayBatchRefund retrieves all the payments in the batch refund
// and validates it before processing it
func (service *RefundService) ProcessGovPayBatchRefund(ctx context.Context, batchRefund models.GovPayRefundBatch) ([]string, error) {
	var validationErrors []string
	var mu = sync.Mutex{}
	errs, ctx := errgroup.WithContext(ctx)
	for _, refund := range batchRefund.GovPayRefunds {
		r := refund
		errs.Go(func() error {
			paymentSession, err := service.DAO.GetPaymentResourceByExternalPaymentStatusID(r.OrderCode)
			if err != nil {
				log.Error(fmt.Errorf("error retrieving payment session from DB: %w", err))
				return err
			}

			if validationError := validateGovPayRefund(paymentSession, r); validationError != "" {
				mu.Lock()
				validationErrors = append(validationErrors, validationError)
				mu.Unlock()
			}

			return nil
		})
	}

	// Return early if the errgroup returned an error
	// when fetching a paymentSession from the DB
	if err := errs.Wait(); err != nil {
		return nil, err
	}

	return validationErrors, nil
}

func validateGovPayRefund(paymentSession *models.PaymentResourceDB, refund models.GovPayRefund) string {
	if paymentSession == nil {
		return fmt.Sprintf("payment session with id [%s] not found", refund.OrderCode)
	}

	if paymentSession.Data.PaymentMethod != "credit-card" {
		return fmt.Sprintf("payment with order code [%s] has not been made via Gov.Pay - refund not eligible", refund.OrderCode)
	}

	if paymentSession.Data.Amount != refund.Amount.Value {
		return fmt.Sprintf("value of refund with order code [%s] does not match payment", refund.OrderCode)
	}

	return ""
}
