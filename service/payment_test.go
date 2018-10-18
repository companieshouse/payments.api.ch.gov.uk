package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/globalsign/mgo"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/jarcoal/httpmock.v1"
)

func createMockPaymentService(dao *dao.MockDAO) PaymentService {
	return PaymentService{
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
		mockPaymentService.CreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Invalid Request Body", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl))
		req, err := http.NewRequest("GET", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader([]byte("invalid_body")))
		w := httptest.NewRecorder()
		mockPaymentService.CreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Invalid Resource Domain", t, func() {
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		w := httptest.NewRecorder()
		getPaymentResource(w, req, "http://dummy-resource", cfg)
		So(w.Code, ShouldEqual, 400)
	})

	cfg.DomainWhitelist = "http://dummy-resource"

	Convey("Error getting cost resource", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl))
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET", "http://dummy-resource", httpmock.NewErrorResponder(fmt.Errorf("error")))
		mockPaymentService.CreatePaymentSession(w, req)
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
		getPaymentResource(w, req, "http://dummy-resource", cfg)
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
		mockPaymentService.CreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("Error Creating DB Resource", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock)
		mock.EXPECT().CreatePaymentResource(gomock.Any()).Return(fmt.Errorf("error"))

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

		mockPaymentService.CreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	cfg.PaymentServiceURL = "https://payments.companieshouse.gov.uk/payments/"
	cfg.PaymentServicePath = "/pay"

	Convey("Valid request", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock)
		mock.EXPECT().CreatePaymentResource(gomock.Any())

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

		mockPaymentService.CreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, 201)

		responseByteArray := w.Body.Bytes()
		var createdPaymentResource models.PaymentResource
		if err := json.Unmarshal(responseByteArray, &createdPaymentResource); err != nil {
			panic(err)
		}

		So(createdPaymentResource.ID, ShouldNotBeEmpty)
		So(createdPaymentResource.Links.Journey, ShouldNotBeEmpty)

		expectedJourneyURL := fmt.Sprintf("https://payments.companieshouse.gov.uk/payments/%s/pay", createdPaymentResource.ID)
		So(createdPaymentResource.Links.Journey, ShouldEqual, expectedJourneyURL)
		So(w.Header().Get("Location"), ShouldEqual, expectedJourneyURL)

		So(createdPaymentResource.CreatedBy, ShouldNotBeEmpty)
	})

	Convey("Valid generated PaymentResource ID", t, func() {
		generatedID := generateID()
		// Generated ID should be 20 characters
		So(len(generatedID), ShouldEqual, 20)
		// Generated ID should contain only numbers
		re := regexp.MustCompile("^[0-9]*$")
		So(re.MatchString(generatedID), ShouldEqual, true)
	})

}

func TestUnitGetPaymentSession(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	Convey("Payment ID missing", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl))
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		w := httptest.NewRecorder()
		mockPaymentService.GetPaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Payment ID not found", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock)
		mock.EXPECT().GetPaymentResource("invalid").Return(models.PaymentResource{}, mgo.ErrNotFound)

		req, err := http.NewRequest("Get", "", nil)
		q := req.URL.Query()
		q.Add(":payment_id", "invalid")
		req.URL.RawQuery = q.Encode()
		So(err, ShouldBeNil)
		w := httptest.NewRecorder()
		mockPaymentService.GetPaymentSession(w, req)
		So(w.Code, ShouldEqual, 403)
	})

	Convey("Error getting payment from DB", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock)
		mock.EXPECT().GetPaymentResource("1234").Return(models.PaymentResource{}, fmt.Errorf("error"))

		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()
		mockPaymentService.GetPaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("Error getting payment resource", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock)
		mock.EXPECT().GetPaymentResource("1234").Return(models.PaymentResource{}, nil)

		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()
		mockPaymentService.GetPaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)

	})

	cfg, _ := config.Get()
	cfg.DomainWhitelist = "http://dummy-resource"
	reqBody := []byte("{\"redirect_uri\": \"dummy-redirect-uri\",\"resource\": \"http://dummy-resource\",\"state\": \"dummy-state\",\"reference\": \"dummy-reference\"}")
	Convey("Amount mismatch - error updating", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(models.PaymentResource{Amount: "100", Links: models.Links{Resource: "http://dummy-resource"}}, nil)
		mock.EXPECT().UpdatePaymentAmount(gomock.Any(), gomock.Any()).Return(fmt.Errorf("error"))

		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		var paymentResource models.PaymentResource
		jsonResponse, _ := httpmock.NewJsonResponder(200, paymentResource)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		mockPaymentService.GetPaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("Get Payment session - success", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(models.PaymentResource{Amount: "", Links: models.Links{Resource: "http://dummy-resource"}}, nil)

		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		var paymentResource models.PaymentResource
		jsonResponse, _ := httpmock.NewJsonResponder(200, paymentResource)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		mockPaymentService.GetPaymentSession(w, req)
		So(w.Code, ShouldEqual, 200)
	})
}
