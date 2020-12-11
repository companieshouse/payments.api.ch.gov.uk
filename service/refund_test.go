package service

import (
	"fmt"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/fixtures"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/golang/mock/gomock"
	"github.com/jarcoal/httpmock"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
			GetGovPayRefundSummary(req, id).
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
			GetGovPayRefundSummary(req, id).
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
			GetGovPayRefundSummary(req, id).
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

	Convey("Return successful response", t, func() {
		body := fixtures.GetRefundRequest(8)
		refundSummary := fixtures.GetRefundSummary(8)
		paymentResource := &models.PaymentResourceRest{}
		refundRequest := fixtures.GetCreateRefundGovPayRequest(body.Amount, refundSummary.AmountAvailable)
		response := fixtures.GetCreateRefundGovPayResponse()

		err := fmt.Errorf("error reading refund GovPayRequest")

		mockGovPayService.EXPECT().
			GetGovPayRefundSummary(req, id).
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
		mockGovPayService.EXPECT().GetGovPayRefundStatus(gomock.Any(), refundId).Return(nil, Error, fmt.Errorf("error generating request for GovPay"))
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

	Convey("Does not patch resource ", t, func() {
		now := time.Now()
		mockGovPayService.EXPECT().GetGovPayRefundStatus(gomock.Any(), refundId).Return(&models.GetRefundStatusGovPayResponse{Status: RefundsStatusSubmitted}, Success, nil)
		mockDao.EXPECT().PatchPaymentResource(paymentId, gomock.Any()).Times(0)
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

		So(status, ShouldEqual, Success)
		So(refund.Status, ShouldEqual, RefundsStatusSubmitted)
		So(err, ShouldBeNil)
	})

	FocusConvey("Patches resource ", t, func() {
		now := time.Now()
		var capturedSession *models.PaymentResourceDB
		mockGovPayService.EXPECT().GetGovPayRefundStatus(gomock.Any(), refundId).Return(&models.GetRefundStatusGovPayResponse{Status: RefundsStatusSuccess}, Success, nil)
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
