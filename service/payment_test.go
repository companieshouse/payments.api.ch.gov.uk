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
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/jarcoal/httpmock.v1"
)

var defaultCost = models.CostResourceRest{
	Amount:                  "10",
	AvailablePaymentMethods: []string{"method"},
	ClassOfPayment:          []string{"class"},
	Description:             "desc",
	DescriptionIdentifier:   "identifier",
	Links:                   models.CostLinksRest{Self: "self"},
}

var defaultCostArray = []models.CostResourceRest{
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

	Convey("Invalid POST request with missing required fields", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		req, err := http.NewRequest("GET", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader([]byte("{\"redirect_uri\": \"dummy-redirect-uri\"}")))
		w := httptest.NewRecorder()
		mockPaymentService.CreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Invalid Resource Domain", t, func() {
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		CostResource, httpStatus, err := getCosts("http://dummy-resource", cfg)
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
		costArray := []models.CostResourceRest{defaultCost}
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
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET", "http://dummy-resource", httpmock.NewStringResponder(502, "string"))
		CostResource, httpStatus, err := getCosts("http://dummy-resource", cfg)
		So(CostResource, ShouldEqual, nil)
		So(err, ShouldNotBeNil)
		So(httpStatus, ShouldEqual, 400)
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

		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
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
		var createdPaymentResource models.PaymentResourceDataDB
		if err := json.Unmarshal(responseByteArray, &createdPaymentResource); err != nil {
			panic(err)
		}

		So(createdPaymentResource.Links.Journey, ShouldNotBeEmpty)
		// Regex format for journey url
		regJourney := regexp.MustCompile("https://payments.companieshouse.gov.uk/payments/(.*)/pay")
		So(regJourney.MatchString(createdPaymentResource.Links.Journey), ShouldEqual, true)
		So(regJourney.MatchString(w.Header().Get("Location")), ShouldEqual, true)
		So(createdPaymentResource.Status, ShouldEqual, Pending.String())
		So(createdPaymentResource.CreatedBy, ShouldNotBeEmpty)
		// Regex format for self url
		regSelf := regexp.MustCompile("payments/(.*)")
		So(regSelf.MatchString(createdPaymentResource.Links.Self), ShouldEqual, true)

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
		costArray := []models.CostResourceRest{defaultCost, defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		mockPaymentService.CreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, 201)

		responseByteArray := w.Body.Bytes()
		var createdPaymentResource models.PaymentResourceDataDB
		if err := json.Unmarshal(responseByteArray, &createdPaymentResource); err != nil {
			panic(err)
		}

		So(createdPaymentResource.Links.Journey, ShouldNotBeEmpty)
		// Regex format for journey url
		regJourney := regexp.MustCompile("https://payments.companieshouse.gov.uk/payments/(.*)/pay")
		So(regJourney.MatchString(createdPaymentResource.Links.Journey), ShouldEqual, true)
		So(regJourney.MatchString(w.Header().Get("Location")), ShouldEqual, true)
		So(createdPaymentResource.Status, ShouldEqual, Pending.String())
		So(createdPaymentResource.CreatedBy, ShouldNotBeEmpty)
		regSelf := regexp.MustCompile("payments/(.*)")
		So(regSelf.MatchString(createdPaymentResource.Links.Self), ShouldEqual, true)
		So(createdPaymentResource.Amount, ShouldEqual, "20.00")
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
	defer resetConfig()

	Convey("Payment ID missing", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		w := httptest.NewRecorder()
		mockPaymentService.GetPaymentSessionFromRequest(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Payment ID not found", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource("invalid").Return(nil, nil)
		path := fmt.Sprintf("/payments/%s", "invalid")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "invalid"})
		So(err, ShouldBeNil)
		w := httptest.NewRecorder()
		mockPaymentService.GetPaymentSessionFromRequest(w, req)
		So(w.Code, ShouldEqual, 403)
	})

	Convey("Error getting payment from DB", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResourceDB{}, fmt.Errorf("error"))
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)
		w := httptest.NewRecorder()
		mockPaymentService.GetPaymentSessionFromRequest(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("Error getting payment resource", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResourceDB{}, nil)
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)
		w := httptest.NewRecorder()
		mockPaymentService.GetPaymentSessionFromRequest(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	cfg.DomainWhitelist = "http://dummy-resource"
	reqBody := []byte("{\"redirect_uri\": \"dummy-redirect-uri\",\"resource\": \"http://dummy-resource\",\"state\": \"dummy-state\",\"reference\": \"dummy-reference\"}")

	Convey("Invalid cost", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", Data: models.PaymentResourceDataDB{Amount: "x", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCost}
		costArray[0].Amount = "x"
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)

		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		w := httptest.NewRecorder()
		mockPaymentService.GetPaymentSessionFromRequest(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("Amount mismatch", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", Data: models.PaymentResourceDataDB{Amount: "100", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCost}
		costArray[0].Amount = "99"
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		mockPaymentService.GetPaymentSessionFromRequest(w, req)
		So(w.Code, ShouldEqual, 403)
	})

	Convey("Get Payment session - success - Single cost", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", Data: models.PaymentResourceDataDB{Amount: "10.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		mockPaymentService.GetPaymentSessionFromRequest(w, req)
		So(w.Code, ShouldEqual, 200)
	})

	Convey("Get Payment session - success - Multiple costs", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", Data: models.PaymentResourceDataDB{Amount: "20.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCost, defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		mockPaymentService.GetPaymentSessionFromRequest(w, req)
		So(w.Code, ShouldEqual, 200)
	})
}

func TestUnitPatchPaymentSession(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	reqBodyPatchInvalid := []byte("{\"amount\": \"130.00\"}")
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
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		mockPaymentService.PatchPaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Invalid Request Body", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader([]byte("invalid_body")))
		w := httptest.NewRecorder()
		mockPaymentService.PatchPaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Error getting session from DB", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		mock.EXPECT().PatchPaymentResource("1234", gomock.Any()).Return(fmt.Errorf("error"))

		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)

		req.Body = ioutil.NopCloser(bytes.NewReader(reqBodyPatch))
		req.Header.Set("Eric-Authorised-User", "test@companieshouse.gov.uk; forename=f; surname=s")
		w := httptest.NewRecorder()

		mockPaymentService.PatchPaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("No valid fields for the patch request supplied", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)

		req.Body = ioutil.NopCloser(bytes.NewReader(reqBodyPatchInvalid))
		req.Header.Set("Eric-Authorised-User", "test@companieshouse.gov.uk; forename=f; surname=s")
		w := httptest.NewRecorder()

		mockPaymentService.PatchPaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Could not find payment resource to patch", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		mock.EXPECT().PatchPaymentResource("1234", gomock.Any()).Return(fmt.Errorf("not found"))

		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)

		req.Body = ioutil.NopCloser(bytes.NewReader(reqBodyPatch))
		req.Header.Set("Eric-Authorised-User", "test@companieshouse.gov.uk; forename=f; surname=s")
		w := httptest.NewRecorder()

		mockPaymentService.PatchPaymentSession(w, req)
		So(w.Code, ShouldEqual, 403)
	})

	Convey("Valid request - Patch resource", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		mock.EXPECT().PatchPaymentResource("1234", gomock.Any()).Return(nil)

		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)

		req.Body = ioutil.NopCloser(bytes.NewReader(reqBodyPatch))
		req.Header.Set("Eric-Authorised-User", "test@companieshouse.gov.uk; forename=f; surname=s")
		w := httptest.NewRecorder()

		mockPaymentService.PatchPaymentSession(w, req)
		So(w.Code, ShouldEqual, 200)
	})
}

func TestUnitGetTotalAmount(t *testing.T) {
	Convey("Get Total Amount - valid", t, func() {
		costs := []models.CostResourceRest{{Amount: "10"}, {Amount: "13"}, {Amount: "13.01"}}
		amount, err := getTotalAmount(&costs)
		So(err, ShouldBeNil)
		So(amount, ShouldEqual, "36.01")
	})
	Convey("Test invalid amounts", t, func() {
		invalidAmounts := []string{"alpha", "12,", "12.", "12,00", "12.012", "a.9", "9.a"}
		for _, amount := range invalidAmounts {
			totalAmount, err := getTotalAmount(&[]models.CostResourceRest{{Amount: amount}})
			So(totalAmount, ShouldEqual, "")
			So(err.Error(), ShouldEqual, fmt.Sprintf("amount [%s] format incorrect", amount))
		}
	})
}

func TestUnitValidateResource(t *testing.T) {
	cfg, _ := config.Get()
	defer resetConfig()

	Convey("Invalid Resource Domain", t, func() {
		err := validateResource("http://dummy-resource", cfg)
		So(err.Error(), ShouldStartWith, "invalid resource domain")
	})

	cfg.DomainWhitelist = "http://dummy-resource"

	Convey("Valid Resource Domain", t, func() {
		err := validateResource("http://dummy-resource", cfg)
		So(err, ShouldBeNil)
	})
}

func TestUnitValidateCosts(t *testing.T) {
	Convey("Invalid Cost", t, func() {
		cost := []models.CostResourceRest{{
			Amount:                  "10",
			AvailablePaymentMethods: []string{"method"},
			ClassOfPayment:          []string{"class"},
			Description:             "",
			DescriptionIdentifier:   "identifier",
			Links:                   models.CostLinksRest{Self: "self"},
		}}
		So(validateCosts(&cost), ShouldNotBeNil)
	})
	Convey("Valid Cost", t, func() {
		cost := []models.CostResourceRest{{
			Amount:                  "10",
			AvailablePaymentMethods: []string{"method"},
			ClassOfPayment:          []string{"class"},
			Description:             "desc",
			DescriptionIdentifier:   "identifier",
			Links:                   models.CostLinksRest{Self: "self"},
		}}
		So(validateCosts(&cost), ShouldBeNil)
	})
	Convey("Multiple Costs", t, func() {
		cost := []models.CostResourceRest{
			{
				Amount:                  "10",
				AvailablePaymentMethods: []string{"method"},
				ClassOfPayment:          []string{"class"},
				Description:             "desc",
				DescriptionIdentifier:   "identifier",
				Links:                   models.CostLinksRest{Self: "self"},
			},
			{
				Amount:                  "20",
				AvailablePaymentMethods: []string{"method"},
				ClassOfPayment:          []string{"class"},
				Description:             "",
				DescriptionIdentifier:   "identifier",
				Links:                   models.CostLinksRest{Self: "self"},
			},
		}
		So(validateCosts(&cost), ShouldNotBeNil)
	})
}

func TestUnitValidatePaymentCreate(t *testing.T) {
	Convey("Invalid Payment Create, Redirect URL missing", t, func() {
		paymentCreate := models.IncomingPaymentResourceRequest{
			Resource:  "http://chs-dev:4000/payment-service-test-harness/payable-resource?amount=150",
			State:     "application-nonce-value",
			Reference: "customer-reference",
		}
		So(validatePaymentCreate(paymentCreate), ShouldNotBeNil)
	})
	Convey("Invalid Payment Create, Resource missing", t, func() {
		paymentCreate := models.IncomingPaymentResourceRequest{
			RedirectURI: "https://client.web.domain/payment-complete-callback",
			State:       "application-nonce-value",
			Reference:   "customer-reference",
		}
		So(validatePaymentCreate(paymentCreate), ShouldNotBeNil)
	})
	Convey("Invalid Payment Create, State missing", t, func() {
		paymentCreate := models.IncomingPaymentResourceRequest{
			RedirectURI: "https://client.web.domain/payment-complete-callback",
			Resource:    "http://chs-dev:4000/payment-service-test-harness/payable-resource?amount=150",
			Reference:   "customer-reference",
		}
		So(validatePaymentCreate(paymentCreate), ShouldNotBeNil)
	})

	Convey("Valid Payment Create, Reference present", t, func() {
		paymentCreate := models.IncomingPaymentResourceRequest{
			RedirectURI: "https://client.web.domain/payment-complete-callback",
			Resource:    "http://chs-dev:4000/payment-service-test-harness/payable-resource?amount=150",
			State:       "application-nonce-value",
			Reference:   "customer-reference",
		}
		So(validatePaymentCreate(paymentCreate), ShouldBeNil)
	})
	Convey("Valid Payment Create, Reference missing", t, func() {
		paymentCreate := models.IncomingPaymentResourceRequest{
			RedirectURI: "https://client.web.domain/payment-complete-callback",
			Resource:    "http://chs-dev:4000/payment-service-test-harness/payable-resource?amount=150",
			State:       "application-nonce-value",
		}
		So(validatePaymentCreate(paymentCreate), ShouldBeNil)
	})
}

func resetConfig() {
	cfg, _ := config.Get()
	cfg.DomainWhitelist = ""
}
