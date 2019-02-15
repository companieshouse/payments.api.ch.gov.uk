package interceptors

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"gopkg.in/jarcoal/httpmock.v1"

	. "github.com/smartystreets/goconvey/convey"
)

var defaultCostRest = models.CostResourceRest{
	Amount:                  "10",
	AvailablePaymentMethods: []string{"method"},
	ClassOfPayment:          []string{"class"},
	Description:             "desc",
	DescriptionIdentifier:   "identifier",
	Links: models.CostLinksRest{Self: "self"},
}

func createMockPaymentService(dao *dao.MockDAO, config *config.Config) service.PaymentService {
	return service.PaymentService{
		DAO:    dao,
		Config: *config,
	}
}

// Function to create a PaymentAuthenticationInterceptor with mock mongo DAO and a mock payment service
func createPaymentAuthenticationInterceptorWithMockDAOAndService(controller *gomock.Controller, cfg *config.Config) PaymentAuthenticationInterceptor {
	mockDAO := dao.NewMockDAO(controller)
	mockPaymentService := createMockPaymentService(mockDAO, cfg)
	return PaymentAuthenticationInterceptor{
		Service: mockPaymentService,
	}
}

// Function to create a PaymentAuthenticationInterceptor with the supplied payment service
func createPaymentAuthenticationInterceptorWithMockService(service *service.PaymentService) PaymentAuthenticationInterceptor {
	return PaymentAuthenticationInterceptor{
		Service: *service,
	}
}

func TestUnitUserPaymentInterceptor(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	cfg.DomainWhitelist = "http://dummy-resource"

	Convey("No payment ID in request", t, func() {
		path := fmt.Sprintf("/payments/")
		req, err := http.NewRequest("GET", path, nil)
		So(err, ShouldBeNil)
		req.Header.Set("Eric-Identity", "authorised_identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "noroles")

		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockDAOAndService(mockCtrl, cfg)

		w := httptest.NewRecorder()
		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Invalid user details in context", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("GET", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "authorised_identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "noroles")
		// The details have to be in a authUserDetails struct, so pass a different struct to fail
		authUserDetails := models.PaymentResourceRest{
			Status: "test",
		}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyUserDetails, authUserDetails)

		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockDAOAndService(mockCtrl, cfg)

		w := httptest.NewRecorder()
		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 500)
	})

	Convey("No authorised identity", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("GET", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "authorised_identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "noroles")
		// Pass no ID (identity)
		authUserDetails := models.AuthUserDetails{}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyUserDetails, authUserDetails)

		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockDAOAndService(mockCtrl, cfg)

		w := httptest.NewRecorder()
		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 401)
	})

	Convey("Happy path where user is creator", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("GET", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "noroles")
		authUserDetails := models.AuthUserDetails{
			Id: "identity",
		}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyUserDetails, authUserDetails)

		mockDAO := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDAO, cfg)
		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockService(&mockPaymentService)

		mockDAO.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResourceDB{ID: "1234", Data: models.PaymentResourceDataDB{Amount: "10.00", CreatedBy: models.CreatedByDB{ID: "identity"}, Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCostRest}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 200)
	})

	Convey("Happy path where user is admin and request is GET", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("GET", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "/admin/payment-lookup")
		authUserDetails := models.AuthUserDetails{
			Id: "identity",
		}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyUserDetails, authUserDetails)

		mockDAO := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDAO, cfg)
		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockService(&mockPaymentService)

		mockDAO.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResourceDB{ID: "1234", Data: models.PaymentResourceDataDB{Amount: "10.00", CreatedBy: models.CreatedByDB{ID: "adminidentity"}, Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCostRest}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 200)
	})

	Convey("Unauthorised where user is admin and request is POST", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("POST", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "/admin/payment-lookup")
		authUserDetails := models.AuthUserDetails{
			Id: "identity",
		}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyUserDetails, authUserDetails)

		mockDAO := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDAO, cfg)
		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockService(&mockPaymentService)

		mockDAO.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResourceDB{ID: "1234", Data: models.PaymentResourceDataDB{Amount: "10.00", CreatedBy: models.CreatedByDB{ID: "adminidentity"}, Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCostRest}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 401)
	})
}
