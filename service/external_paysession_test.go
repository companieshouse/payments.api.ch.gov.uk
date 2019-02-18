package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/jarcoal/httpmock.v1"
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

		mock.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResourceDB{}, fmt.Errorf("error"))

		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		mockPaymentService.CreateExternalPaymentJourney(w, req)
		So(w.Code, ShouldEqual, 500)
	})
	Convey("Payment session not in progress", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		mock.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResourceDB{ID: "1234", Data: models.PaymentResourceDataDB{Amount: "10.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}, PaymentMethod: "GovPay", Status: "paid"}}, nil)

		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		mockPaymentService.CreateExternalPaymentJourney(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	cfg.DomainWhitelist = "http://dummy-resource"
	defer resetConfig()

	Convey("Invalid payment method", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		mock.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResourceDB{ID: "1234", Data: models.PaymentResourceDataDB{Amount: "10.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}, PaymentMethod: "PayPal"}}, nil)

		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		mockPaymentService.CreateExternalPaymentJourney(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Error response from GovPay", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		mock.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResourceDB{ID: "1234", Data: models.PaymentResourceDataDB{Amount: "10.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}, PaymentMethod: "GovPay", Status: InProgress.String()}}, nil)

		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		httpmock.RegisterResponder("POST", cfg.GovPayURL, httpmock.NewStringResponder(400, "error"))

		mockPaymentService.CreateExternalPaymentJourney(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("No NextURL returned from GovPay", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		mock.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResourceDB{ID: "1234", Data: models.PaymentResourceDataDB{Amount: "10.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}, PaymentMethod: "GovPay", Status: InProgress.String()}}, nil)

		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		IncomingGovPayResponse := models.IncomingGovPayResponse{}
		govPayJSONResponse, _ := httpmock.NewJsonResponder(http.StatusCreated, IncomingGovPayResponse)

		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		httpmock.RegisterResponder("POST", cfg.GovPayURL, govPayJSONResponse)

		mockPaymentService.CreateExternalPaymentJourney(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("Valid Request - Start session with GovPay ", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		mock.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResourceDB{ID: "1234", Data: models.PaymentResourceDataDB{Amount: "10.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}, PaymentMethod: "GovPay", Status: InProgress.String()}}, nil)
		mock.EXPECT().PatchPaymentResource("1234", gomock.Any()).Return(nil)

		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		NextURL := models.NextURL{HREF: "nextURL"}
		Self := models.Self{HREF: "paymentStatusURL"}

		GovPayLinks := models.GovPayLinks{NextURL: NextURL, Self: Self}

		IncomingGovPayResponse := models.IncomingGovPayResponse{GovPayLinks: GovPayLinks}
		govPayJSONResponse, _ := httpmock.NewJsonResponder(http.StatusCreated, IncomingGovPayResponse)
		httpmock.RegisterResponder("POST", cfg.GovPayURL, govPayJSONResponse)

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
