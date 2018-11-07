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

var defaultCost = models.CostResource{
	Amount:                  "10",
	AvailablePaymentMethods: []string{"method"},
	ClassOfPayment:          []string{"class"},
	Description:             "desc",
	DescriptionIdentifier:   "identifier",
	Links:                   models.Links{Self: "self"},
}

var defaultCostArray = []models.CostResource{
	defaultCost,
}

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
		w := httptest.NewRecorder()
		getCosts(w, req, "http://dummy-resource", cfg)
		So(w.Code, ShouldEqual, 400)
	})

	cfg.DomainWhitelist = "http://dummy-resource"

	Convey("Invalid cost", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResource{defaultCost}
		costArray[0].Amount = "x"
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)

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
		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET", "http://dummy-resource", httpmock.NewStringResponder(200, "string"))
		getCosts(w, req, "http://dummy-resource", cfg)
		So(w.Code, ShouldEqual, 500)
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
		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCostArray)
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
		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCostArray)
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
		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCostArray)
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
		costArray := []models.CostResource{defaultCost, defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
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
		So(createdPaymentResource.Amount, ShouldEqual, "20")
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
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResource{Amount: "x", Links: models.Links{Resource: "http://dummy-resource"}}, nil)
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResource{defaultCost}
		costArray[0].Amount = "x"
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)

		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		w := httptest.NewRecorder()
		mockPaymentService.GetPaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("Amount mismatch", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResource{Amount: "100", Links: models.Links{Resource: "http://dummy-resource"}}, nil)
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResource{defaultCost}
		costArray[0].Amount = "99"
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		mockPaymentService.GetPaymentSession(w, req)
		So(w.Code, ShouldEqual, 403)
	})

	Convey("Get Payment session - success - Single cost", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResource{Amount: "10", Links: models.Links{Resource: "http://dummy-resource"}}, nil)
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResource{defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		mockPaymentService.GetPaymentSession(w, req)
		So(w.Code, ShouldEqual, 200)
	})

	Convey("Get Payment session - success - Multiple costs", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResource{Amount: "20", Links: models.Links{Resource: "http://dummy-resource"}}, nil)
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		q := req.URL.Query()
		q.Add(":payment_id", "1234")
		req.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResource{defaultCost, defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		mockPaymentService.GetPaymentSession(w, req)
		So(w.Code, ShouldEqual, 200)
	})
}

func TestUnitGetTotalAmount(t *testing.T) {
	Convey("Get Total Amount - valid", t, func() {
		costs := []models.CostResource{{Amount: "10"}, {Amount: "13"}, {Amount: "13.01"}}
		amount, err := getTotalAmount(&costs)
		So(err, ShouldBeNil)
		So(amount, ShouldEqual, "36.01")
	})
	Convey("Test invalid amounts", t, func() {
		invalidAmounts := []string{"alpha", "12,", "12.", "12,00", "12.012", "a.9", "9.a"}
		for _, amount := range invalidAmounts {
			totalAmount, err := getTotalAmount(&[]models.CostResource{{Amount: amount}})
			So(totalAmount, ShouldEqual, "")
			So(err.Error(), ShouldEqual, fmt.Sprintf("amount [%s] format incorrect", amount))
		}
	})

}
