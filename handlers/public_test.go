package handlers

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/pat"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/jarcoal/httpmock.v1"
)

func TestUnitRegisterRoutes(t *testing.T) {
	Convey("Register routes", t, func() {
		router := pat.New()
		cfg, _ := config.Get()
		Register(router, *cfg)
		So(router.GetRoute("get-healthcheck"), ShouldNotBeNil)
		So(router.GetRoute("create-payment"), ShouldNotBeNil)
	})
}

func TestUnitGetHealthCheck(t *testing.T) {
	Convey("Get HealthCheck", t, func() {
		req, err := http.NewRequest("GET", "", nil)
		So(err, ShouldBeNil)
		w := httptest.NewRecorder()
		getHealthCheck(w, req)
		So(w.Code, ShouldEqual, 200)
	})
}

func createMockPaymentService(dao *dao.MockDAO) paymentService {
	return paymentService{
		DAO: dao,
	}
}

func TestUnitCreatePaymentSession(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	reqBody := []byte("{\"redirect_uri\": \"dummy-redirect-uri\",\"resource\": \"http://dummy-resource\",\"state\": \"dummy-state\",\"reference\": \"dummy-reference\"}")

	Convey("Empty Request Body", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl))
		req, err := http.NewRequest("GET", "", nil)
		So(err, ShouldBeNil)
		w := httptest.NewRecorder()
		mockPaymentService.createPaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Invalid Request Body", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl))
		req, err := http.NewRequest("GET", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader([]byte("invalid_body")))
		w := httptest.NewRecorder()
		mockPaymentService.createPaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Invalid Resource Domain", t, func() {
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		w := httptest.NewRecorder()
		getPaymentResource(w, req, "http://dummy-resource")
		So(w.Code, ShouldEqual, 400)
	})

	cfg.DomainWhitelist = "dummy-resource"

	Convey("Error getting cost resource", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl))
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET", "http://dummy-resource", httpmock.NewErrorResponder(fmt.Errorf("error")))
		mockPaymentService.createPaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("Error reading cost resource", t, func() {
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET", "http://dummy-resource", httpmock.NewStringResponder(500, "string"))
		getPaymentResource(w, req, "http://dummy-resource")
		So(w.Code, ShouldEqual, 500)
	})

	Convey("Invalid user header", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl))
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		req.Header.Set("Eric-Authorised-User", "test@companieshouse.gov.uk; forename=f; surname=s; invalid=invalid")
		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		var paymentResource models.IncomingPaymentResourceRequest
		jsonResponse, _ := httpmock.NewJsonResponder(500, paymentResource)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		mockPaymentService.createPaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("Error Creating DB Resource", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock)
		mock.EXPECT().CreatePaymentResourceDB(gomock.Any()).Return(fmt.Errorf("error"))

		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)

		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		req.Header.Set("Eric-Authorised-User", "test@companieshouse.gov.uk; forename=f; surname=s")
		w := httptest.NewRecorder()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		var paymentResource models.PaymentResource
		jsonResponse, _ := httpmock.NewJsonResponder(200, paymentResource)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		mockPaymentService.createPaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("Valid request", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock)
		mock.EXPECT().CreatePaymentResourceDB(gomock.Any())

		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)

		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		req.Header.Set("Eric-Authorised-User", "test@companieshouse.gov.uk; forename=f; surname=s")
		w := httptest.NewRecorder()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		var paymentResource models.PaymentResource
		jsonResponse, _ := httpmock.NewJsonResponder(200, paymentResource)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		mockPaymentService.createPaymentSession(w, req)
		So(w.Code, ShouldEqual, 200)
	})

}
