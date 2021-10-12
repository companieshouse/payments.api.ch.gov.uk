package service

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/fixtures"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/golang/mock/gomock"
	"github.com/jarcoal/httpmock"
	. "github.com/smartystreets/goconvey/convey"
)

func CreateMockGovPayService(service *PaymentService) GovPayService {
	return GovPayService{
		PaymentService: *service,
	}
}

func TestUnitCheckProvider(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	mock := dao.NewMockDAO(mockCtrl)
	mockPaymentService := createMockPaymentService(mock, cfg)
	mockGovPayService := CreateMockGovPayService(&mockPaymentService)

	Convey("Error getting state of GovPay payment", t, func() {

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse := httpmock.NewErrorResponder(errors.New("error"))
		httpmock.RegisterResponder("GET", "external_uri", jsonResponse)
		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentResourceRest := models.PaymentResourceRest{
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "external_uri",
			},
			Costs: []models.CostResourceRest{costResource},
		}

		statusResponse, responseType, err := mockGovPayService.CheckPaymentProviderStatus(&paymentResourceRest)
		So(responseType.String(), ShouldEqual, Error.String())
		So(statusResponse, ShouldBeNil)
		So(err.Error(), ShouldEqual, "error getting state of GovPay payment: [error sending request to GovPay: [Get \"external_uri\": error]]")
	})

	Convey("Status - success", t, func() {

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		GovPayState := models.State{Status: "success", Finished: true}
		IncomingGovPayResponse := models.IncomingGovPayResponse{State: GovPayState}
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, IncomingGovPayResponse)
		httpmock.RegisterResponder("GET", "external_uri", jsonResponse)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentResourceRest := models.PaymentResourceRest{
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "external_uri",
			},
			Costs: []models.CostResourceRest{costResource},
		}

		statusResponse, responseType, err := mockGovPayService.CheckPaymentProviderStatus(&paymentResourceRest)
		So(responseType.String(), ShouldEqual, Success.String())
		So(statusResponse.Status, ShouldEqual, "paid")
		So(err, ShouldBeNil)
	})

	Convey("Status - failure", t, func() {

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		GovPayState := models.State{Status: "failure", Finished: true}
		IncomingGovPayResponse := models.IncomingGovPayResponse{State: GovPayState}
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, IncomingGovPayResponse)
		httpmock.RegisterResponder("GET", "external_uri", jsonResponse)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentResourceRest := models.PaymentResourceRest{
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "external_uri",
			},
			Costs: []models.CostResourceRest{costResource},
		}

		statusResponse, responseType, err := mockGovPayService.CheckPaymentProviderStatus(&paymentResourceRest)
		So(responseType.String(), ShouldEqual, Error.String())
		So(statusResponse.Status, ShouldEqual, "failed")
		So(err, ShouldBeNil)
	})

	Convey("Status - cancelled", t, func() {

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		GovPayState := models.State{Status: "failed", Finished: true, Code: "P0030"}
		IncomingGovPayResponse := models.IncomingGovPayResponse{State: GovPayState}
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, IncomingGovPayResponse)
		httpmock.RegisterResponder("GET", "external_uri", jsonResponse)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentResourceRest := models.PaymentResourceRest{
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "external_uri",
			},
			Costs: []models.CostResourceRest{costResource},
		}

		statusResponse, responseType, err := mockGovPayService.CheckPaymentProviderStatus(&paymentResourceRest)
		So(responseType.String(), ShouldEqual, Success.String())
		So(statusResponse.Status, ShouldEqual, "cancelled")
		So(err, ShouldBeNil)
	})
}

func TestUnitGenerateNextURLGovPay(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	cfg.GovPayURL = "http://dummy-govpay-url"
	mock := dao.NewMockDAO(mockCtrl)
	mockPaymentService := createMockPaymentService(mock, cfg)
	mockGovPayService := CreateMockGovPayService(&mockPaymentService)

	Convey("Error converting amount to pence", t, func() {

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("POST", cfg.GovPayURL, httpmock.NewErrorResponder(fmt.Errorf("error")))

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentResource := models.PaymentResourceRest{
			Amount: "250.567",
			CreatedBy: models.CreatedByRest{
				Email: "demo@demo.uk",
			},
			Costs: []models.CostResourceRest{costResource},
		}

		req := httptest.NewRequest("", "/test", nil)
		govPayResponse, responseType, err := mockGovPayService.CreatePaymentAndGenerateNextURL(req, &paymentResource)

		So(responseType.String(), ShouldEqual, Error.String())
		So(govPayResponse, ShouldEqual, "")
		So(err, ShouldNotBeNil)
	})

	Convey("Invalid class of payment", t, func() {

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		journeyURL := "nextUrl"
		NextURL := models.NextURL{HREF: journeyURL}
		Self := models.Self{HREF: "paymentStatusURL"}

		GovPayLinks := models.GovPayLinks{NextURL: NextURL, Self: Self}
		IncomingGovPayResponse := models.IncomingGovPayResponse{GovPayLinks: GovPayLinks}

		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusCreated, IncomingGovPayResponse)
		httpmock.RegisterResponder("POST", cfg.GovPayURL, jsonResponse)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"invalid"},
		}

		paymentResource := models.PaymentResourceRest{
			Amount: "250",
			Costs:  []models.CostResourceRest{costResource},
		}

		req := httptest.NewRequest("", "/test", nil)
		govPayResponse, responseType, err := mockGovPayService.CreatePaymentAndGenerateNextURL(req, &paymentResource)

		So(govPayResponse, ShouldEqual, "")
		So(responseType.String(), ShouldEqual, InvalidData.String())
		So(err.Error(), ShouldEqual, "error adding GovPay headers: [payment class [invalid] not recognised]")
	})

	Convey("Error sending request to GovPay", t, func() {

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("POST", cfg.GovPayURL, httpmock.NewErrorResponder(fmt.Errorf("error")))

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentResource := models.PaymentResourceRest{
			Amount: "250",
			Costs:  []models.CostResourceRest{costResource},
		}

		req := httptest.NewRequest("", "/test", nil)
		govPayResponse, responseType, err := mockGovPayService.CreatePaymentAndGenerateNextURL(req, &paymentResource)

		So(responseType.String(), ShouldEqual, Error.String())
		So(govPayResponse, ShouldEqual, "")
		So(err, ShouldNotBeNil)
	})

	Convey("Error reading response from GovPay", t, func() {

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(500, "string")
		httpmock.RegisterResponder("POST", cfg.GovPayURL, jsonResponse)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentResource := models.PaymentResourceRest{
			Amount: "250",
			Costs:  []models.CostResourceRest{costResource},
		}

		req := httptest.NewRequest("", "/test", nil)
		govPayResponse, responseType, err := mockGovPayService.CreatePaymentAndGenerateNextURL(req, &paymentResource)

		So(responseType.String(), ShouldEqual, Error.String())
		So(govPayResponse, ShouldEqual, "")
		So(err, ShouldNotBeNil)
	})

	Convey("Status code not 201", t, func() {

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		IncomingGovPayResponse := models.IncomingGovPayResponse{}

		jsonResponse, _ := httpmock.NewJsonResponder(500, IncomingGovPayResponse)
		httpmock.RegisterResponder("POST", cfg.GovPayURL, jsonResponse)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentResource := models.PaymentResourceRest{
			Amount: "250",
			Costs:  []models.CostResourceRest{costResource},
		}

		req := httptest.NewRequest("", "/test", nil)
		govPayResponse, responseType, err := mockGovPayService.CreatePaymentAndGenerateNextURL(req, &paymentResource)

		So(responseType.String(), ShouldEqual, Error.String())
		So(govPayResponse, ShouldEqual, "")
		So(err, ShouldNotBeNil)
	})

	Convey("Error storing ExternalPaymentStatusURI", t, func() {

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

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentResource := models.PaymentResourceRest{
			Amount: "250",
			Costs:  []models.CostResourceRest{costResource},
		}

		req := httptest.NewRequest("", "/test", nil)
		_, _, err := mockGovPayService.CreatePaymentAndGenerateNextURL(req, &paymentResource)

		So(err.Error(), ShouldEqual, "error storing GovPay external payment details for payment session: [error storing the External Payment Status Details against the payment session: [error]]")
	})

	Convey("Valid request to GovPay and returned NextURL for penalty", t, func() {

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

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentResource := models.PaymentResourceRest{
			Amount: "250",
			Costs:  []models.CostResourceRest{costResource},
		}

		req := httptest.NewRequest("", "/test", nil)
		govPayResponse, responseType, err := mockGovPayService.CreatePaymentAndGenerateNextURL(req, &paymentResource)

		So(responseType.String(), ShouldEqual, Success.String())
		So(govPayResponse, ShouldEqual, journeyURL)
		So(err, ShouldBeNil)
	})

	Convey("Valid request to GovPay and returned NextURL for orderable-item", t, func() {

		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		journeyURL := "orderable-item-nextUrl"
		NextURL := models.NextURL{HREF: journeyURL}
		Self := models.Self{HREF: "paymentStatusURL"}

		GovPayLinks := models.GovPayLinks{NextURL: NextURL, Self: Self}
		IncomingGovPayResponse := models.IncomingGovPayResponse{GovPayLinks: GovPayLinks}

		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusCreated, IncomingGovPayResponse)
		httpmock.RegisterResponder("POST", cfg.GovPayURL, jsonResponse)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"orderable-item"},
		}

		paymentResource := models.PaymentResourceRest{
			Amount: "250",
			Costs:  []models.CostResourceRest{costResource},
		}

		req := httptest.NewRequest("", "/test", nil)
		govPayResponse, responseType, err := mockGovPayService.CreatePaymentAndGenerateNextURL(req, &paymentResource)

		So(responseType.String(), ShouldEqual, Success.String())
		So(govPayResponse, ShouldEqual, journeyURL)
		So(err, ShouldBeNil)
	})

	Convey("Valid request to GovPay and returned NextURL for data maintenance", t, func() {

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

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"data-maintenance"},
		}

		paymentResource := models.PaymentResourceRest{
			Amount: "250",
			Costs:  []models.CostResourceRest{costResource},
		}

		req := httptest.NewRequest("", "/test", nil)
		govPayResponse, responseType, err := mockGovPayService.CreatePaymentAndGenerateNextURL(req, &paymentResource)

		So(responseType.String(), ShouldEqual, Success.String())
		So(govPayResponse, ShouldEqual, journeyURL)
		So(err, ShouldBeNil)
	})

	Convey("Valid request to GovPay and returned NextURL for legacy service", t, func() {

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

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"legacy"},
		}

		paymentResource := models.PaymentResourceRest{
			Amount: "250",
			Costs:  []models.CostResourceRest{costResource},
		}

		req := httptest.NewRequest("", "/test", nil)
		govPayResponse, responseType, err := mockGovPayService.CreatePaymentAndGenerateNextURL(req, &paymentResource)

		So(responseType.String(), ShouldEqual, Success.String())
		So(govPayResponse, ShouldEqual, journeyURL)
		So(err, ShouldBeNil)
	})

}

func TestUnitGetGovPayPaymentState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	mock := dao.NewMockDAO(mockCtrl)
	mockPaymentService := createMockPaymentService(mock, cfg)
	mockGovPayService := CreateMockGovPayService(&mockPaymentService)

	Convey("Error sending request to GOV.UK Pay", t, func() {

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse := httpmock.NewErrorResponder(errors.New("error"))
		httpmock.RegisterResponder("GET", "external_uri", jsonResponse)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentResourceRest := models.PaymentResourceRest{
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "external_uri",
			},
			Costs: []models.CostResourceRest{costResource},
		}
		govPayResponse, responseType, err := mockGovPayService.getGovPayPaymentState(&paymentResourceRest, cfg)
		So(responseType.String(), ShouldEqual, Error.String())
		So(govPayResponse, ShouldBeNil)
		So(err.Error(), ShouldEqual, "error sending request to GovPay: [Get \"external_uri\": error]")
	})

	Convey("Valid GET request to GovPay and return status", t, func() {

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		GovPayState := models.State{Status: "complete", Finished: true}
		IncomingGovPayResponse := models.IncomingGovPayResponse{State: GovPayState}

		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, IncomingGovPayResponse)
		httpmock.RegisterResponder("GET", "external_uri", jsonResponse)
		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentResource := models.PaymentResourceRest{
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "external_uri",
			},
			Costs: []models.CostResourceRest{costResource},
		}
		govPayResponse, responseType, err := mockGovPayService.getGovPayPaymentState(&paymentResource, cfg)

		So(responseType.String(), ShouldEqual, Success.String())
		So(govPayResponse, ShouldResemble, &GovPayState)
		So(err, ShouldBeNil)
	})

}

func TestUnitGetGovPayPaymentDetails(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	mock := dao.NewMockDAO(mockCtrl)
	mockPaymentService := createMockPaymentService(mock, cfg)
	mockGovPayService := CreateMockGovPayService(&mockPaymentService)

	Convey("Error sending request to GOV.UK Pay", t, func() {

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse := httpmock.NewErrorResponder(errors.New("error"))
		httpmock.RegisterResponder("GET", "external_uri", jsonResponse)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentResourceRest := models.PaymentResourceRest{
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "external_uri",
			},
			Costs: []models.CostResourceRest{costResource},
		}
		govPayResponse, responseType, err := mockGovPayService.GetPaymentDetails(&paymentResourceRest)
		So(responseType.String(), ShouldEqual, Error.String())
		So(govPayResponse, ShouldBeNil)
		So(err.Error(), ShouldEqual, "error sending request to GovPay: [Get \"external_uri\": error]")
	})

	Convey("Valid GET request to GovPay and return payment details", t, func() {

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		govPayPaymentDetails := models.PaymentDetails{CardType: "Visa", ExternalPaymentID: "1234", TransactionDate: "2016-01-21T17:15:000Z", PaymentStatus: "accepted"}
		govPayState := models.State{Status: "success", Finished: true}
		incomingGovPayResponse := models.IncomingGovPayResponse{CardBrand: "Visa", PaymentID: "1234", CreatedDate: "2016-01-21T17:15:000Z", State: govPayState}

		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, incomingGovPayResponse)
		httpmock.RegisterResponder("GET", "external_uri", jsonResponse)
		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentResource := models.PaymentResourceRest{
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "external_uri",
			},
			Costs: []models.CostResourceRest{costResource},
		}
		govPayResponse, responseType, err := mockGovPayService.GetPaymentDetails(&paymentResource)

		So(responseType.String(), ShouldEqual, Success.String())
		So(govPayResponse, ShouldResemble, &govPayPaymentDetails)
		So(err, ShouldBeNil)
	})

}

func TestUnitGetGovPayRefundSummary(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	mock := dao.NewMockDAO(mockCtrl)
	mockPaymentService := createMockPaymentService(mock, cfg)
	mockGovPayService := CreateMockGovPayService(&mockPaymentService)

	Convey("Error getting payment resource from db", t, func() {
		req := httptest.NewRequest("", "/test", nil)
		id := "123"

		mock.EXPECT().GetPaymentResource(id).Return(&models.PaymentResourceDB{}, fmt.Errorf("error"))

		paymentResource, refundSummary, responseType, err := mockGovPayService.GetRefundSummary(req, id)

		So(responseType.String(), ShouldEqual, Error.String())
		So(paymentResource, ShouldBeNil)
		So(refundSummary, ShouldBeNil)
		So(err.Error(), ShouldEqual, "error getting payment resource: [error getting payment resource from db: [error]]")
	})

	Convey("Payment resource not found in db", t, func() {
		req := httptest.NewRequest("", "/test", nil)
		id := "123"

		mock.EXPECT().GetPaymentResource(id).Return(nil, nil)

		paymentResource, refundSummary, responseType, err := mockGovPayService.GetRefundSummary(req, id)

		So(responseType.String(), ShouldEqual, NotFound.String())
		So(paymentResource, ShouldBeNil)
		So(refundSummary, ShouldBeNil)
		So(err.Error(), ShouldEqual, "error getting payment resource")
	})

	Convey("Error sending request to GovPay", t, func() {
		req := httptest.NewRequest("", "/test", nil)
		id := "123"

		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", ExternalPaymentStatusURI: "http://external_uri", Data: models.PaymentResourceDataDB{Amount: "10.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		govPayResponse := httpmock.NewErrorResponder(errors.New("error"))
		httpmock.RegisterResponder("GET", "http://external_uri", govPayResponse)

		paymentResource, refundSummary, responseType, err := mockGovPayService.GetRefundSummary(req, id)

		So(responseType.String(), ShouldEqual, Error.String())
		So(paymentResource, ShouldBeNil)
		So(refundSummary, ShouldBeNil)
		So(err.Error(), ShouldEqual, "error getting payment information from gov pay: [error sending request to GovPay: [Get \"http://external_uri\": error]]")
	})

	Convey("Refund failed - unavailable", t, func() {
		req := httptest.NewRequest("", "/test", nil)
		id := "123"

		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", ExternalPaymentStatusURI: "http://external_uri", Data: models.PaymentResourceDataDB{Amount: "10.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		govPayState := models.State{Status: "success", Finished: true}
		incomingGovPayResponse := models.IncomingGovPayResponse{CardBrand: "Visa", PaymentID: "1234", CreatedDate: "2016-01-21T17:15:000Z", State: govPayState, RefundSummary: models.RefundSummary{
			Status:          "unavailable",
			AmountAvailable: 0,
			AmountSubmitted: 0,
		}}

		govPayResponse, _ := httpmock.NewJsonResponder(http.StatusOK, incomingGovPayResponse)
		httpmock.RegisterResponder("GET", "http://external_uri", govPayResponse)

		paymentResource, refundSummary, responseType, err := mockGovPayService.GetRefundSummary(req, id)

		So(responseType.String(), ShouldEqual, InvalidData.String())
		So(paymentResource, ShouldBeNil)
		So(refundSummary, ShouldBeNil)
		So(err.Error(), ShouldEqual, "cannot refund the payment - check if the payment failed")
	})

	Convey("Refund failed - full", t, func() {
		req := httptest.NewRequest("", "/test", nil)
		id := "123"

		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", ExternalPaymentStatusURI: "http://external_uri", Data: models.PaymentResourceDataDB{Amount: "10.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)

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

		paymentResource, refundSummary, responseType, err := mockGovPayService.GetRefundSummary(req, id)

		So(responseType.String(), ShouldEqual, InvalidData.String())
		So(paymentResource, ShouldBeNil)
		So(refundSummary, ShouldBeNil)
		So(err.Error(), ShouldEqual, "cannot refund the payment - the full amount has already been refunded")
	})

	Convey("Refund failed - pending", t, func() {
		req := httptest.NewRequest("", "/test", nil)
		id := "123"

		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", ExternalPaymentStatusURI: "http://external_uri", Data: models.PaymentResourceDataDB{Amount: "10.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		govPayState := models.State{Status: "success", Finished: true}
		incomingGovPayResponse := models.IncomingGovPayResponse{CardBrand: "Visa", PaymentID: "1234", CreatedDate: "2016-01-21T17:15:000Z", State: govPayState, RefundSummary: models.RefundSummary{
			Status:          "pending",
			AmountAvailable: 0,
			AmountSubmitted: 0,
		}}

		govPayResponse, _ := httpmock.NewJsonResponder(http.StatusOK, incomingGovPayResponse)
		httpmock.RegisterResponder("GET", "http://external_uri", govPayResponse)

		paymentResource, refundSummary, responseType, err := mockGovPayService.GetRefundSummary(req, id)

		So(responseType.String(), ShouldEqual, InvalidData.String())
		So(paymentResource, ShouldBeNil)
		So(refundSummary, ShouldBeNil)
		So(err.Error(), ShouldEqual, "cannot refund the payment - the user has not completed the payment")
	})

	Convey("Refund information successfully returned", t, func() {
		req := httptest.NewRequest("", "/test", nil)
		id := "123"

		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", ExternalPaymentStatusURI: "http://external_uri", Data: models.PaymentResourceDataDB{Amount: "10.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		govPayState := models.State{Status: "success", Finished: true}
		incomingGovPayResponse := models.IncomingGovPayResponse{CardBrand: "Visa", PaymentID: "1234", CreatedDate: "2016-01-21T17:15:000Z", State: govPayState, RefundSummary: models.RefundSummary{
			Status:          "available",
			AmountAvailable: 0,
			AmountSubmitted: 800,
		}}

		govPayResponse, _ := httpmock.NewJsonResponder(http.StatusOK, incomingGovPayResponse)
		httpmock.RegisterResponder("GET", "http://external_uri", govPayResponse)

		paymentResource, refundSummary, responseType, err := mockGovPayService.GetRefundSummary(req, id)

		So(responseType.String(), ShouldEqual, Success.String())
		So(paymentResource, ShouldNotBeNil)
		So(refundSummary.Status, ShouldEqual, incomingGovPayResponse.RefundSummary.Status)
		So(refundSummary.AmountAvailable, ShouldEqual, incomingGovPayResponse.RefundSummary.AmountAvailable)
		So(refundSummary.AmountSubmitted, ShouldEqual, incomingGovPayResponse.RefundSummary.AmountSubmitted)
		So(err, ShouldBeNil)
	})

	Convey("Refund information has no status", t, func() {
		req := httptest.NewRequest("", "/test", nil)
		id := "123"

		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", ExternalPaymentStatusURI: "http://external_uri", Data: models.PaymentResourceDataDB{Amount: "10.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		govPayState := models.State{Status: "success", Finished: true}
		incomingGovPayResponse := models.IncomingGovPayResponse{CardBrand: "Visa", PaymentID: "1234", CreatedDate: "2016-01-21T17:15:000Z", State: govPayState, RefundSummary: models.RefundSummary{
			Status:          "",
			AmountAvailable: 0,
			AmountSubmitted: 0,
		}}

		govPayResponse, _ := httpmock.NewJsonResponder(http.StatusOK, incomingGovPayResponse)
		httpmock.RegisterResponder("GET", "http://external_uri", govPayResponse)

		paymentResource, refundSummary, responseType, err := mockGovPayService.GetRefundSummary(req, id)

		So(responseType.String(), ShouldEqual, NotFound.String())
		So(paymentResource, ShouldBeNil)
		So(refundSummary, ShouldBeNil)

		So(err.Error(), ShouldEqual, "cannot refund the payment - payment information not found")
	})
}

func TestUnitGovPayCreateRefund(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	mock := dao.NewMockDAO(mockCtrl)
	mockPaymentService := createMockPaymentService(mock, cfg)
	mockGovPayService := CreateMockGovPayService(&mockPaymentService)

	Convey("Invalid class of payment", t, func() {
		refundRequest := fixtures.GetCreateRefundGovPayRequest(800, 800)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"invalid"},
		}

		paymentResource := &models.PaymentResourceRest{
			Amount: "250",
			Costs:  []models.CostResourceRest{costResource},
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "http://external_uri",
			},
		}

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("POST", "http://external_uri/refunds", httpmock.NewErrorResponder(fmt.Errorf("error")))

		govPayRefundResponse, responseType, err := mockGovPayService.CreateRefund(paymentResource, refundRequest)

		So(responseType.String(), ShouldEqual, Error.String())
		So(paymentResource, ShouldNotBeNil)
		So(govPayRefundResponse, ShouldBeNil)
		So(err.Error(), ShouldEqual, "error adding GovPay headers: [payment class [invalid] not recognised]")
	})

	Convey("Error sending request to GovPay", t, func() {
		refundRequest := fixtures.GetCreateRefundGovPayRequest(800, 800)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentResource := &models.PaymentResourceRest{
			Amount: "250",
			Costs:  []models.CostResourceRest{costResource},
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "http://external_uri",
			},
		}

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("POST", "http://external_uri/refunds", httpmock.NewErrorResponder(fmt.Errorf("error")))

		govPayRefundResponse, responseType, err := mockGovPayService.CreateRefund(paymentResource, refundRequest)

		So(responseType.String(), ShouldEqual, Error.String())
		So(paymentResource, ShouldNotBeNil)
		So(govPayRefundResponse, ShouldBeNil)
		So(err.Error(), ShouldEqual, "error sending request to GovPay to create a refund: [Post \"http://external_uri/refunds\": error]")
	})

	Convey("Successful request to GovPay", t, func() {
		refundRequest := fixtures.GetCreateRefundGovPayRequest(800, 800)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentResource := &models.PaymentResourceRest{
			Amount: "250",
			Costs:  []models.CostResourceRest{costResource},
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "http://external_uri",
			},
		}

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		gpResponse := fixtures.GetCreateRefundGovPayResponse()
		jsonResponse, _ := httpmock.NewJsonResponder(200, gpResponse)

		httpmock.RegisterResponder("POST", "http://external_uri/refunds", jsonResponse)

		govPayRefundResponse, responseType, err := mockGovPayService.CreateRefund(paymentResource, refundRequest)

		So(responseType.String(), ShouldEqual, Success.String())
		So(govPayRefundResponse.Status, ShouldEqual, gpResponse.Status)
		So(govPayRefundResponse.Amount, ShouldEqual, gpResponse.Amount)
		So(govPayRefundResponse.RefundId, ShouldEqual, gpResponse.RefundId)
		So(err, ShouldBeNil)
	})
}

func TestUnitGetGovPayRefundStatus(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	mock := dao.NewMockDAO(mockCtrl)
	mockPaymentService := createMockPaymentService(mock, cfg)
	mockGovPayService := CreateMockGovPayService(&mockPaymentService)

	Convey("Error adding GovPay headers", t, func() {
		refundId := "321"
		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"invalid"},
		}

		paymentResource := &models.PaymentResourceRest{
			Amount: "250",
			Costs:  []models.CostResourceRest{costResource},
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "http://external_uri",
			},
		}

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", "http://external_uri/refunds/321", httpmock.NewErrorResponder(fmt.Errorf("error")))

		govPayStatusResponse, responseType, err := mockGovPayService.GetRefundStatus(paymentResource, refundId)

		So(responseType.String(), ShouldEqual, Error.String())
		So(paymentResource, ShouldNotBeNil)
		So(govPayStatusResponse, ShouldBeNil)
		So(err.Error(), ShouldEqual, "error adding GovPay headers: [payment class [invalid] not recognised]")
	})

	Convey("Error sending request to GovPay", t, func() {
		refundId := "321"
		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentResource := &models.PaymentResourceRest{
			Amount: "250",
			Costs:  []models.CostResourceRest{costResource},
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "http://external_uri",
			},
		}

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", "http://external_uri/refunds/321", httpmock.NewErrorResponder(fmt.Errorf("error")))

		govPayStatusResponse, responseType, err := mockGovPayService.GetRefundStatus(paymentResource, refundId)

		So(responseType.String(), ShouldEqual, Error.String())
		So(paymentResource, ShouldNotBeNil)
		So(govPayStatusResponse, ShouldBeNil)
		So(err.Error(), ShouldEqual, "error sending request to GovPay to get status of a refund: [Get \"http://external_uri/refunds/321\": error]")
	})

	Convey("Successful request to GovPay", t, func() {
		refundID := "321"
		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentResource := &models.PaymentResourceRest{
			Amount: "250",
			Costs:  []models.CostResourceRest{costResource},
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "http://external_uri",
			},
		}

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		gpResponse := fixtures.GetRefundStatusGovPayResponse()
		jsonResponse, _ := httpmock.NewJsonResponder(200, gpResponse)

		httpmock.RegisterResponder("GET", "http://external_uri/refunds/321", jsonResponse)

		govPayStatusResponse, responseType, err := mockGovPayService.GetRefundStatus(paymentResource, refundID)

		So(responseType.String(), ShouldEqual, Success.String())
		So(govPayStatusResponse.Status, ShouldEqual, gpResponse.Status)
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

func TestUnitCallToGovPay(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()

	Convey("URL missing", t, func() {
		resource := &models.PaymentResourceRest{
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "",
			},
		}
		response, err := callGovPay(nil, resource)
		So(response, ShouldBeNil)
		So(err.Error(), ShouldEqual, "gov pay URL not defined")
	})

	Convey("Error generating URL", t, func() {
		resource := &models.PaymentResourceRest{
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "\n",
			},
		}
		response, err := callGovPay(nil, resource)
		So(response, ShouldBeNil)
		So(err.Error(), ShouldEqual, "error generating request for GovPay: [parse "+`"\n"`+": net/url: invalid control character in URL]")
	})

	Convey("Error adding GovPay headers", t, func() {
		resource := &models.PaymentResourceRest{
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "external_uri",
			},
			Costs: []models.CostResourceRest{
				{ClassOfPayment: []string{"invalid"}},
			},
		}

		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mockGovPayService := CreateMockGovPayService(&mockPaymentService)

		response, err := callGovPay(&mockGovPayService, resource)
		So(response, ShouldBeNil)
		So(err.Error(), ShouldEqual, "error adding GovPay headers: [payment class [invalid] not recognised]")
	})

	Convey("Successful call to GOV.UK Pay", t, func() {
		resource := &models.PaymentResourceRest{
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "external_uri",
			},
			Costs: []models.CostResourceRest{
				{ClassOfPayment: []string{"data-maintenance"}},
			},
		}

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		IncomingGovPayResponse := models.IncomingGovPayResponse{CardBrand: "Visa", PaymentID: "1234"}

		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, IncomingGovPayResponse)
		httpmock.RegisterResponder("GET", "external_uri", jsonResponse)

		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mockGovPayService := CreateMockGovPayService(&mockPaymentService)
		mockGovPayService.PaymentService.Config.GovPayBearerTokenLegacy = "legacy"

		response, err := callGovPay(&mockGovPayService, resource)
		So(response.PaymentID, ShouldEqual, "1234")
		So(err, ShouldBeNil)

	})
}
