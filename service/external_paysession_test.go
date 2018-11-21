package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gopkg.in/jarcoal/httpmock.v1"

	"github.com/companieshouse/payments.api.ch.gov.uk/models"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/golang/mock/gomock"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitCreateExternalPayment(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()

	Convey("Payment ID not supplied", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		req, err := http.NewRequest("GET", "", nil)
		So(err, ShouldBeNil)
		w := httptest.NewRecorder()
		mockPaymentService.CreateExternalPaymentJourney(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Error receiving Payment Session", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		mock.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResource{}, fmt.Errorf("error"))

		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)

		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()

		w := httptest.NewRecorder()
		mockPaymentService.CreateExternalPaymentJourney(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	cfg.DomainWhitelist = "http://dummy-resource"
	defer resetConfig()

	Convey("Invalid payment method", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		mock.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResource{ID: "1234", Data: models.PaymentResourceData{Amount: "10.00", Links: models.Links{Resource: "http://dummy-resource"}, PaymentMethod: "PayPal"}}, nil)

		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)

		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()

		w := httptest.NewRecorder()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResource{defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		mockPaymentService.CreateExternalPaymentJourney(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Error response from GovPay", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		mock.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResource{ID: "1234", Data: models.PaymentResourceData{Amount: "10.00", Links: models.Links{Resource: "http://dummy-resource"}, PaymentMethod: "GovPay"}}, nil)

		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)

		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()

		w := httptest.NewRecorder()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResource{defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		httpmock.RegisterResponder("POST", cfg.GovPayUrl, httpmock.NewStringResponder(400, "error"))

		mockPaymentService.CreateExternalPaymentJourney(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("No NextURL returned from GovPay", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		mock.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResource{ID: "1234", Data: models.PaymentResourceData{Amount: "10.00", Links: models.Links{Resource: "http://dummy-resource"}, PaymentMethod: "GovPay"}}, nil)

		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)

		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()

		w := httptest.NewRecorder()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResource{defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		IncomingGovPayResponse := models.IncomingGovPayResponse{}
		govPayJSONResponse, _ := httpmock.NewJsonResponder(201, IncomingGovPayResponse)

		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		httpmock.RegisterResponder("POST", cfg.GovPayUrl, govPayJSONResponse)

		mockPaymentService.CreateExternalPaymentJourney(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("Valid Request - Start session with GovPay ", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		mock.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResource{ID: "1234", Data: models.PaymentResourceData{Amount: "10.00", Links: models.Links{Resource: "http://dummy-resource"}, PaymentMethod: "GovPay"}}, nil)

		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)

		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()

		w := httptest.NewRecorder()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResource{defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		NextURL := models.NextURL{HREF: "nextURL"}
		GovPayLinks := models.GovPayLinks{NextURL: NextURL}

		IncomingGovPayResponse := models.IncomingGovPayResponse{GovPayLinks: GovPayLinks}
		govPayJSONResponse, _ := httpmock.NewJsonResponder(201, IncomingGovPayResponse)
		httpmock.RegisterResponder("POST", cfg.GovPayUrl, govPayJSONResponse)

		mockPaymentService.CreateExternalPaymentJourney(w, req)
		So(w.Code, ShouldEqual, 200)

		responseByteArray := w.Body.Bytes()
		var createdExternalJourney models.ExternalPaymentJourney
		if err := json.Unmarshal(responseByteArray, &createdExternalJourney); err != nil {
			panic(err)
		}

		So(createdExternalJourney.NextURL, ShouldEqual, "nextURL")
	})
}

func TestUnitConvertToPenceFromDecimal(t *testing.T) {
	Convey("Convert decimal payment in pounds to pence", t, func() {
		amount, err := convertToPenceFromDecimal("116.32")
		So(err, ShouldBeNil)
		So(amount, ShouldEqual, 11632)
	})
}
