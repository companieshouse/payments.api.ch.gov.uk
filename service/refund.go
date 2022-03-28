package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

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

// BulkRefundStatus Enum Type
type BulkRefundStatus int

// Enumeration containing all possible bulk refund statuses
const (
	BulkRefundPending BulkRefundStatus = 1 + iota
	BulkRefundRequested
)

// String representation of bulk refund statuses
var bulkRefundStatuses = [...]string{
	"refund-pending",
	"refund-requested",
}

// String returns the string representation of the bulk refund status
func (bulkRefundStatus BulkRefundStatus) String() string {
	return bulkRefundStatuses[bulkRefundStatus-1]
}

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

// ValidateGovPayBatchRefund retrieves all the payments in the batch refund
// and validates it before processing it
func (service *RefundService) ValidateGovPayBatchRefund(ctx context.Context, batchRefund models.GovPayRefundBatch) ([]string, error) {
	var validationErrors []string
	var mu = sync.Mutex{}
	errs, _ := errgroup.WithContext(ctx)
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

	if paymentSession.Data.Status != Paid.String() {
		return fmt.Sprintf("payment with order code [%s] has a status of [%s] - refund not eligible", refund.OrderCode, paymentSession.Data.Status)
	}

	return ""
}

// UpdateGovPayBatchRefund updates each paymentSession in the DB corresponding
// to the refunds in the batch refund file with the necessary refund information
func (service *RefundService) UpdateGovPayBatchRefund(ctx context.Context, batchRefund models.GovPayRefundBatch, filename string, user string) error {
	errs, _ := errgroup.WithContext(ctx)
	for _, refund := range batchRefund.GovPayRefunds {
		r := refund
		errs.Go(func() error {

			bulkRefundDB := models.BulkRefundDB{
				Status:            BulkRefundPending.String(),
				UploadedFilename:  filename,
				UploadedAt:        time.Now().Truncate(time.Millisecond).String(),
				UploadedBy:        user,
				Amount:            r.Amount.Value,
				RefundID:          "",
				ProcessedAt:       "",
				ExternalRefundURL: "",
			}

			err := service.DAO.CreateBulkRefund(r.OrderCode, bulkRefundDB)
			if err != nil {
				log.Error(fmt.Errorf("error updating payment session in DB: %w", err))
				return err
			}

			return nil
		})
	}

	if err := errs.Wait(); err != nil {
		return err
	}

	return nil
}

// ProcessBatchRefund processes all refunds in the DB with a refund-pending status
func (service *RefundService) ProcessBatchRefund(req *http.Request) []error {
	var errorList []error
	payments, err := service.DAO.GetPaymentsWithRefundStatus()
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error retrieving payments with refund-pending status: %w", err))
		errorList = append(errorList, errors.New("error retrieving payments with refund-pending status"))
		return errorList
	}
	if len(payments) == 0 {
		log.ErrorR(req, errors.New("no payments with refund-pending status found"))
		errorList = append(errorList, errors.New("no payments with refund-pending status found"))
		return errorList
	}

	for _, p := range payments {
		if p.Data.PaymentMethod == "credit-card" {
			err := service.processGovPayBatchRefund(req, p)
			if err != nil {
				errorList = append(errorList, err)
			}
		}
	}

	return errorList
}

func (service *RefundService) processGovPayBatchRefund(req *http.Request, payment models.PaymentResourceDB) error {
	recentRefund := payment.BulkRefund[len(payment.BulkRefund)-1]
	a := strings.Replace(recentRefund.Amount, ".", "", -1)
	amount, err := strconv.Atoi(a)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error converting amount string to int [%w]", err))
		return fmt.Errorf("error converting amount string to int for payment with id [%s]", payment.ID)
	}
	// Get RefundSummary from GovPay to check the available amount
	paymentSession, refundSummary, _, err := service.GovPayService.GetRefundSummary(req, payment.ID)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error getting refund summary from govpay: [%w]", err))
		return fmt.Errorf("error getting refund summary from govpay for payment with id [%s]", payment.ID)
	}

	if refundSummary.AmountAvailable != amount {
		err := fmt.Errorf("refund amount is not equal than available amount for payment with id [%s]", payment.ID)
		log.ErrorR(req, err)
		return err
	}

	refundRequest := &models.CreateRefundGovPayRequest{
		Amount:                amount,
		RefundAmountAvailable: refundSummary.AmountAvailable,
	}
	// Call GovPay to initiate a Refund
	refund, _, err := service.GovPayService.CreateRefund(paymentSession, refundRequest)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error creating refund in govpay: [%w]", err))
		return fmt.Errorf("error creating refund in govpay for payment with id [%s]", payment.ID)
	}

	refundResource := mappers.MapGovPayToRefundResponse(*refund)

	recentRefund.RefundID = refundResource.RefundId
	recentRefund.ProcessedAt = refundResource.CreatedDateTime
	recentRefund.Status = RefundRequested.String()
	recentRefund.ExternalRefundURL = payment.ExternalPaymentStatusURI + "/refund"
	payment.BulkRefund[len(payment.BulkRefund)-1] = recentRefund
	err = service.DAO.PatchPaymentResource(payment.ID, &payment)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error patching payment [%w]", err))
		return fmt.Errorf("error patching payment with id [%s]", payment.ID)
	}

	return nil
}
