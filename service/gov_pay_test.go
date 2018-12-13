package service

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/dao"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/jarcoal/httpmock.v1"
)

func TestUnitGovPay(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()

	Convey("Error converting amount to pay to pence", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("POST", cfg.GovPayURL, httpmock.NewErrorResponder(fmt.Errorf("error")))

		paymentResourceData := models.PaymentResourceData{Amount: "250.567"}
		govPayResponse, err := mockPaymentService.returnNextURLGovPay(&paymentResourceData, "1234", cfg)

		So(govPayResponse, ShouldEqual, "")
		So(err, ShouldNotBeNil)
	})

	Convey("Error sending request to GovPay", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("POST", cfg.GovPayURL, httpmock.NewErrorResponder(fmt.Errorf("error")))

		paymentResourceData := models.PaymentResourceData{Amount: "250"}
		govPayResponse, err := mockPaymentService.returnNextURLGovPay(&paymentResourceData, "1234", cfg)

		So(govPayResponse, ShouldEqual, "")
		So(err, ShouldNotBeNil)
	})

	Convey("Error reading response from GovPay", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(500, "string")
		httpmock.RegisterResponder("POST", cfg.GovPayURL, jsonResponse)

		paymentResourceData := models.PaymentResourceData{Amount: "250"}
		govPayResponse, err := mockPaymentService.returnNextURLGovPay(&paymentResourceData, "1234", cfg)

		So(govPayResponse, ShouldEqual, "")
		So(err, ShouldNotBeNil)
	})

	Convey("Status code not 201", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		IncomingGovPayResponse := models.IncomingGovPayResponse{}

		jsonResponse, _ := httpmock.NewJsonResponder(500, IncomingGovPayResponse)
		httpmock.RegisterResponder("POST", cfg.GovPayURL, jsonResponse)

		paymentResourceData := models.PaymentResourceData{Amount: "250"}
		govPayResponse, err := mockPaymentService.returnNextURLGovPay(&paymentResourceData, "1234", cfg)

		So(govPayResponse, ShouldEqual, "")
		So(err, ShouldNotBeNil)
	})

	Convey("Valid request to GovPay and returned NextURL", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		mock.EXPECT().PatchPaymentResource("1234", gomock.Any()).Return(nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		journeyURL := "nextUrl"
		NextURL := models.NextURL{HREF: journeyURL}
		Self := models.Self{HREF: "paymentStatusURL"}

		GovPayLinks := models.GovPayLinks{NextURL: NextURL, Self: Self}
		IncomingGovPayResponse := models.IncomingGovPayResponse{GovPayLinks: GovPayLinks}

		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusCreated, IncomingGovPayResponse)
		httpmock.RegisterResponder("POST", cfg.GovPayURL, jsonResponse)

		paymentResourceData := models.PaymentResourceData{Amount: "250"}
		govPayResponse, err := mockPaymentService.returnNextURLGovPay(&paymentResourceData, "1234", cfg)

		So(govPayResponse, ShouldEqual, journeyURL)
		So(err, ShouldBeNil)
	})

}
