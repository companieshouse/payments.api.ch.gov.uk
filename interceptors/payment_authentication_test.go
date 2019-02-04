package interceptors

import (
	"context"
	"fmt"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"gopkg.in/jarcoal/httpmock.v1"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

var defaultCost = models.CostResource{
	Amount:                  "10",
	AvailablePaymentMethods: []string{"method"},
	ClassOfPayment:          []string{"class"},
	Description:             "desc",
	DescriptionIdentifier:   "identifier",
	Links: models.Links{Self: "self"},
}

var defaultCostArray = []models.CostResource{
	defaultCost,
}

func createMockPaymentService(dao *dao.MockDAO, config *config.Config) service.PaymentService {
	return service.PaymentService{
		DAO:    dao,
		Config: *config,
	}
}

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
		req.Header.Set("Eric-Identity", "authorised_identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "noroles")
		So(err, ShouldBeNil)

		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockService(&mockPaymentService)

		w := httptest.NewRecorder()
		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Invalid user details in context", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("GET", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "authorised_identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "noroles")
		// The details have to be in a authUserDetails struct, so pass a different struct to fail
		authUserDetails := models.PaymentResourceData{
			Status: "test",
		}
		ctx := context.WithValue(req.Context(), helpers.UserDetailsKey, authUserDetails)
		So(err, ShouldBeNil)

		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockService(&mockPaymentService)

		w := httptest.NewRecorder()
		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 500)
	})

	Convey("No authorised identity", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("GET", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "authorised_identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "noroles")
		// Pass no ID (identity)
		authUserDetails := models.AuthUserDetails{}
		ctx := context.WithValue(req.Context(), helpers.UserDetailsKey, authUserDetails)
		So(err, ShouldBeNil)

		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockService(&mockPaymentService)

		w := httptest.NewRecorder()
		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 401)
	})

	Convey("Happy path where user is creator", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("GET", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "noroles")
		// Pass no ID (identity)
		authUserDetails := models.AuthUserDetails{
			Id: "identity",
		}
		ctx := context.WithValue(req.Context(), helpers.UserDetailsKey, authUserDetails)
		So(err, ShouldBeNil)

		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockService(&mockPaymentService)

		mock.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResource{ID: "1234", Data: models.PaymentResourceData{Amount: "10.00", CreatedBy: models.CreatedBy{ID:"identity"}, Links: models.Links{Resource: "http://dummy-resource"}}}, nil)

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResource{defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 200)
	})

	Convey("Happy path where user is admin and request is GET", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("GET", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "/admin/payment-lookup")
		authUserDetails := models.AuthUserDetails{
			Id: "identity",
		}
		ctx := context.WithValue(req.Context(), helpers.UserDetailsKey, authUserDetails)
		So(err, ShouldBeNil)

		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockService(&mockPaymentService)

		mock.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResource{ID: "1234", Data: models.PaymentResourceData{Amount: "10.00", CreatedBy: models.CreatedBy{ID:"adminidentity"}, Links: models.Links{Resource: "http://dummy-resource"}}}, nil)

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResource{defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 200)
	})

	Convey("Unauthorised where user is admin and request is POST", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("POST", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "/admin/payment-lookup")
		authUserDetails := models.AuthUserDetails{
			Id: "identity",
		}
		ctx := context.WithValue(req.Context(), helpers.UserDetailsKey, authUserDetails)
		So(err, ShouldBeNil)

		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockService(&mockPaymentService)

		mock.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResource{ID: "1234", Data: models.PaymentResourceData{Amount: "10.00", CreatedBy: models.CreatedBy{ID:"adminidentity"}, Links: models.Links{Resource: "http://dummy-resource"}}}, nil)

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResource{defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 401)
	})
}
