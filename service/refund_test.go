package service

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/fixtures"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/golang/mock/gomock"
	"github.com/jarcoal/httpmock"
	"github.com/plutov/paypal/v4"
	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitCreateRefund(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()

	mockDao := dao.NewMockDAO(mockCtrl)
	mockGovPayService := NewMockPaymentProviderService(mockCtrl)
	mockPaymentService := createMockPaymentService(mockDao, cfg)

	service := RefundService{
		GovPayService:  mockGovPayService,
		PaymentService: &mockPaymentService,
		DAO:            mockDao,
		Config:         *cfg,
	}

	req := httptest.NewRequest("POST", "/test", nil)
	id := "123"

	Convey("Error when getting payment session", t, func() {
		body := fixtures.GetRefundRequest(8)

		payment := generatePaymentSession("invalid-method")
		payment.Data.Links = models.PaymentLinksDB{Resource: "http://dummy-resource"}
		mockDao.EXPECT().GetPaymentResource(gomock.Any()).Return(&payment, nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		paymentSession, refund, status, err := service.CreateRefund(req, id, body)

		So(paymentSession, ShouldBeNil)
		So(refund, ShouldBeNil)
		So(status, ShouldEqual, Forbidden)
		So(err.Error(), ShouldEqual, "unexpected payment method: invalid-method")
	})

	Convey("Refund conflict", t, func() {
		body := fixtures.GetRefundRequest(8)
		body.RefundReference = "123"

		payment := generatePaymentSessionGovPay()
		payment.Data.Links = models.PaymentLinksDB{Resource: "http://dummy-resource"}
		refunds := []models.RefundResourceDB{
			{
				RefundId:          "",
				CreatedAt:         "",
				Amount:            0,
				Status:            "",
				ExternalRefundUrl: "",
				RefundReference:   "123",
			},
		}
		payment.Refunds = refunds
		mockDao.EXPECT().GetPaymentResource(gomock.Any()).Return(&payment, nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		paymentSession, refund, status, err := service.CreateRefund(req, id, body)

		So(paymentSession, ShouldBeNil)
		So(refund, ShouldBeNil)
		So(status, ShouldEqual, Conflict)
		So(err.Error(), ShouldEqual, "duplicate refund reference found: 123")
	})

	Convey("Error when getting GovPay refund summary", t, func() {
		body := fixtures.GetRefundRequest(8)
		err := fmt.Errorf("error getting payment resource")

		payment := generatePaymentSessionGovPay()
		fmt.Println(payment.Data.PaymentMethod)
		payment.Data.Links = models.PaymentLinksDB{Resource: "http://dummy-resource"}
		mockDao.EXPECT().GetPaymentResource(gomock.Any()).Return(&payment, nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		mockGovPayService.EXPECT().
			GetRefundSummary(req, id).
			Return(nil, nil, Error, err)

		paymentSession, refund, status, err := service.CreateRefund(req, id, body)

		So(paymentSession, ShouldBeNil)
		So(refund, ShouldBeNil)
		So(status, ShouldEqual, Error)
		So(err.Error(), ShouldEqual, "error getting refund summary from govpay: [error getting payment resource]")
	})

	Convey("Error because amount is higher than amount available", t, func() {
		body := fixtures.GetRefundRequest(8)
		refundSummary := fixtures.GetRefundSummary(4)

		payment := generatePaymentSessionGovPay()
		payment.Data.Links = models.PaymentLinksDB{Resource: "http://dummy-resource"}
		mockDao.EXPECT().GetPaymentResource(gomock.Any()).Return(&payment, nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		mockGovPayService.EXPECT().
			GetRefundSummary(req, id).
			Return(nil, refundSummary, Success, nil)

		paymentSession, refund, status, err := service.CreateRefund(req, id, body)

		So(paymentSession, ShouldBeNil)
		So(refund, ShouldBeNil)
		So(status, ShouldEqual, InvalidData)
		So(err.Error(), ShouldEqual, "refund amount is higher than available amount")
	})

	Convey("Error creating refund in GovPay", t, func() {
		body := fixtures.GetRefundRequest(8)
		refundSummary := fixtures.GetRefundSummary(8)
		paymentResource := &models.PaymentResourceRest{}
		refundRequest := fixtures.GetCreateRefundGovPayRequest(body.Amount, refundSummary.AmountAvailable)

		payment := generatePaymentSessionGovPay()
		payment.Data.Links = models.PaymentLinksDB{Resource: "http://dummy-resource"}
		mockDao.EXPECT().GetPaymentResource(gomock.Any()).Return(&payment, nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		err := fmt.Errorf("error reading refund GovPayRequest")

		mockGovPayService.EXPECT().
			GetRefundSummary(req, id).
			Return(paymentResource, refundSummary, Success, nil)

		mockGovPayService.EXPECT().
			CreateRefund(paymentResource, refundRequest).
			Return(nil, Error, err)

		paymentSession, refund, status, err := service.CreateRefund(req, id, body)

		So(paymentSession, ShouldBeNil)
		So(refund, ShouldBeNil)
		So(status, ShouldEqual, Error)
		So(err.Error(), ShouldEqual, "error creating refund in govpay: [error reading refund GovPayRequest]")
	})

	Convey("Error patching payment session", t, func() {
		body := fixtures.GetRefundRequest(8)
		refundSummary := fixtures.GetRefundSummary(8)
		paymentResource := &models.PaymentResourceRest{}
		refundRequest := fixtures.GetCreateRefundGovPayRequest(body.Amount, refundSummary.AmountAvailable)
		response := fixtures.GetCreateRefundGovPayResponse()

		payment := generatePaymentSessionGovPay()
		payment.Data.Links = models.PaymentLinksDB{Resource: "http://dummy-resource"}
		mockDao.EXPECT().GetPaymentResource(gomock.Any()).Return(&payment, nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		err := fmt.Errorf("error reading refund GovPayRequest")

		mockGovPayService.EXPECT().
			GetRefundSummary(req, id).
			Return(paymentResource, refundSummary, Success, nil)

		mockGovPayService.EXPECT().
			CreateRefund(paymentResource, refundRequest).
			Return(response, Success, nil)

		mockDao.EXPECT().
			PatchPaymentResource(id, gomock.Any()).
			Return(fmt.Errorf("err"))

		paymentSession, refund, status, err := service.CreateRefund(req, id, body)

		So(paymentSession, ShouldBeNil)
		So(refund, ShouldBeNil)
		So(status, ShouldEqual, Error)
		So(err.Error(), ShouldEqual, "error patching payment session on database: [err]")
	})

	Convey("Return successful response", t, func() {
		body := fixtures.GetRefundRequest(8)
		refundSummary := fixtures.GetRefundSummary(8)
		paymentResource := &models.PaymentResourceRest{}
		refundRequest := fixtures.GetCreateRefundGovPayRequest(body.Amount, refundSummary.AmountAvailable)
		response := fixtures.GetCreateRefundGovPayResponse()

		payment := generatePaymentSessionGovPay()
		payment.Data.Links = models.PaymentLinksDB{Resource: "http://dummy-resource"}
		mockDao.EXPECT().GetPaymentResource(gomock.Any()).Return(&payment, nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		err := fmt.Errorf("error reading refund GovPayRequest")

		mockGovPayService.EXPECT().
			GetRefundSummary(req, id).
			Return(paymentResource, refundSummary, Success, nil)

		mockGovPayService.EXPECT().
			CreateRefund(paymentResource, refundRequest).
			Return(response, Success, nil)

		mockDao.EXPECT().
			PatchPaymentResource(id, gomock.Any()).
			Return(nil)

		service.Config.GovPaySandbox = true
		paymentSession, refund, status, err := service.CreateRefund(req, id, body)

		So(paymentSession, ShouldNotBeNil)
		So(refund, ShouldNotBeNil)
		So(status, ShouldEqual, Success)
		So(err, ShouldBeNil)
	})
}

func TestUnitUpdateRefund(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()

	mockDao := dao.NewMockDAO(mockCtrl)
	mockGovPayService := NewMockPaymentProviderService(mockCtrl)
	mockPaymentService := createMockPaymentService(mockDao, cfg)

	service := RefundService{
		GovPayService:  mockGovPayService,
		PaymentService: &mockPaymentService,
		DAO:            mockDao,
		Config:         *cfg,
	}

	req := httptest.NewRequest("PATCH", "/test", nil)
	paymentId := "123"
	refundId := "321"

	Convey("Error when getting payment session", t, func() {
		mockDao.EXPECT().GetPaymentResource(paymentId).Return(&models.PaymentResourceDB{}, fmt.Errorf("error"))

		refund, status, err := service.UpdateRefund(req, paymentId, refundId)

		So(refund, ShouldBeNil)
		So(status, ShouldEqual, Error)
		So(err.Error(), ShouldEqual, "error getting payment resource: [error getting payment resource from db: [error]]")
	})

	Convey("Error getting payment resource from db", t, func() {
		mockDao.EXPECT().GetPaymentResource(paymentId).Return(&models.PaymentResourceDB{}, fmt.Errorf("error"))

		refund, status, err := service.UpdateRefund(req, paymentId, refundId)

		So(status, ShouldEqual, Error)
		So(refund, ShouldBeNil)
		So(err.Error(), ShouldEqual, "error getting payment resource: [error getting payment resource from db: [error]]")
	})

	Convey("Payment resource not found in db", t, func() {
		mockDao.EXPECT().GetPaymentResource(paymentId).Return(nil, nil)

		refund, status, err := service.UpdateRefund(req, paymentId, refundId)

		So(status, ShouldEqual, NotFound)
		So(refund, ShouldBeNil)
		So(err.Error(), ShouldEqual, "error getting payment resource")
	})

	Convey("Refund id not present in payment session", t, func() {
		mockDao.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", ExternalPaymentStatusURI: "http://external_uri", Data: models.PaymentResourceDataDB{Amount: "10.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		govPayState := models.State{Status: "success", Finished: true}
		incomingGovPayResponse := models.IncomingGovPayResponse{CardBrand: "Visa", PaymentID: "1234", CreatedDate: "2016-01-21T17:15:000Z", State: govPayState, RefundSummary: models.RefundSummary{
			Status:          "available",
			AmountAvailable: 0,
			AmountSubmitted: 0,
		}}

		govPayResponse, _ := httpmock.NewJsonResponder(http.StatusOK, incomingGovPayResponse)
		httpmock.RegisterResponder("GET", "http://external_uri", govPayResponse)

		refund, status, err := service.UpdateRefund(req, paymentId, refundId)

		So(status, ShouldEqual, NotFound)
		So(refund, ShouldBeNil)
		So(err.Error(), ShouldEqual, "refund id not found in payment refunds")
	})

	Convey("Error getting response from GovPay ", t, func() {
		now := time.Now()
		mockGovPayService.EXPECT().GetRefundStatus(gomock.Any(), refundId).Return(nil, Error, fmt.Errorf("error generating request for GovPay"))
		mockDao.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", ExternalPaymentStatusURI: "http://external_uri", Refunds: []models.RefundResourceDB{
			{
				RefundId:          refundId,
				CreatedAt:         now.String(),
				Amount:            400,
				Status:            "success",
				ExternalRefundUrl: "external",
			},
		}, Data: models.PaymentResourceDataDB{Amount: "10.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		govPayState := models.State{Status: "success", Finished: true}
		incomingGovPayResponse := models.IncomingGovPayResponse{CardBrand: "Visa", PaymentID: "1234", CreatedDate: "2016-01-21T17:15:000Z", State: govPayState, RefundSummary: models.RefundSummary{
			Status:          "full",
			AmountAvailable: 0,
			AmountSubmitted: 0,
		}}

		govPayResponse, _ := httpmock.NewJsonResponder(http.StatusOK, incomingGovPayResponse)
		httpmock.RegisterResponder("GET", "http://external_uri", govPayResponse)

		refund, status, err := service.UpdateRefund(req, paymentId, refundId)

		So(status, ShouldEqual, Error)
		So(refund, ShouldBeNil)
		So(err.Error(), ShouldEqual, "error getting refund status from govpay: [error generating request for GovPay]")
	})

	Convey("Error patching payment session", t, func() {
		now := time.Now()
		mockGovPayService.EXPECT().GetRefundStatus(gomock.Any(), refundId).Return(&models.CreateRefundGovPayResponse{Status: RefundsStatusSuccess}, Success, nil)
		mockDao.EXPECT().PatchPaymentResource(paymentId, gomock.Any()).Return(fmt.Errorf("err"))
		mockDao.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", ExternalPaymentStatusURI: "http://external_uri", Refunds: []models.RefundResourceDB{
			{
				RefundId:          refundId,
				CreatedAt:         now.String(),
				Amount:            400,
				Status:            "submitted",
				ExternalRefundUrl: "external",
			},
		}, Data: models.PaymentResourceDataDB{Amount: "10.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		govPayState := models.State{Status: "success", Finished: true}
		incomingGovPayResponse := models.IncomingGovPayResponse{CardBrand: "Visa", PaymentID: "1234", CreatedDate: "2016-01-21T17:15:000Z", State: govPayState, RefundSummary: models.RefundSummary{
			Status:          "full",
			AmountAvailable: 0,
			AmountSubmitted: 0,
		}}

		govPayResponse, _ := httpmock.NewJsonResponder(http.StatusOK, incomingGovPayResponse)
		httpmock.RegisterResponder("GET", "http://external_uri", govPayResponse)

		refund, status, err := service.UpdateRefund(req, paymentId, refundId)

		So(refund, ShouldBeNil)
		So(status, ShouldEqual, Error)
		So(err.Error(), ShouldEqual, "error patching payment session to database: [err]")
	})

	Convey("Patches resource", t, func() {
		now := time.Now()
		var capturedSession *models.PaymentResourceDB
		mockGovPayService.EXPECT().GetRefundStatus(gomock.Any(), refundId).Return(&models.CreateRefundGovPayResponse{Status: RefundsStatusSuccess}, Success, nil)
		mockDao.EXPECT().PatchPaymentResource(paymentId, gomock.Any()).Do(func(paymentId string, session *models.PaymentResourceDB) {
			capturedSession = session
		})
		mockDao.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", ExternalPaymentStatusURI: "http://external_uri", Refunds: []models.RefundResourceDB{
			{
				RefundId:          refundId,
				CreatedAt:         now.String(),
				Amount:            400,
				Status:            "submitted",
				ExternalRefundUrl: "external",
			},
		}, Data: models.PaymentResourceDataDB{Amount: "10.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		govPayState := models.State{Status: "success", Finished: true}
		incomingGovPayResponse := models.IncomingGovPayResponse{CardBrand: "Visa", PaymentID: "1234", CreatedDate: "2016-01-21T17:15:000Z", State: govPayState, RefundSummary: models.RefundSummary{
			Status:          "full",
			AmountAvailable: 0,
			AmountSubmitted: 0,
		}}

		govPayResponse, _ := httpmock.NewJsonResponder(http.StatusOK, incomingGovPayResponse)
		httpmock.RegisterResponder("GET", "http://external_uri", govPayResponse)

		refund, status, err := service.UpdateRefund(req, paymentId, refundId)

		So(capturedSession.Refunds[0].Status, ShouldEqual, RefundsStatusSuccess)
		So(status, ShouldEqual, Success)
		So(refund.Status, ShouldEqual, RefundsStatusSuccess)
		So(err, ShouldBeNil)
	})
}

func TestUnitValidateBatchRefund(t *testing.T) {
	req := httptest.NewRequest("POST", "/test", nil)

	Convey("Error getting payment session GOV.UK Pay", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefundGovPay()

		mockDao.EXPECT().GetPaymentResourceByProviderID(gomock.Any()).Return(nil, fmt.Errorf("error")).AnyTimes()
		validationErrors, err := service.ValidateBatchRefund(req.Context(), batchRefund)

		So(len(validationErrors), ShouldEqual, 0)
		So(err, ShouldNotBeNil)
	})

	Convey("Error getting payment session - PayPal", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefundPayPal()

		mockDao.EXPECT().GetPaymentResourceByExternalPaymentTransactionID(gomock.Any()).Return(nil, fmt.Errorf("error")).AnyTimes()
		validationErrors, err := service.ValidateBatchRefund(req.Context(), batchRefund)

		So(len(validationErrors), ShouldEqual, 0)
		So(err, ShouldNotBeNil)
	})

	Convey("Validation errors - payment session not found - GOV.UK Pay", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefundGovPay()

		mockDao.EXPECT().GetPaymentResourceByProviderID(gomock.Any()).Return(nil, nil).AnyTimes()
		validationErrors, err := service.ValidateBatchRefund(req.Context(), batchRefund)

		So(len(validationErrors), ShouldEqual, 2)
		So(err, ShouldBeNil)
	})

	Convey("Validation errors - payment session not found - PayPal", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefundPayPal()

		mockDao.EXPECT().GetPaymentResourceByExternalPaymentTransactionID(gomock.Any()).Return(nil, nil).AnyTimes()
		validationErrors, err := service.ValidateBatchRefund(req.Context(), batchRefund)

		So(len(validationErrors), ShouldEqual, 2)
		So(err, ShouldBeNil)
	})

	Convey("Validation errors - invalid payment provider", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefund("invalid")

		mockDao.EXPECT().GetPaymentResourceByProviderID(gomock.Any()).Return(nil, nil).AnyTimes()
		validationErrors, err := service.ValidateBatchRefund(req.Context(), batchRefund)

		So(len(validationErrors), ShouldEqual, 0)
		So(err.Error(), ShouldEqual, "invalid payment provider supplied: invalid")
	})

	Convey("Validation errors - payment method is not GovPay (credit-card)", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefundGovPay()
		paymentSession := generatePaymentSessionPayPal()

		mockDao.EXPECT().GetPaymentResourceByProviderID(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		validationErrors, err := service.ValidateBatchRefund(req.Context(), batchRefund)

		So(len(validationErrors), ShouldEqual, 2)
		So(err, ShouldBeNil)
	})

	Convey("Validation errors - payment method is not PayPal", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefundPayPal()
		paymentSession := generatePaymentSessionGovPay()

		mockDao.EXPECT().GetPaymentResourceByExternalPaymentTransactionID(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		validationErrors, err := service.ValidateBatchRefund(req.Context(), batchRefund)

		So(len(validationErrors), ShouldEqual, 2)
		So(err, ShouldBeNil)
	})

	Convey("Validation errors - amount does not match - GOV.UK Pay", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefundGovPay()
		paymentSession := generatePaymentSessionGovPay()
		paymentSession.Data.Amount = "1.00"

		mockDao.EXPECT().GetPaymentResourceByProviderID(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		validationErrors, err := service.ValidateBatchRefund(req.Context(), batchRefund)

		So(len(validationErrors), ShouldEqual, 2)
		So(err, ShouldBeNil)
	})

	Convey("Validation errors - amount does not match - GOV.UK Pay", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefundPayPal()
		paymentSession := generatePaymentSessionPayPal()
		paymentSession.Data.Amount = "1.00"

		mockDao.EXPECT().GetPaymentResourceByExternalPaymentTransactionID(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		validationErrors, err := service.ValidateBatchRefund(req.Context(), batchRefund)

		So(len(validationErrors), ShouldEqual, 2)
		So(err, ShouldBeNil)
	})

	Convey("Validation errors - status is not paid - GOV.UK Pay", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefundGovPay()
		paymentSession := generatePaymentSessionGovPay()
		paymentSession.Data.Status = Pending.String()

		mockDao.EXPECT().GetPaymentResourceByProviderID(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		validationErrors, err := service.ValidateBatchRefund(req.Context(), batchRefund)

		So(len(validationErrors), ShouldEqual, 2)
		So(err, ShouldBeNil)
	})

	Convey("Validation errors - status is not paid - PayPal", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefundPayPal()
		paymentSession := generatePaymentSessionPayPal()
		paymentSession.Data.Status = Pending.String()

		mockDao.EXPECT().GetPaymentResourceByExternalPaymentTransactionID(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		validationErrors, err := service.ValidateBatchRefund(req.Context(), batchRefund)

		So(len(validationErrors), ShouldEqual, 2)
		So(err, ShouldBeNil)
	})

	Convey("Successfully validate XML refunds - GOV.UK Pay", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefundGovPay()
		paymentSession := generatePaymentSessionGovPay()
		paymentSession2 := generatePaymentSessionGovPay()
		paymentSession2.ExternalPaymentStatusID = "1212"

		mockDao.EXPECT().GetPaymentResourceByProviderID(batchRefund.RefundDetails[0].OrderCode).Return(&paymentSession, nil).AnyTimes()
		mockDao.EXPECT().GetPaymentResourceByProviderID(batchRefund.RefundDetails[1].OrderCode).Return(&paymentSession2, nil).AnyTimes()
		validationErrors, err := service.ValidateBatchRefund(req.Context(), batchRefund)

		So(len(validationErrors), ShouldEqual, 0)
		So(err, ShouldBeNil)
	})

	Convey("Successfully validate XML refunds - PayPal", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefundPayPal()
		paymentSession := generatePaymentSessionPayPal()
		paymentSession2 := generatePaymentSessionPayPal()
		paymentSession2.ExternalPaymentStatusID = "1212"

		mockDao.EXPECT().GetPaymentResourceByExternalPaymentTransactionID(batchRefund.RefundDetails[0].OrderCode).Return(&paymentSession, nil).AnyTimes()
		mockDao.EXPECT().GetPaymentResourceByExternalPaymentTransactionID(batchRefund.RefundDetails[1].OrderCode).Return(&paymentSession2, nil).AnyTimes()
		validationErrors, err := service.ValidateBatchRefund(req.Context(), batchRefund)

		So(len(validationErrors), ShouldEqual, 0)
		So(err, ShouldBeNil)
	})
}

func TestUnitUpdateBatchRefund(t *testing.T) {
	req := httptest.NewRequest("POST", "/test", nil)

	Convey("Error updating batch refund", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefundGovPay()

		mockDao.EXPECT().CreateBulkRefundByProviderID(gomock.Any()).Return(fmt.Errorf("error")).AnyTimes()
		err := service.UpdateBatchRefund(req.Context(), batchRefund, "filename", "userID")

		So(err, ShouldNotBeNil)
	})

	Convey("Invalid payment provider", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefund("invalid")

		mockDao.EXPECT().CreateBulkRefundByProviderID(gomock.Any()).Return(nil).AnyTimes()
		err := service.UpdateBatchRefund(req.Context(), batchRefund, "filename", "userID")

		So(err.Error(), ShouldEqual, "invalid payment provider: [invalid]")
	})

	Convey("Successfully update batch refund - GOV.UK Pay", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefundGovPay()

		mockDao.EXPECT().CreateBulkRefundByProviderID(gomock.Any()).Return(nil).AnyTimes()
		err := service.UpdateBatchRefund(req.Context(), batchRefund, "filename", "userID")

		So(err, ShouldBeNil)
	})

	Convey("Successfully update batch refund - PayPal", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefundPayPal()

		mockDao.EXPECT().CreateBulkRefundByExternalPaymentTransactionID(gomock.Any()).Return(nil).AnyTimes()
		err := service.UpdateBatchRefund(req.Context(), batchRefund, "filename", "userID")

		So(err, ShouldBeNil)
	})
}

func TestUnitGetRefundPendingPayments(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()

	mockDao := dao.NewMockDAO(mockCtrl)

	service := RefundService{
		DAO:    mockDao,
		Config: *cfg,
	}

	Convey("Error when getting payment session", t, func() {
		mockDao.EXPECT().GetPaymentsWithRefundStatus().Return([]models.PaymentResourceDB{}, fmt.Errorf("error"))

		pendingRefundPayments, err := service.GetPaymentsWithPendingRefundStatus()

		So(pendingRefundPayments, ShouldBeNil)
		So(err.Error(), ShouldEqual, "error getting payment resources with pending refund status: [error]")
	})

	Convey("Return successful response", t, func() {
		pendingRefunds := fixtures.GetPendingRefundPayments()

		mockDao.EXPECT().
			GetPaymentsWithRefundStatus().
			Return(pendingRefunds, nil)

		paymentResources, err := service.GetPaymentsWithPendingRefundStatus()

		So(paymentResources, ShouldNotBeNil)
		So(err, ShouldBeNil)
	})
}

func TestUnitProcessBatchRefund(t *testing.T) {
	req := httptest.NewRequest("POST", "/test", nil)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockDao := dao.NewMockDAO(mockCtrl)
	mockPayPalService := NewMockPaymentProviderService(mockCtrl)

	service := RefundService{
		PayPalService: mockPayPalService,
		DAO:           mockDao,
	}

	Convey("Error retrieving payments from DB", t, func() {
		mockDao.EXPECT().GetPaymentsWithRefundStatus().Return(nil, fmt.Errorf("error"))

		errs := service.ProcessBatchRefund(req)

		So(len(errs), ShouldEqual, 1)
		So(errs[0].Error(), ShouldEqual, "error retrieving payments with refund-pending status")
	})

	Convey("No payments with refund-pending status found in DB", t, func() {
		mockDao.EXPECT().GetPaymentsWithRefundStatus().Return([]models.PaymentResourceDB{}, nil)

		errs := service.ProcessBatchRefund(req)

		So(len(errs), ShouldEqual, 1)
		So(errs[0].Error(), ShouldEqual, "no payments with refund-pending status found")
	})

	Convey("GOV.UK Pay batch refund", t, func() {
		paymentSession := generatePaymentSessionGovPay()
		bulkRefund := models.BulkRefundDB{
			Amount: "invalid",
		}
		paymentSession.BulkRefund = append(paymentSession.BulkRefund, bulkRefund)
		pList := []models.PaymentResourceDB{paymentSession}
		mockDao.EXPECT().GetPaymentsWithRefundStatus().Return(pList, nil)

		errs := service.ProcessBatchRefund(req)
		So(len(errs), ShouldEqual, 1)
	})

	Convey("PayPal batch refund", t, func() {
		paymentSession := generatePaymentSessionPayPal()
		bulkRefund := models.BulkRefundDB{Amount: "invalid"}
		paymentSession.BulkRefund = append(paymentSession.BulkRefund, bulkRefund)
		pList := []models.PaymentResourceDB{paymentSession}
		mockDao.EXPECT().GetPaymentsWithRefundStatus().Return(pList, nil)

		mockPayPalService.EXPECT().GetCapturedPaymentDetails(gomock.Any()).Return(nil, fmt.Errorf("err"))

		errs := service.ProcessBatchRefund(req)
		So(len(errs), ShouldEqual, 1)
	})

	Convey("Invalid payment method", t, func() {
		paymentSession := generatePaymentSession("invalid")
		bulkRefund := models.BulkRefundDB{Amount: "invalid"}
		paymentSession.BulkRefund = append(paymentSession.BulkRefund, bulkRefund)
		pList := []models.PaymentResourceDB{paymentSession}
		mockDao.EXPECT().GetPaymentsWithRefundStatus().Return(pList, nil)

		errs := service.ProcessBatchRefund(req)
		So(len(errs), ShouldEqual, 1)
	})
}

func TestUnitGetPaymentRefunds(t *testing.T) {
	req := httptest.NewRequest("GET", "/payments/123/refunds", nil)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()

	mockDao := dao.NewMockDAO(mockCtrl)

	service := RefundService{
		DAO:    mockDao,
		Config: *cfg,
	}

	Convey("Error when getting payment's refunds session", t, func() {
		mockDao.EXPECT().GetPaymentRefunds("123").Return([]models.RefundResourceDB{}, fmt.Errorf("error"))

		pendingRefundPayments, err := service.GetPaymentRefunds(req, "123")

		So(pendingRefundPayments, ShouldBeNil)
		So(err.Error(), ShouldEqual, "error retrieving the payment refunds")
	})

	Convey("Error is nil witn no refunds", t, func() {
		paymentRefunds := []models.RefundResourceDB{}

		mockDao.EXPECT().GetPaymentRefunds("123").Return(paymentRefunds, nil)

		_, err := service.GetPaymentRefunds(req, "123")

		So(err.Error(), ShouldEqual, "no refunds with paymentId found")
	})

	Convey("Successful request - with payment's refunds", t, func() {
		refundData := models.RefundResourceDB{
			RefundId:          "sasaswewq23wsw",
			CreatedAt:         "2020-11-19T12:57:30.Z06Z",
			Amount:            800.0,
			Status:            "pending",
			ExternalRefundUrl: "https://pulicapi.payments.service.gov.uk",
		}
		paymentRefunds := []models.RefundResourceDB{refundData}

		mockDao.EXPECT().GetPaymentRefunds("123").Return(paymentRefunds, nil)

		refunds, _ := service.GetPaymentRefunds(req, "123")

		So(refunds, ShouldNotBeNil)
		So(len(refunds), ShouldEqual, 1)
	})
}

func TestUnitProcessPendingRefunds(t *testing.T) {
	cfg, _ := config.Get()
	req := httptest.NewRequest("POST", "/test", nil)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockDao := dao.NewMockDAO(mockCtrl)

	mockGovPayService := NewMockPaymentProviderService(mockCtrl)
	mockPaymentService := createMockPaymentService(mockDao, cfg)

	service := RefundService{
		GovPayService:  mockGovPayService,
		PaymentService: &mockPaymentService,
		DAO:            mockDao,
		Config:         *cfg,
	}

	Convey("Error retrieving payments with paid status from DB", t, func() {
		mockDao.EXPECT().GetPaymentsWithRefundPendingStatus().Return(nil, fmt.Errorf("error"))

		_, Error, errs := service.ProcessPendingRefunds(req)

		So(len(errs), ShouldEqual, 1)
		So(Error, ShouldNotEqual, 0)
		So(errs[0].Error(), ShouldEqual, "error retrieving payments with refund pending status")
	})

	Convey("No payments with paid status found in DB", t, func() {
		mockDao.EXPECT().GetPaymentsWithRefundPendingStatus().Return([]models.PaymentResourceDB{}, nil)
		mockDao.EXPECT().IncrementRefundAttempts(gomock.Any(), gomock.Any()).Return(nil)

		_, Success, errs := service.ProcessPendingRefunds(req)

		So(len(errs), ShouldEqual, 1)
		So(Success, ShouldNotEqual, 0)
		So(errs[0].Error(), ShouldEqual, "no payments with refund pending status found")
	})

	Convey("Payments with paid status found", t, func() {
		paymentResourceDataDB := models.PaymentResourceDataDB{}
		refundData := models.RefundResourceDB{
			RefundId:          "sasaswewq23wsw",
			CreatedAt:         "2020-11-19T12:57:30.Z06Z",
			Amount:            800.0,
			Status:            "pending",
			ExternalRefundUrl: "https://pulicapi.payments.service.gov.uk",
		}
		refundDatas := []models.RefundResourceDB{refundData}
		paymentsPaidData := models.PaymentResourceDB{
			ID:                           "xVzfvN3TlKWAAPp",
			RedirectURI:                  "https://www.google.com",
			State:                        "application-nonce-value",
			ExternalPaymentStatusURI:     "https://publicapi.payments/5212tt8usgl6k574f6",
			ExternalPaymentStatusID:      "tPM43f6ck5212tt8usgl6k574f6",
			ExternalPaymentTransactionID: "",
			Data:                         paymentResourceDataDB,
			Refunds:                      refundDatas,
			BulkRefund:                   nil,
		}

		paymentsPaidStatus := []models.PaymentResourceDB{}
		paymentsPaidStatus = append(paymentsPaidStatus, paymentsPaidData)

		mockDao.EXPECT().GetPaymentResource("xVzfvN3TlKWAAPp").Return(&models.PaymentResourceDB{}, nil)

		mockDao.EXPECT().GetPaymentsWithRefundPendingStatus().Return(paymentsPaidStatus, nil)

		payments, ResponseType, errs := service.ProcessPendingRefunds(req)

		So(len(errs), ShouldEqual, 0)
		So(ResponseType, ShouldEqual, Success)
		So(len(payments), ShouldEqual, 0)
		So(payments, ShouldBeNil)

	})
}

func TestUnitProcessGovPayBatchRefund(t *testing.T) {
	Convey("Set up unit tests", t, func() {
		req := httptest.NewRequest("POST", "/test", nil)
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		cfg, _ := config.Get()

		mockDao := dao.NewMockDAO(mockCtrl)
		mockGovPayService := NewMockPaymentProviderService(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDao, cfg)

		service := RefundService{
			GovPayService:  mockGovPayService,
			PaymentService: &mockPaymentService,
			DAO:            mockDao,
			Config:         *cfg,
		}

		Convey("Error converting amount string to integer", func() {
			paymentSession := generatePaymentSessionGovPay()
			bulkRefund := models.BulkRefundDB{
				Status:           "refund-pending",
				UploadedFilename: "name",
				UploadedAt:       "time",
				UploadedBy:       "Name",
			}
			paymentSession.BulkRefund = append(paymentSession.BulkRefund, bulkRefund)
			pList := []models.PaymentResourceDB{paymentSession}
			mockDao.EXPECT().GetPaymentsWithRefundStatus().Return(pList, nil)

			res := service.ProcessBatchRefund(req)

			So(len(res), ShouldEqual, 1)
			So(res[0].Error(), ShouldContainSubstring, "error converting amount string to int")
		})

		Convey("Multiple errors converting amount string to integer", func() {
			paymentSession := generatePaymentSessionGovPay()
			bulkRefund := models.BulkRefundDB{
				Status:           "refund-pending",
				UploadedFilename: "name",
				UploadedAt:       "time",
				UploadedBy:       "Name",
			}
			paymentSession.BulkRefund = append(paymentSession.BulkRefund, bulkRefund)
			paymentSession1 := generatePaymentSessionGovPay()
			paymentSession1.BulkRefund = append(paymentSession1.BulkRefund, bulkRefund)
			pList := []models.PaymentResourceDB{paymentSession, paymentSession1}
			mockDao.EXPECT().GetPaymentsWithRefundStatus().Return(pList, nil)

			res := service.ProcessBatchRefund(req)

			So(len(res), ShouldEqual, 2)
			So(res[0].Error(), ShouldContainSubstring, "error converting amount string to int")
			So(res[1].Error(), ShouldContainSubstring, "error converting amount string to int")
		})

		Convey("Error retrieved from calling GetRefundSummary", func() {
			paymentSession := generatePaymentSessionGovPay()
			bulkRefund := models.BulkRefundDB{
				Status:           "refund-pending",
				UploadedFilename: "name",
				UploadedAt:       "time",
				UploadedBy:       "Name",
				Amount:           "10.00",
			}
			paymentSession.BulkRefund = append(paymentSession.BulkRefund, bulkRefund)
			pList := []models.PaymentResourceDB{paymentSession}
			mockDao.EXPECT().GetPaymentsWithRefundStatus().Return(pList, nil)
			mockGovPayService.EXPECT().GetRefundSummary(req, gomock.Any()).Return(nil, nil, Error, fmt.Errorf("error"))

			res := service.ProcessBatchRefund(req)

			So(len(res), ShouldEqual, 1)
			So(res[0].Error(), ShouldContainSubstring, "error getting refund summary from govpay")
		})

		Convey("Amount available in refund summary is not equal to amount in database", func() {
			refundSummary := fixtures.GetRefundSummary(2)
			paymentResource := &models.PaymentResourceRest{}
			paymentSession := generatePaymentSessionGovPay()
			bulkRefund := models.BulkRefundDB{
				Status:           "refund-pending",
				UploadedFilename: "name",
				UploadedAt:       "time",
				UploadedBy:       "Name",
				Amount:           "10.00",
			}
			paymentSession.BulkRefund = append(paymentSession.BulkRefund, bulkRefund)
			pList := []models.PaymentResourceDB{paymentSession}
			mockDao.EXPECT().GetPaymentsWithRefundStatus().Return(pList, nil)
			mockGovPayService.EXPECT().GetRefundSummary(req, gomock.Any()).Return(paymentResource, refundSummary, Success, nil)

			res := service.ProcessBatchRefund(req)

			So(len(res), ShouldEqual, 1)
			So(res[0].Error(), ShouldContainSubstring, "refund amount is not equal to available amount for payment with id [1234]")
		})

		Convey("Error retrieved from calling GovPay service CreateRefund", func() {
			body := fixtures.GetRefundRequest(1000)
			refundSummary := fixtures.GetRefundSummary(1000)
			paymentResource := &models.PaymentResourceRest{}
			refundRequest := fixtures.GetCreateRefundGovPayRequest(body.Amount, refundSummary.AmountAvailable)
			paymentSession := generatePaymentSessionGovPay()
			bulkRefund := models.BulkRefundDB{
				Status:           "refund-pending",
				UploadedFilename: "name",
				UploadedAt:       "time",
				UploadedBy:       "Name",
				Amount:           "10.00",
			}
			paymentSession.BulkRefund = append(paymentSession.BulkRefund, bulkRefund)
			pList := []models.PaymentResourceDB{paymentSession}
			mockDao.EXPECT().GetPaymentsWithRefundStatus().Return(pList, nil)
			mockGovPayService.EXPECT().GetRefundSummary(gomock.Any(), gomock.Any()).Return(paymentResource, refundSummary, Success, nil)
			mockGovPayService.EXPECT().CreateRefund(paymentResource, refundRequest).Return(nil, Error, fmt.Errorf("error"))

			res := service.ProcessBatchRefund(req)

			So(len(res), ShouldEqual, 1)
			So(res[0].Error(), ShouldContainSubstring, "error creating refund in govpay")
		})

		Convey("Error patching payment resource in DB", func() {
			body := fixtures.GetRefundRequest(1000)
			refundSummary := fixtures.GetRefundSummary(1000)
			paymentResource := &models.PaymentResourceRest{}
			refundRequest := fixtures.GetCreateRefundGovPayRequest(body.Amount, refundSummary.AmountAvailable)
			response := fixtures.GetCreateRefundGovPayResponse()
			paymentSession := generatePaymentSessionGovPay()
			bulkRefund := models.BulkRefundDB{
				Status:           "refund-pending",
				UploadedFilename: "name",
				UploadedAt:       "time",
				UploadedBy:       "Name",
				Amount:           "10.00",
			}
			paymentSession.BulkRefund = append(paymentSession.BulkRefund, bulkRefund)
			pList := []models.PaymentResourceDB{paymentSession}
			mockDao.EXPECT().GetPaymentsWithRefundStatus().Return(pList, nil)
			mockGovPayService.EXPECT().GetRefundSummary(gomock.Any(), gomock.Any()).Return(paymentResource, refundSummary, Success, nil)
			mockGovPayService.EXPECT().CreateRefund(paymentResource, refundRequest).Return(response, Success, nil)
			mockDao.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(fmt.Errorf("error"))

			res := service.ProcessBatchRefund(req)

			So(len(res), ShouldEqual, 1)
			So(res[0].Error(), ShouldContainSubstring, "error patching payment")
		})

		Convey("Successfully create bulk refund", func() {
			body := fixtures.GetRefundRequest(1000)
			refundSummary := fixtures.GetRefundSummary(1000)
			paymentResource := &models.PaymentResourceRest{}
			refundRequest := fixtures.GetCreateRefundGovPayRequest(body.Amount, refundSummary.AmountAvailable)
			paymentSession := generatePaymentSessionGovPay()
			response := fixtures.GetCreateRefundGovPayResponse()

			bulkRefund := models.BulkRefundDB{
				Status:           "refund-pending",
				UploadedFilename: "name",
				UploadedAt:       "time",
				UploadedBy:       "Name",
				Amount:           "10.00",
			}
			paymentSession.BulkRefund = append(paymentSession.BulkRefund, bulkRefund)
			pList := []models.PaymentResourceDB{paymentSession}
			mockDao.EXPECT().GetPaymentsWithRefundStatus().Return(pList, nil)
			mockGovPayService.EXPECT().GetRefundSummary(gomock.Any(), gomock.Any()).Return(paymentResource, refundSummary, Success, nil)
			mockGovPayService.EXPECT().CreateRefund(paymentResource, refundRequest).Return(response, Success, nil)
			mockDao.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

			res := service.ProcessBatchRefund(req)

			So(len(res), ShouldEqual, 0)
		})

		Convey("Successfully create multiple bulk refunds", func() {
			body := fixtures.GetRefundRequest(1000)
			refundSummary := fixtures.GetRefundSummary(1000)
			paymentResource := &models.PaymentResourceRest{}
			refundRequest := fixtures.GetCreateRefundGovPayRequest(body.Amount, refundSummary.AmountAvailable)
			paymentSession := generatePaymentSessionGovPay()
			response := fixtures.GetCreateRefundGovPayResponse()
			paymentSession1 := generatePaymentSessionGovPay()
			bulkRefund := models.BulkRefundDB{
				Status:           "refund-pending",
				UploadedFilename: "name",
				UploadedAt:       "time",
				UploadedBy:       "Name",
				Amount:           "10.00",
			}
			paymentSession.BulkRefund = append(paymentSession.BulkRefund, bulkRefund)
			paymentSession1.BulkRefund = append(paymentSession.BulkRefund, bulkRefund)
			pList := []models.PaymentResourceDB{paymentSession, paymentSession1}
			mockDao.EXPECT().GetPaymentsWithRefundStatus().Return(pList, nil)
			mockGovPayService.EXPECT().GetRefundSummary(gomock.Any(), gomock.Any()).Return(paymentResource, refundSummary, Success, nil).Times(2)
			mockGovPayService.EXPECT().CreateRefund(paymentResource, refundRequest).Return(response, Success, nil).Times(2)
			mockDao.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil).Times(2)

			res := service.ProcessBatchRefund(req)

			So(len(res), ShouldEqual, 0)
		})
	})
}

func TestUnitProcessPayPalBatchRefund(t *testing.T) {

	req := httptest.NewRequest("POST", "/test", nil)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDao := dao.NewMockDAO(mockCtrl)
	mockPayPalService := NewMockPaymentProviderService(mockCtrl)

	refundService := RefundService{
		PayPalService: mockPayPalService,
		DAO:           mockDao,
	}

	Convey("Error getting payment details from PayPal", t, func() {
		mockPayPalService.EXPECT().GetCapturedPaymentDetails(gomock.Any()).Return(nil, fmt.Errorf("err"))
		paymentResource := models.PaymentResourceDB{
			ID:         "123",
			BulkRefund: []models.BulkRefundDB{{}},
		}
		err := refundService.processPayPalBatchRefund(req, paymentResource)
		So(err.Error(), ShouldEqual, "error getting capture details from paypal for payment ID [123]")
	})

	Convey("Capture not complete", t, func() {
		captureDetails := paypal.CaptureDetailsResponse{
			Status: "FAILED",
		}
		mockPayPalService.EXPECT().GetCapturedPaymentDetails(gomock.Any()).Return(&captureDetails, nil)

		paymentResource := models.PaymentResourceDB{
			ID:         "123",
			BulkRefund: []models.BulkRefundDB{{}},
		}

		err := refundService.processPayPalBatchRefund(req, paymentResource)
		So(err.Error(), ShouldEqual, "captured payment status [FAILED] is not complete for payment ID [123]")
	})

	Convey("Incorrect refund amount", t, func() {
		captureDetails := paypal.CaptureDetailsResponse{
			Status: "COMPLETED",
			Amount: &paypal.Money{
				Value: "10.00",
			},
		}
		mockPayPalService.EXPECT().GetCapturedPaymentDetails(gomock.Any()).Return(&captureDetails, nil)

		paymentResource := models.PaymentResourceDB{
			ID:         "123",
			BulkRefund: []models.BulkRefundDB{{}},
			Data: models.PaymentResourceDataDB{
				Amount: "9.00",
			},
		}

		err := refundService.processPayPalBatchRefund(req, paymentResource)
		So(err.Error(), ShouldEqual, "refund amount is not equal to available amount for payment ID [123]")
	})

	Convey("Error creating refund", t, func() {
		captureDetails := paypal.CaptureDetailsResponse{
			Status: "COMPLETED",
			Amount: &paypal.Money{
				Value: "10.00",
			},
		}
		mockPayPalService.EXPECT().GetCapturedPaymentDetails(gomock.Any()).Return(&captureDetails, nil)

		mockPayPalService.EXPECT().RefundCapture(gomock.Any()).Return(nil, fmt.Errorf("err"))

		paymentResource := models.PaymentResourceDB{
			ID:         "123",
			BulkRefund: []models.BulkRefundDB{{}},
			Data: models.PaymentResourceDataDB{
				Amount: "10.00",
			},
		}

		err := refundService.processPayPalBatchRefund(req, paymentResource)
		So(err.Error(), ShouldEqual, "error creating refund in PayPal for payment with id [123]")
	})

	Convey("Refund not completed", t, func() {
		captureDetails := paypal.CaptureDetailsResponse{
			Status: "COMPLETED",
			Amount: &paypal.Money{
				Value: "10.00",
			},
		}
		mockPayPalService.EXPECT().GetCapturedPaymentDetails(gomock.Any()).Return(&captureDetails, nil)

		refundResponse := paypal.RefundResponse{
			Status: "CANCELLED",
		}
		mockPayPalService.EXPECT().RefundCapture(gomock.Any()).Return(&refundResponse, nil)

		paymentResource := models.PaymentResourceDB{
			ID:         "123",
			BulkRefund: []models.BulkRefundDB{{}},
			Data: models.PaymentResourceDataDB{
				Amount: "10.00",
			},
		}

		err := refundService.processPayPalBatchRefund(req, paymentResource)
		So(err.Error(), ShouldEqual, "error completing refund in PayPal for payment with id [123]")
	})

	Convey("Error patching payment", t, func() {
		captureDetails := paypal.CaptureDetailsResponse{
			Status: "COMPLETED",
			Amount: &paypal.Money{
				Value: "10.00",
			},
		}
		mockPayPalService.EXPECT().GetCapturedPaymentDetails(gomock.Any()).Return(&captureDetails, nil)

		refundResponse := paypal.RefundResponse{
			Status: "COMPLETED",
		}
		mockPayPalService.EXPECT().RefundCapture(gomock.Any()).Return(&refundResponse, nil)

		paymentResource := models.PaymentResourceDB{
			ID:         "123",
			BulkRefund: []models.BulkRefundDB{{}},
			Data: models.PaymentResourceDataDB{
				Amount: "10.00",
			},
		}

		mockDao.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))

		err := refundService.processPayPalBatchRefund(req, paymentResource)
		So(err.Error(), ShouldEqual, "error patching payment with id [123]")
	})

	Convey("Successful refund", t, func() {
		captureDetails := paypal.CaptureDetailsResponse{
			Status: "COMPLETED",
			Amount: &paypal.Money{
				Value: "10.00",
			},
		}
		mockPayPalService.EXPECT().GetCapturedPaymentDetails(gomock.Any()).Return(&captureDetails, nil)

		refundResponse := paypal.RefundResponse{
			Status: "COMPLETED",
		}
		mockPayPalService.EXPECT().RefundCapture(gomock.Any()).Return(&refundResponse, nil)

		paymentResource := models.PaymentResourceDB{
			ID:         "123",
			BulkRefund: []models.BulkRefundDB{{}},
			Data: models.PaymentResourceDataDB{
				Amount: "10.00",
			},
		}

		mockDao.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		err := refundService.processPayPalBatchRefund(req, paymentResource)
		So(err, ShouldBeNil)
	})
}

func TestUnitCheckGovPayAndUpdateRefundStatus(t *testing.T) {
	cfg, _ := config.Get()

	req := httptest.NewRequest("POST", "/test", nil)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockDao := dao.NewMockDAO(mockCtrl)
	mockGovPayService := NewMockPaymentProviderService(mockCtrl)
	mockPaymentService := createMockPaymentService(mockDao, cfg)

	service := RefundService{
		GovPayService:  mockGovPayService,
		PaymentService: &mockPaymentService,
		DAO:            mockDao,
		Config:         *cfg,
	}

	paymentResourceDataDB := models.PaymentResourceDataDB{}
	refundData := models.RefundResourceDB{
		RefundId:          "sasaswewq23wsw",
		CreatedAt:         "2020-11-19T12:57:30.Z06Z",
		Amount:            800.0,
		Status:            "pending",
		ExternalRefundUrl: "https://pulicapi.payments.service.gov.uk",
	}
	refundDatas := []models.RefundResourceDB{refundData}
	paymentsPaidData := models.PaymentResourceDB{
		ID:                           "xVzfvN3TlKWAAPp",
		RedirectURI:                  "https://www.google.com",
		State:                        "application-nonce-value",
		ExternalPaymentStatusURI:     "https://publicapi.payments/5212tt8usgl6k574f6",
		ExternalPaymentStatusID:      "tPM43f6ck5212tt8usgl6k574f6",
		ExternalPaymentTransactionID: "",
		Data:                         paymentResourceDataDB,
		Refunds:                      refundDatas,
		BulkRefund:                   nil,
	}

	var paymentsPaidDatas []models.PaymentResourceDB

	Convey("Process pending refunds payments status with no payments", t, func() {
		var payments []models.PaymentResourceDB

		updatedPayments := service.checkGovPayAndUpdateRefundStatus(req, payments)
		So(len(updatedPayments), ShouldBeZeroValue)
	})

	Convey("Process pending refunds payments status with payments", t, func() {
		paymentsPaidDatas = append(paymentsPaidDatas, paymentsPaidData)
		mockDao.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentsPaidData, nil)
		mockDao.EXPECT().IncrementRefundAttempts(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		updatedPayments := service.checkGovPayAndUpdateRefundStatus(req, paymentsPaidDatas)
		So(len(updatedPayments), ShouldBeZeroValue)
	})

	Convey("Process pending refunds with number of calls to checkGovPayAndUpdateRefundStatus equals payments count", t, func() {
		newPaymentsPaidData := models.PaymentResourceDB{
			ID:                           "xVzfvN3TlKWAMMM",
			RedirectURI:                  "https://www.google.com",
			State:                        "application-nonce-value",
			ExternalPaymentStatusURI:     "https://publicapi.payments/5212tt8usgl6k574f6",
			ExternalPaymentStatusID:      "tPM43f6ck5212tt8usgl6k574f6",
			ExternalPaymentTransactionID: "",
			Data:                         paymentResourceDataDB,
			Refunds:                      refundDatas,
			BulkRefund:                   nil,
		}
		paymentsPaidDatas := []models.PaymentResourceDB{paymentsPaidData, newPaymentsPaidData}
		mockDao.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{}, nil).MinTimes(1)

		updatedPayments := service.checkGovPayAndUpdateRefundStatus(req, paymentsPaidDatas)
		So(len(updatedPayments), ShouldBeZeroValue)
	})

}

func setUp(controller *gomock.Controller) (RefundService, *dao.MockDAO) {
	cfg, _ := config.Get()

	mockDao := dao.NewMockDAO(controller)
	mockGovPayService := NewMockPaymentProviderService(controller)
	mockPaymentService := createMockPaymentService(mockDao, cfg)

	return RefundService{
		GovPayService:  mockGovPayService,
		PaymentService: &mockPaymentService,
		DAO:            mockDao,
		Config:         *cfg,
	}, mockDao
}

func generatePaymentSessionGovPay() models.PaymentResourceDB {
	return generatePaymentSession("credit-card")
}

func generatePaymentSessionPayPal() models.PaymentResourceDB {
	return generatePaymentSession("PayPal")
}

func generatePaymentSession(method string) models.PaymentResourceDB {
	return models.PaymentResourceDB{
		ID:                       "1234",
		RedirectURI:              "/internal/redirect",
		State:                    "state",
		ExternalPaymentStatusURI: "/external/status",
		ExternalPaymentStatusID:  "1122",
		Data: models.PaymentResourceDataDB{
			Amount:        "10.00",
			PaymentMethod: method,
			Status:        Paid.String(),
		},
		Refunds: nil,
	}
}

func generateXMLBatchRefundGovPay() models.RefundBatch {
	return generateXMLBatchRefund("govpay")
}

func generateXMLBatchRefundPayPal() models.RefundBatch {
	return generateXMLBatchRefund("paypal")
}

func generateXMLBatchRefund(provider string) models.RefundBatch {
	var refunds []models.RefundDetails

	refund := models.RefundDetails{
		XMLName:   xml.Name{"", "refund"},
		Reference: "1212",
		OrderCode: "1212",
		Amount: models.Amount{
			XMLName:      xml.Name{"", "amount"},
			Value:        "10.00",
			CurrencyCode: "GBP",
			Exponent:     "2",
		},
	}

	refund2 := models.RefundDetails{
		XMLName:   xml.Name{"", "refund"},
		Reference: "1122",
		OrderCode: "1122",
		Amount: models.Amount{
			XMLName:      xml.Name{"", "amount"},
			Value:        "10.00",
			CurrencyCode: "GBP",
			Exponent:     "2",
		},
	}

	refunds = append(refunds, refund)
	refunds = append(refunds, refund2)

	return models.RefundBatch{
		XMLName:         xml.Name{"", "batchService"},
		Version:         "1.0",
		MerchantCode:    "1234",
		BatchCode:       "1234",
		RefundDetails:   refunds,
		PaymentProvider: provider,
	}
}
