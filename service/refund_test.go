package service

import (
	"fmt"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/fixtures"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/golang/mock/gomock"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitCreateRefund(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()

	mockDao := dao.NewMockDAO(mockCtrl)
	mockGovPayService := NewMockProviderService(mockCtrl)

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
