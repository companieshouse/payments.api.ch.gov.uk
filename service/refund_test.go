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

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitCreateRefund(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()

	mockDao := dao.NewMockDAO(mockCtrl)
	mockGovPayService := NewMockPaymentProviderService(mockCtrl)

	service := RefundService{
		GovPayService: mockGovPayService,
		DAO:           mockDao,
		Config:        *cfg,
	}

	req := httptest.NewRequest("POST", "/test", nil)
	id := "123"

	Convey("Error when getting GovPay refund summary", t, func() {
		body := fixtures.GetRefundRequest(8)
		err := fmt.Errorf("error getting payment resource")

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
		mockGovPayService.EXPECT().GetRefundStatus(gomock.Any(), refundId).Return(&models.GetRefundStatusGovPayResponse{Status: RefundsStatusSuccess}, Success, nil)
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
		mockGovPayService.EXPECT().GetRefundStatus(gomock.Any(), refundId).Return(&models.GetRefundStatusGovPayResponse{Status: RefundsStatusSuccess}, Success, nil)
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

func TestUnitValidateGovPayBatchRefund(t *testing.T) {
	req := httptest.NewRequest("POST", "/test", nil)

	Convey("Error getting payment session", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefund()

		mockDao.EXPECT().GetPaymentResourceByExternalPaymentStatusID(gomock.Any()).Return(nil, fmt.Errorf("error")).AnyTimes()
		validationErrors, err := service.ValidateGovPayBatchRefund(req.Context(), batchRefund)

		So(len(validationErrors), ShouldEqual, 0)
		So(err, ShouldNotBeNil)
	})

	Convey("Validation errors - payment session not found", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefund()

		mockDao.EXPECT().GetPaymentResourceByExternalPaymentStatusID(gomock.Any()).Return(nil, nil).AnyTimes()
		validationErrors, err := service.ValidateGovPayBatchRefund(req.Context(), batchRefund)

		So(len(validationErrors), ShouldEqual, 2)
		So(err, ShouldBeNil)
	})

	Convey("Validation errors - payment method is not GovPay (credit-card)", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefund()
		paymentSession := generatePaymentSession()
		paymentSession.Data.PaymentMethod = "paypal"

		mockDao.EXPECT().GetPaymentResourceByExternalPaymentStatusID(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		validationErrors, err := service.ValidateGovPayBatchRefund(req.Context(), batchRefund)

		So(len(validationErrors), ShouldEqual, 2)
		So(err, ShouldBeNil)
	})

	Convey("Validation errors - amount does not match", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefund()
		paymentSession := generatePaymentSession()
		paymentSession.Data.Amount = "1.00"

		mockDao.EXPECT().GetPaymentResourceByExternalPaymentStatusID(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		validationErrors, err := service.ValidateGovPayBatchRefund(req.Context(), batchRefund)

		So(len(validationErrors), ShouldEqual, 2)
		So(err, ShouldBeNil)
	})

	Convey("Validation errors - status is not paid", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefund()
		paymentSession := generatePaymentSession()
		paymentSession.Data.Status = Pending.String()

		mockDao.EXPECT().GetPaymentResourceByExternalPaymentStatusID(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		validationErrors, err := service.ValidateGovPayBatchRefund(req.Context(), batchRefund)

		So(len(validationErrors), ShouldEqual, 2)
		So(err, ShouldBeNil)
	})

	Convey("Successfully validate XML refunds", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefund()
		paymentSession := generatePaymentSession()
		paymentSession2 := generatePaymentSession()
		paymentSession2.ExternalPaymentStatusID = "1212"

		mockDao.EXPECT().GetPaymentResourceByExternalPaymentStatusID(batchRefund.GovPayRefunds[0].OrderCode).Return(&paymentSession, nil).AnyTimes()
		mockDao.EXPECT().GetPaymentResourceByExternalPaymentStatusID(batchRefund.GovPayRefunds[1].OrderCode).Return(&paymentSession2, nil).AnyTimes()
		validationErrors, err := service.ValidateGovPayBatchRefund(req.Context(), batchRefund)

		So(len(validationErrors), ShouldEqual, 0)
		So(err, ShouldBeNil)
	})
}

func TestUnitUpdateGovPayBatchRefund(t *testing.T) {
	req := httptest.NewRequest("POST", "/test", nil)

	Convey("Error updating GovPay batch refund", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefund()

		mockDao.EXPECT().CreateBulkRefund(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("error")).AnyTimes()
		err := service.UpdateGovPayBatchRefund(req.Context(), batchRefund, "filename", "userID")

		So(err, ShouldNotBeNil)
	})

	Convey("Successfully update GovPay batch refund", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		service, mockDao := setUp(mockCtrl)

		batchRefund := generateXMLBatchRefund()

		mockDao.EXPECT().CreateBulkRefund(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		err := service.UpdateGovPayBatchRefund(req.Context(), batchRefund, "filename", "userID")

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

func generatePaymentSession() models.PaymentResourceDB {
	return models.PaymentResourceDB{
		ID:                       "1234",
		RedirectURI:              "/internal/redirect",
		State:                    "state",
		ExternalPaymentStatusURI: "/external/status",
		ExternalPaymentStatusID:  "1122",
		Data: models.PaymentResourceDataDB{
			Amount:        "10.00",
			PaymentMethod: "credit-card",
			Status:        Paid.String(),
		},
		Refunds: nil,
	}

}

func generateXMLBatchRefund() models.GovPayRefundBatch {
	var govPayRefunds []models.GovPayRefund

	refund := models.GovPayRefund{
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

	refund2 := models.GovPayRefund{
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

	govPayRefunds = append(govPayRefunds, refund)
	govPayRefunds = append(govPayRefunds, refund2)

	return models.GovPayRefundBatch{
		XMLName:       xml.Name{"", "batchService"},
		Version:       "1.0",
		MerchantCode:  "1234",
		BatchCode:     "1234",
		GovPayRefunds: govPayRefunds,
	}
}
