package service

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/jarcoal/httpmock.v1"
)

func createMockGovPayService(service *PaymentService) GovPayService {
	return GovPayService{
		PaymentService: *service,
	}
}

func TestUnitCheckProvider(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()

	Convey("Error getting state of GovPay payment", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mockGovPayService := createMockGovPayService(&mockPaymentService)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse := httpmock.NewErrorResponder(errors.New("error"))
		httpmock.RegisterResponder("GET", "external_uri", jsonResponse)

		paymentResourceRest := models.PaymentResourceRest{
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "external_uri",
			},
		}

		responseType, statusResponse, err := mockGovPayService.CheckProvider(&paymentResourceRest)
		So(responseType.String(), ShouldEqual, Error.String())
		So(statusResponse, ShouldBeNil)
		So(err.Error(), ShouldEqual, "error getting state of GovPay payment: [error sending request to GovPay to check payment status: [Get external_uri: error]]")
	})

	Convey("Status - success", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mockGovPayService := createMockGovPayService(&mockPaymentService)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		GovPayState := models.State{Status: "success", Finished: true}
		IncomingGovPayResponse := models.IncomingGovPayResponse{State: GovPayState}
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, IncomingGovPayResponse)
		httpmock.RegisterResponder("GET", "external_uri", jsonResponse)

		paymentResourceRest := models.PaymentResourceRest{
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "external_uri",
			},
		}

		responseType, statusResponse, err := mockGovPayService.CheckProvider(&paymentResourceRest)
		So(responseType.String(), ShouldEqual, Success.String())
		So(statusResponse.Status, ShouldEqual, "paid")
		So(err, ShouldBeNil)
	})

	Convey("Status - failure", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mockGovPayService := createMockGovPayService(&mockPaymentService)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		GovPayState := models.State{Status: "failure", Finished: true}
		IncomingGovPayResponse := models.IncomingGovPayResponse{State: GovPayState}
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, IncomingGovPayResponse)
		httpmock.RegisterResponder("GET", "external_uri", jsonResponse)

		paymentResourceRest := models.PaymentResourceRest{
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "external_uri",
			},
		}

		responseType, statusResponse, err := mockGovPayService.CheckProvider(&paymentResourceRest)
		So(responseType.String(), ShouldEqual, Error.String())
		So(statusResponse.Status, ShouldEqual, "failed")
		So(err, ShouldBeNil)
	})
}

func TestUnitGenerateNextURLGovPay(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	cfg.GovPayURL = "http://dummy-govpay-url"

	Convey("Error converting amount to pence", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mockGovPayService := createMockGovPayService(&mockPaymentService)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("POST", cfg.GovPayURL, httpmock.NewErrorResponder(fmt.Errorf("error")))

		paymentResource := models.PaymentResourceRest{Amount: "250.567"}
		req := httptest.NewRequest("", "/test", nil)
		responseType, govPayResponse, err := mockGovPayService.GenerateNextURLGovPay(req, &paymentResource)

		So(responseType.String(), ShouldEqual, Error.String())
		So(govPayResponse, ShouldEqual, "")
		So(err, ShouldNotBeNil)
	})

	Convey("Error sending request to GovPay", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mockGovPayService := createMockGovPayService(&mockPaymentService)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("POST", cfg.GovPayURL, httpmock.NewErrorResponder(fmt.Errorf("error")))

		paymentResource := models.PaymentResourceRest{Amount: "250"}
		req := httptest.NewRequest("", "/test", nil)
		responseType, govPayResponse, err := mockGovPayService.GenerateNextURLGovPay(req, &paymentResource)

		So(responseType.String(), ShouldEqual, Error.String())
		So(govPayResponse, ShouldEqual, "")
		So(err, ShouldNotBeNil)
	})

	Convey("Error reading response from GovPay", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mockGovPayService := createMockGovPayService(&mockPaymentService)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(500, "string")
		httpmock.RegisterResponder("POST", cfg.GovPayURL, jsonResponse)

		paymentResource := models.PaymentResourceRest{Amount: "250"}
		req := httptest.NewRequest("", "/test", nil)
		responseType, govPayResponse, err := mockGovPayService.GenerateNextURLGovPay(req, &paymentResource)

		So(responseType.String(), ShouldEqual, Error.String())
		So(govPayResponse, ShouldEqual, "")
		So(err, ShouldNotBeNil)
	})

	Convey("Status code not 201", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mockGovPayService := createMockGovPayService(&mockPaymentService)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		IncomingGovPayResponse := models.IncomingGovPayResponse{}

		jsonResponse, _ := httpmock.NewJsonResponder(500, IncomingGovPayResponse)
		httpmock.RegisterResponder("POST", cfg.GovPayURL, jsonResponse)

		paymentResource := models.PaymentResourceRest{Amount: "250"}
		req := httptest.NewRequest("", "/test", nil)
		responseType, govPayResponse, err := mockGovPayService.GenerateNextURLGovPay(req, &paymentResource)

		So(responseType.String(), ShouldEqual, Error.String())
		So(govPayResponse, ShouldEqual, "")
		So(err, ShouldNotBeNil)
	})

	Convey("Error storing ExternalPaymentStatusURI", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mockGovPayService := createMockGovPayService(&mockPaymentService)

		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(errors.New("error"))

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		journeyURL := "nextUrl"
		NextURL := models.NextURL{HREF: journeyURL}
		Self := models.Self{HREF: "paymentStatusURL"}

		GovPayLinks := models.GovPayLinks{NextURL: NextURL, Self: Self}
		IncomingGovPayResponse := models.IncomingGovPayResponse{GovPayLinks: GovPayLinks}

		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusCreated, IncomingGovPayResponse)
		httpmock.RegisterResponder("POST", cfg.GovPayURL, jsonResponse)

		paymentResource := models.PaymentResourceRest{Amount: "250"}
		req := httptest.NewRequest("", "/test", nil)
		_, _, err := mockGovPayService.GenerateNextURLGovPay(req, &paymentResource)

		So(err.Error(), ShouldEqual, "error storing ExternalPaymentStatusURI for payment session: [error storing ExternalPaymentStatusURI on payment session: [error]]")
	})

	Convey("Valid request to GovPay and returned NextURL", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mockGovPayService := createMockGovPayService(&mockPaymentService)

		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		journeyURL := "nextUrl"
		NextURL := models.NextURL{HREF: journeyURL}
		Self := models.Self{HREF: "paymentStatusURL"}

		GovPayLinks := models.GovPayLinks{NextURL: NextURL, Self: Self}
		IncomingGovPayResponse := models.IncomingGovPayResponse{GovPayLinks: GovPayLinks}

		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusCreated, IncomingGovPayResponse)
		httpmock.RegisterResponder("POST", cfg.GovPayURL, jsonResponse)

		paymentResource := models.PaymentResourceRest{Amount: "250"}
		req := httptest.NewRequest("", "/test", nil)
		responseType, govPayResponse, err := mockGovPayService.GenerateNextURLGovPay(req, &paymentResource)

		So(responseType.String(), ShouldEqual, Success.String())
		So(govPayResponse, ShouldEqual, journeyURL)
		So(err, ShouldBeNil)
	})

}

func TestUnitGetGovPayPaymentState(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()

	Convey("Error sending request to GOV.UK Pay", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mockGovPayService := createMockGovPayService(&mockPaymentService)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse := httpmock.NewErrorResponder(errors.New("error"))
		httpmock.RegisterResponder("GET", "external_uri", jsonResponse)

		paymentResourceRest := models.PaymentResourceRest{
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "external_uri",
			},
		}
		responseType, govPayResponse, err := mockGovPayService.getGovPayPaymentState(&paymentResourceRest, cfg)
		So(responseType.String(), ShouldEqual, Error.String())
		So(govPayResponse, ShouldBeNil)
		So(err.Error(), ShouldEqual, "error sending request to GovPay to check payment status: [Get external_uri: error]")
	})

	Convey("Valid GET request to GovPay and return status", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mockGovPayService := createMockGovPayService(&mockPaymentService)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		GovPayState := models.State{Status: "complete", Finished: true}
		IncomingGovPayResponse := models.IncomingGovPayResponse{State: GovPayState}

		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, IncomingGovPayResponse)
		httpmock.RegisterResponder("GET", "external_uri", jsonResponse)

		paymentResource := models.PaymentResourceRest{
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "external_uri",
			},
		}
		responseType, govPayResponse, err := mockGovPayService.getGovPayPaymentState(&paymentResource, cfg)

		So(responseType.String(), ShouldEqual, Success.String())
		So(govPayResponse, ShouldResemble, &GovPayState)
		So(err, ShouldBeNil)
	})

}

func TestUnitConvertToPenceFromDecimal(t *testing.T) {
	Convey("Convert decimal payment in pounds to pence", t, func() {
		amount, err := convertToPenceFromDecimal("116.32")
		So(err, ShouldBeNil)
		So(amount, ShouldEqual, 11632)
	})
}
