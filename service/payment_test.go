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
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/jarcoal/httpmock.v1"
)

func createMockPaymentService(dao *dao.MockDAO, config *config.Config) PaymentService {
	return PaymentService{
		DAO:    dao,
		Config: *config,
	}
}

func TestUnitCreatePaymentSession(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	reqBody := []byte("{\"redirect_uri\": \"dummy-redirect-uri\",\"resource\": \"http://dummy-resource\",\"state\": \"dummy-state\",\"reference\": \"dummy-reference\"}")

	Convey("Empty Request Body", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		req, err := http.NewRequest("GET", "", nil)
		So(err, ShouldBeNil)
		w := httptest.NewRecorder()
		mockPaymentService.CreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Invalid Request Body", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
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
		CostResource, err, httpStatus := getCosts("http://dummy-resource", cfg)
		So(CostResource, ShouldEqual, nil)
		So(err, ShouldNotBeNil)
		So(httpStatus, ShouldEqual, 400)
	})

	cfg.DomainWhitelist = "http://dummy-resource"

	Convey("Invalid cost", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResource{{Amount: "x"}}
		jsonResponse, _ := httpmock.NewJsonResponder(500, costArray)

		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		w := httptest.NewRecorder()
		mockPaymentService.CreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("Error getting cost resource", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
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
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET", "http://dummy-resource", httpmock.NewStringResponder(500, "string"))
		CostResource, err, httpStatus := getCosts("http://dummy-resource", cfg)
		So(CostResource, ShouldEqual, nil)
		So(err, ShouldNotBeNil)
		So(httpStatus, ShouldEqual, 500)
	})

	Convey("Invalid user header", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		req.Header.Set("Eric-Authorised-User", "test@companieshouse.gov.uk; forename=f; surname=s; invalid=invalid")
		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		var costArray []models.CostResource
		jsonResponse, _ := httpmock.NewJsonResponder(500, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		mockPaymentService.CreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("Error Creating DB Resource", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().CreatePaymentResource(gomock.Any()).Return(fmt.Errorf("error"))

		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)

		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		req.Header.Set("Eric-Authorised-User", "test@companieshouse.gov.uk; forename=f; surname=s")
		w := httptest.NewRecorder()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		var costArray []models.CostResource
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		mockPaymentService.CreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	cfg.PaymentsWebURL = "https://payments.companieshouse.gov.uk"

	Convey("Valid request - single cost", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().CreatePaymentResource(gomock.Any())

		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)

		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		req.Header.Set("Eric-Authorised-User", "test@companieshouse.gov.uk; forename=f; surname=s")
		w := httptest.NewRecorder()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		var costArray []models.CostResource
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		mockPaymentService.CreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, 201)
		responseByteArray := w.Body.Bytes()
		var createdPaymentResource models.PaymentResourceData
		if err := json.Unmarshal(responseByteArray, &createdPaymentResource); err != nil {
			panic(err)
		}
		So(createdPaymentResource.Links.Journey, ShouldNotBeEmpty)
		re := regexp.MustCompile("https://payments.companieshouse.gov.uk/payments/(.*)/pay")
		So(re.MatchString(createdPaymentResource.Links.Journey), ShouldEqual, true)
		So(re.MatchString(w.Header().Get("Location")), ShouldEqual, true)
		So(createdPaymentResource.CreatedBy, ShouldNotBeEmpty)
	})

	Convey("Valid request - multiple costs", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().CreatePaymentResource(gomock.Any())
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		req.Header.Set("Eric-Authorised-User", "test@companieshouse.gov.uk; forename=f; surname=s")
		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResource{{Amount: "10"}, {Amount: "12"}}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		mockPaymentService.CreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, 201)

		responseByteArray := w.Body.Bytes()
		var createdPaymentResource models.PaymentResourceData
		if err := json.Unmarshal(responseByteArray, &createdPaymentResource); err != nil {
			panic(err)
		}

		So(createdPaymentResource.Links.Journey, ShouldNotBeEmpty)
		re := regexp.MustCompile("https://payments.companieshouse.gov.uk/payments/(.*)/pay")
		So(re.MatchString(createdPaymentResource.Links.Journey), ShouldEqual, true)
		So(re.MatchString(w.Header().Get("Location")), ShouldEqual, true)

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

func TestUnitGetPayment(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	cfg, _ := config.Get()

	Convey("Payment ID missing", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		w := httptest.NewRecorder()
		mockPaymentService.GetPaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Payment ID not found", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource("invalid").Return(nil, nil)
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
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResource{}, fmt.Errorf("error"))
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
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResource{}, nil)
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()
		mockPaymentService.GetPaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	cfg.DomainWhitelist = "http://dummy-resource"
	reqBody := []byte("{\"redirect_uri\": \"dummy-redirect-uri\",\"resource\": \"http://dummy-resource\",\"state\": \"dummy-state\",\"reference\": \"dummy-reference\"}")

	Convey("Invalid cost", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResource{"1234", models.PaymentResourceData{Amount: "x", Links: models.Links{Resource: "http://dummy-resource"}}}, nil)
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResource{{Amount: "x"}}
		jsonResponse, _ := httpmock.NewJsonResponder(500, costArray)

		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		w := httptest.NewRecorder()
		mockPaymentService.GetPaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("Amount mismatch", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResource{"1234", models.PaymentResourceData{Amount: "100", Links: models.Links{Resource: "http://dummy-resource"}}}, nil)
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResource{{Amount: "99"}}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		mockPaymentService.GetPaymentSession(w, req)
		So(w.Code, ShouldEqual, 403)
	})

	Convey("Get Payment session - success - Single cost", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResource{"1234", models.PaymentResourceData{Amount: "10", Links: models.Links{Resource: "http://dummy-resource"}}}, nil)
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResource{{Amount: "10"}}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		mockPaymentService.GetPaymentSession(w, req)
		So(w.Code, ShouldEqual, 200)
	})

	Convey("Get Payment session - success - Multiple costs", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResource{"1234", models.PaymentResourceData{Amount: "23", Links: models.Links{Resource: "http://dummy-resource"}}}, nil)
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResource{{Amount: "10"}, {Amount: "13"}}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		mockPaymentService.GetPaymentSession(w, req)
		So(w.Code, ShouldEqual, 200)
	})
}

func TestUnitPatchPaymentSession(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	reqBodyPatch := []byte("{\"payment_method\": \"dummy-payment-method\",\"status\": \"dummy-status\"}")

	Convey("Payment ID missing", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		req, err := http.NewRequest("GET", "", nil)
		So(err, ShouldBeNil)
		w := httptest.NewRecorder()
		mockPaymentService.PatchPaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Empty Request Body", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		req, err := http.NewRequest("GET", "", nil)
		So(err, ShouldBeNil)
		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()
		mockPaymentService.PatchPaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Invalid Request Body", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		req, err := http.NewRequest("GET", "", nil)
		So(err, ShouldBeNil)
		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()
		req.Body = ioutil.NopCloser(bytes.NewReader([]byte("invalid_body")))
		w := httptest.NewRecorder()
		mockPaymentService.PatchPaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Error getting session from DB", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		mock.EXPECT().PatchPaymentResource("1234", gomock.Any()).Return(fmt.Errorf("error"))

		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)

		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()

		req.Body = ioutil.NopCloser(bytes.NewReader(reqBodyPatch))
		req.Header.Set("Eric-Authorised-User", "test@companieshouse.gov.uk; forename=f; surname=s")
		w := httptest.NewRecorder()

		mockPaymentService.PatchPaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("Valid request - Patch resource", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		mock.EXPECT().PatchPaymentResource("1234", gomock.Any()).Return(nil)

		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)

		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()

		req.Body = ioutil.NopCloser(bytes.NewReader(reqBodyPatch))
		req.Header.Set("Eric-Authorised-User", "test@companieshouse.gov.uk; forename=f; surname=s")
		w := httptest.NewRecorder()

		mockPaymentService.PatchPaymentSession(w, req)
		So(w.Code, ShouldEqual, 200)
	})
}
