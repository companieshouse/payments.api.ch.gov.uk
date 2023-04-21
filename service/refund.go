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

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/mappers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/transformers"
	"golang.org/x/sync/errgroup"
)

const (
	RefundPending           = "pending"
	RefundUnavailable       = "unavailable"
	RefundAvailable         = "available"
	RefundFull              = "full"
	RefundsStatusSuccess    = "success"
	RefundsStatusSubmitted  = "submitted"
	RefundsStatusError      = "error"
	PaymentMethodCreditCard = "credit-card"
	PaymentMethodPayPal     = "PayPal"
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
	PayPalService  PaymentProviderService
	PaymentService *PaymentService
	DAO            dao.DAO
	Config         config.Config
}

// CreateRefund creates refund in GovPay and saves refund information to payment object in database
func (service *RefundService) CreateRefund(req *http.Request, paymentID string, createRefundResource models.CreateRefundRequest) (*models.PaymentResourceRest, *models.RefundResponse, ResponseType, error) {

	paymentSession, _, _ := service.PaymentService.GetPaymentSession(req, paymentID)

	// Currently, refunds are only enabled for Gov Pay
	if paymentSession.PaymentMethod != PaymentMethodCreditCard {
		err := fmt.Errorf("unexpected payment method: %s", paymentSession.PaymentMethod)
		return nil, nil, Forbidden, err
	}

	// return error if refund reference is not unique
	// unless other uses of this refund reference are cancelled or failed
	if paymentSession.Refunds != nil {
		for _, refund := range paymentSession.Refunds {
			// RefundsStatusError is returned by GovPay if a refund has failed to be processed
			if refund.RefundReference == createRefundResource.RefundReference && refund.Status != RefundsStatusError {
				err := fmt.Errorf("duplicate refund reference found: %s", refund.RefundReference)
				return nil, nil, Conflict, err
			}
		}
	}

	// Get RefundSummary from GovPay to check the available amount
	paymentSession, refundSummary, response, err := service.GovPayService.GetRefundSummary(req, paymentID)
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

	// GOV.UK Pay returns different refund statuses in Sandbox and Live.
	// Hard-coding the initial status here enables testing in Sandbox.
	// https://docs.payments.service.gov.uk/refunding_payments/
	if service.Config.GovPaySandbox {
		log.Info("GOV.UK Pay sandbox enabled for test environment: hard-coding initial refund status to `submitted`")
		refund.Status = "submitted"
	}

	refundResource := mappers.MapGovPayToRefundResponse(*refund)

	// Add refund information to payment session
	paymentSession.Refunds = append(paymentSession.Refunds, mappers.MapToRefundRest(*refund, createRefundResource.RefundReference))
	paymentSession.Links.Refunds = fmt.Sprintf("%s/payments/%s/refunds", service.Config.PaymentsAPIURL, paymentID)
	paymentResourceUpdate := transformers.PaymentTransformer{}.TransformToDB(*paymentSession)

	// Save refund information to database
	err = service.DAO.PatchPaymentResource(paymentID, &paymentResourceUpdate)
	if err != nil {
		err = fmt.Errorf("error patching payment session on database: [%v]", err)
		log.Error(err)
		return nil, nil, Error, err
	}

	return paymentSession, &refundResource, Success, nil
}

// GetPaymentRefunds processes all refunds in the DB by paymentId
func (service *RefundService) GetPaymentRefunds(req *http.Request, paymentId string) ([]models.RefundResourceDB, error) {

	refunds, err := service.DAO.GetPaymentRefunds(paymentId)

	if err != nil {
		log.ErrorR(req, fmt.Errorf("error retrieving the payment refunds with : %w", err))
		return nil, errors.New("error retrieving the payment refunds")
	}

	if len(refunds) == 0 {
		log.ErrorR(req, fmt.Errorf("no refunds with paymentId: %s found", paymentId))
		return nil, errors.New("no refunds with paymentId found")
	}

	return refunds, nil
}

// UpdateRefund checks refund status in GovPay and if status is successful saves it to payment object in database
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

// ValidateBatchRefund retrieves all the payments in the batch refund
// and validates it before processing it
func (service *RefundService) ValidateBatchRefund(ctx context.Context, batchRefund models.RefundBatch) ([]string, error) {
	var validationErrors []string
	var mu = sync.Mutex{}
	errs, _ := errgroup.WithContext(ctx)
	for _, refund := range batchRefund.RefundDetails {
		r := refund
		errs.Go(func() error {

			var paymentSession *models.PaymentResourceDB
			var validationError string
			var err error

			switch batchRefund.PaymentProvider {
			case "govpay":
				paymentSession, err = service.DAO.GetPaymentResourceByProviderID(r.OrderCode)
				if err != nil {
					log.Error(fmt.Errorf("error retrieving payment session from DB: %w", err))
					return err
				}
				validationError = validateGovPayRefund(paymentSession, r)
			case "paypal":
				paymentSession, err := service.DAO.GetPaymentResourceByExternalPaymentTransactionID(r.OrderCode)
				if err != nil {
					log.Error(fmt.Errorf("error retrieving payment session from DB: %w", err))
					return err
				}
				validationError = validatePayPalRefund(paymentSession, r)
			default:
				return fmt.Errorf("invalid payment provider supplied: %s", batchRefund.PaymentProvider)
			}

			if validationError != "" {
				mu.Lock()
				validationErrors = append(validationErrors, validationError)
				mu.Unlock()
			}

			return nil
		})
	}

	// Return early if the errgroup returned an error
	// when fetching a paymentSession from the DB
	err := errs.Wait()

	return validationErrors, err
}

func validateGovPayRefund(paymentSession *models.PaymentResourceDB, refund models.RefundDetails) string {
	if paymentSession == nil {
		return fmt.Sprintf("payment session with id [%s] not found", refund.OrderCode)
	}

	if paymentSession.Data.PaymentMethod != PaymentMethodCreditCard {
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

func validatePayPalRefund(paymentSession *models.PaymentResourceDB, refund models.RefundDetails) string {
	if paymentSession == nil {
		return fmt.Sprintf("payment session with id [%s] not found", refund.OrderCode)
	}

	if paymentSession.Data.PaymentMethod != PaymentMethodPayPal {
		return fmt.Sprintf("payment with order code [%s] has not been made via PayPal - refund not eligible", refund.OrderCode)
	}

	if paymentSession.Data.Amount != refund.Amount.Value {
		return fmt.Sprintf("value of refund with order code [%s] does not match payment", refund.OrderCode)
	}

	if paymentSession.Data.Status != Paid.String() {
		return fmt.Sprintf("payment with order code [%s] has a status of [%s] - refund not eligible", refund.OrderCode, paymentSession.Data.Status)
	}

	return ""
}

// UpdateBatchRefund updates each paymentSession in the DB corresponding
// to the refunds in the batch refund file with the necessary refund information
func (service *RefundService) UpdateBatchRefund(ctx context.Context, batchRefund models.RefundBatch, filename string, user string) error {

	bulkRefunds := make(map[string]models.BulkRefundDB)

	for _, refund := range batchRefund.RefundDetails {
		bulkRefundDB := models.BulkRefundDB{
			Status:            BulkRefundPending.String(),
			UploadedFilename:  filename,
			UploadedAt:        time.Now().Truncate(time.Millisecond).String(),
			UploadedBy:        user,
			Amount:            refund.Amount.Value,
			RefundID:          "",
			ProcessedAt:       "",
			ExternalRefundURL: "",
		}

		bulkRefunds[refund.OrderCode] = bulkRefundDB
	}

	var err error
	switch batchRefund.PaymentProvider {
	case "govpay":
		err = service.DAO.CreateBulkRefundByProviderID(bulkRefunds)
	case "paypal":
		err = service.DAO.CreateBulkRefundByExternalPaymentTransactionID(bulkRefunds)
	default:
		err = fmt.Errorf("invalid payment provider: [%s]", batchRefund.PaymentProvider)
	}

	if err != nil {
		log.Error(fmt.Errorf("error updating payment session in DB: %w", err))
		return err
	}

	return nil
}

// GetPaymentsWithPendingRefundStatus gets all payment sessions in the DB that
// have the pending refund status
func (service *RefundService) GetPaymentsWithPendingRefundStatus() (*models.PendingRefundPaymentsResourceRest, error) {
	paymentSessions, err := service.DAO.GetPaymentsWithRefundStatus()
	if err != nil {
		err = fmt.Errorf("error getting payment resources with pending refund status: [%v]", err)
		log.Error(err)
		return nil, err
	}

	paymentSessionsRest := []models.PaymentResourceRest{}
	for _, paymentSession := range paymentSessions {
		paymentSessionsRest = append(paymentSessionsRest, transformers.PaymentTransformer{}.TransformToRest(paymentSession))
	}

	pendingRefundPayments := models.PendingRefundPaymentsResourceRest{Payments: paymentSessionsRest, Total: len(paymentSessionsRest)}

	return &pendingRefundPayments, nil
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
		switch p.Data.PaymentMethod {
		case PaymentMethodCreditCard:
			err := service.processGovPayBatchRefund(req, p)
			if err != nil {
				errorList = append(errorList, err)
			}
		case PaymentMethodPayPal:
			err := service.processPayPalBatchRefund(req, p)
			if err != nil {
				errorList = append(errorList, err)
			}
		default:
			err := fmt.Errorf("invalid payment method [%s] for Payment ID %s", p.Data.PaymentMethod, p.ID)
			errorList = append(errorList, err)
		}
	}

	return errorList
}

// ProcessPendingRefunds processes all refunds in the DB with a pending status
func (service *RefundService) ProcessPendingRefunds(req *http.Request) ([]models.PaymentResourceDB, ResponseType, []error) {
	var errorList []error
	payments, err := service.DAO.GetPaymentsWithRefundPendingStatus()

	if err != nil {
		log.ErrorR(req, fmt.Errorf("error retrieving payments with : %w", err))
		errorList = append(errorList, errors.New("error retrieving payments with refund pending status"))
		return nil, Error, errorList
	}

	if len(payments) == 0 {
		log.ErrorR(req, errors.New("no payments with refund pending status found"))
		errorList = append(errorList, errors.New("no payments with refund pending status found"))
		return nil, Success, errorList
	}

	payments = service.checkGovPayAndUpdateRefundStatus(req, payments)

	return payments, Success, nil
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
		err := fmt.Errorf("refund amount is not equal to available amount for payment with id [%s]", payment.ID)
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

func (service *RefundService) processPayPalBatchRefund(req *http.Request, payment models.PaymentResourceDB) error {
	recentRefund := payment.BulkRefund[len(payment.BulkRefund)-1]
	captureID := payment.ExternalPaymentTransactionID

	// Get Captured Details Response from PayPal to check the status and  available amount
	captureDetailsResponse, err := service.PayPalService.GetCapturedPaymentDetails(captureID)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error getting capture details from paypal: [%w]", err))
		return fmt.Errorf("error getting capture details from paypal for payment ID [%s]", payment.ID)
	}

	if captureDetailsResponse.Status != "COMPLETED" {
		err := fmt.Errorf("captured payment status [%s] is not complete for payment ID [%s]", captureDetailsResponse.Status, payment.ID)
		log.ErrorR(req, err)
		return err
	}

	if captureDetailsResponse.Amount.Value != payment.Data.Amount {
		err := fmt.Errorf("refund amount is not equal to available amount for payment ID [%s]", payment.ID)
		log.ErrorR(req, err)
		return err
	}

	// Send Refund Capture request to PayPal
	refundResponse, err := service.PayPalService.RefundCapture(captureID)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error creating refund in PayPal: [%w]", err))
		return fmt.Errorf("error creating refund in PayPal for payment with id [%s]", payment.ID)
	}

	if refundResponse.Status != "COMPLETED" {
		log.ErrorR(req, fmt.Errorf("refund incomplete. Status [%s] returned from PayPal for Payment ID [%s]", refundResponse.Status, payment.ID))
		return fmt.Errorf("error completing refund in PayPal for payment with id [%s]", payment.ID)
	}

	// Patch refund details to DB
	recentRefund.RefundID = refundResponse.ID
	recentRefund.ProcessedAt = time.Now().String()
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

func (service *RefundService) checkGovPayAndUpdateRefundStatus(req *http.Request, payments []models.PaymentResourceDB) []models.PaymentResourceDB {
	var updatedPayments []models.PaymentResourceDB
	for _, i := range payments {
		x := i
		if x.Refunds != nil {
			refund := x.Refunds[0]
			paymentSession, response, err := service.PaymentService.GetPaymentSession(req, x.ID)

			if err != nil {
				log.ErrorR(req, fmt.Errorf("error getting payment resource ID [%s]: [%w]", x.ID, err))
				err := service.DAO.IncrementRefundAttempts(x.ID, &x)
				if err != nil {
					log.ErrorR(req, fmt.Errorf("error incrementing attempts in DB: [%w]", err))
				}
				continue
			}

			if response == NotFound {
				log.ErrorR(req, fmt.Errorf("not found error from payment service session with payment ID:[%s]", x.ID))
				err := service.DAO.IncrementRefundAttempts(x.ID, &x)
				if err != nil {
					log.ErrorR(req, fmt.Errorf("error incrementing attempts in DB: [%w]", err))
				}
				continue
			}

			govPayStatusResponse, response, err := service.GovPayService.GetRefundStatus(paymentSession, refund.RefundId)

			if err != nil {
				log.ErrorR(req, fmt.Errorf("error getting refund status for ID [%s] [%w]", refund.RefundId, err))
				err := service.DAO.IncrementRefundAttempts(x.ID, &x)
				if err != nil {
					log.ErrorR(req, fmt.Errorf("error incrementing attempts in DB: [%w]", err))
				}
				continue
			}

			isRefunded := govPayStatusResponse != nil && govPayStatusResponse.Status == RefundsStatusSuccess

			payment, err := service.DAO.PatchRefundSuccessStatus(x.ID, isRefunded, &x)
			if err != nil {
				log.ErrorR(req, fmt.Errorf("error patching payment ID [%s] [%w]", x.ID, err))
				// No need to error out here, can continue to reconciliation
			}

			if payment.Refunds[0].Status == "refund-success" {
				updatedPayments = append(updatedPayments, payment)
			}
		} else {
			log.ErrorR(req, fmt.Errorf("no refund found with payment Id: [%s]", x.ID))
		}
	}

	return updatedPayments
}
