package interceptors

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/chs.go/authentication"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/jarcoal/httpmock"

	. "github.com/smartystreets/goconvey/convey"
)

const resourceURL = "http://dummy-resource"

func GetTestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

var defaultCostRest = models.CostResourceRest{
	Amount:                  "10",
	AvailablePaymentMethods: []string{"method"},
	ClassOfPayment:          []string{"class"},
	Description:             "desc",
	DescriptionIdentifier:   "identifier",
	ProductType:             "productType",
}

var defaultCosts = models.CostsRest{
	Description: "costs_desc",
	Costs:       []models.CostResourceRest{defaultCostRest},
}

func createMockPaymentService(mockDAO *dao.MockDAO, cfg *config.Config) service.PaymentService {
	return service.PaymentService{
		DAO:    mockDAO,
		Config: *cfg,
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
func createPaymentAuthenticationInterceptorWithMockService(paymentService *service.PaymentService) PaymentAuthenticationInterceptor {
	return PaymentAuthenticationInterceptor{
		Service: *paymentService,
	}
}

func TestUnitUserPaymentInterceptor(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	cfg.DomainAllowList = resourceURL

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
		So(w.Code, ShouldEqual, http.StatusBadRequest)
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
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockDAOAndService(mockCtrl, cfg)

		w := httptest.NewRecorder()
		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
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
		authUserDetails := authentication.AuthUserDetails{}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockDAOAndService(mockCtrl, cfg)

		w := httptest.NewRecorder()
		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("Payment not found in DB", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("GET", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "/admin/payment-lookup")
		authUserDetails := authentication.AuthUserDetails{
			ID: "identity",
		}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		mockDAO := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDAO, cfg)
		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockService(&mockPaymentService)

		mockDAO.EXPECT().GetPaymentResource("1234").Return(nil, nil)

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCostRest}
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, costArray)
		httpmock.RegisterResponder("GET", resourceURL, jsonResponse)

		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusNotFound)
	})

	Convey("Error reading from DB", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("GET", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "/admin/payment-lookup")
		authUserDetails := authentication.AuthUserDetails{
			ID: "identity",
		}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		mockDAO := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDAO, cfg)
		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockService(&mockPaymentService)

		mockDAO.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResourceDB{}, fmt.Errorf("error"))

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCostRest}
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, costArray)
		httpmock.RegisterResponder("GET", resourceURL, jsonResponse)

		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Status Forbidden", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("GET", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "/admin/payment-lookup")
		authUserDetails := authentication.AuthUserDetails{
			ID: "identity",
		}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		mockDAO := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDAO, cfg)
		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockService(&mockPaymentService)

		mockDAO.EXPECT().GetPaymentResource("1234").Return(
			&models.PaymentResourceDB{
				ID: "1234",
				Data: models.PaymentResourceDataDB{
					Amount:    "20.00",
					CreatedBy: models.CreatedByDB{ID: "identity"},
					Links:     models.PaymentLinksDB{Resource: resourceURL},
				},
			},
			nil,
		)

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder("GET", resourceURL, jsonResponse)

		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusForbidden)
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
		authUserDetails := authentication.AuthUserDetails{
			ID: "identity",
		}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		mockDAO := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDAO, cfg)
		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockService(&mockPaymentService)

		mockDAO.EXPECT().GetPaymentResource("1234").Return(
			&models.PaymentResourceDB{
				ID: "1234",
				Data: models.PaymentResourceDataDB{
					Amount:    "10.00",
					CreatedBy: models.CreatedByDB{ID: "identity"},
					Links:     models.PaymentLinksDB{Resource: resourceURL},
				},
			},
			nil,
		)

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder("GET", resourceURL, jsonResponse)

		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusOK)
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
		authUserDetails := authentication.AuthUserDetails{
			ID: "identity",
		}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		mockDAO := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDAO, cfg)
		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockService(&mockPaymentService)

		mockDAO.EXPECT().GetPaymentResource("1234").Return(
			&models.PaymentResourceDB{
				ID: "1234",
				Data: models.PaymentResourceDataDB{
					Amount:    "10.00",
					CreatedBy: models.CreatedByDB{ID: "adminidentity"},
					Links:     models.PaymentLinksDB{Resource: resourceURL},
				},
			},
			nil,
		)

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder("GET", resourceURL, jsonResponse)

		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusOK)
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
		authUserDetails := authentication.AuthUserDetails{
			ID: "identity",
		}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		mockDAO := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDAO, cfg)
		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockService(&mockPaymentService)

		mockDAO.EXPECT().GetPaymentResource("1234").Return(
			&models.PaymentResourceDB{
				ID: "1234",
				Data: models.PaymentResourceDataDB{
					Amount:    "10.00",
					CreatedBy: models.CreatedByDB{ID: "adminidentity"},
					Links:     models.PaymentLinksDB{Resource: resourceURL},
				},
			},
			nil,
		)

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder("GET", resourceURL, jsonResponse)

		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("Happy path where user has elevated privileges key accessing a non-creator resource", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("GET", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "identity")
		req.Header.Set("Eric-Identity-Type", "key")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Key-Roles", "*")
		authUserDetails := authentication.AuthUserDetails{
			ID: "api-key-user",
		}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)
		mockDAO := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDAO, cfg)
		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockService(&mockPaymentService)

		mockDAO.EXPECT().GetPaymentResource("1234").Return(
			&models.PaymentResourceDB{
				ID: "1234",
				Data: models.PaymentResourceDataDB{
					Amount:    "10.00",
					CreatedBy: models.CreatedByDB{ID: "identity"},
					Links:     models.PaymentLinksDB{Resource: resourceURL},
				},
			},
			nil,
		)

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder("GET", resourceURL, jsonResponse)

		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusOK)
	})

	Convey("Happy path where user has payment privileges key accessing a non-creator resource", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("GET", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "identity")
		req.Header.Set("Eric-Identity-Type", "key")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Key-Privileges", "payment")
		authUserDetails := authentication.AuthUserDetails{
			ID: "api-key-user",
		}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)
		mockDAO := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDAO, cfg)
		paymentAuthenticationInterceptor := createPaymentAuthenticationInterceptorWithMockService(&mockPaymentService)

		mockDAO.EXPECT().GetPaymentResource("1234").Return(
			&models.PaymentResourceDB{
				ID: "1234",
				Data: models.PaymentResourceDataDB{
					Amount:    "10.00",
					CreatedBy: models.CreatedByDB{ID: "identity"},
					Links:     models.PaymentLinksDB{Resource: resourceURL},
				},
			},
			nil,
		)

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder("GET", resourceURL, jsonResponse)

		test := paymentAuthenticationInterceptor.PaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusOK)
	})
}

func TestUnitAdminUserPaymentInterceptor(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	cfg.DomainAllowList = resourceURL

	Convey("No oauth2 or API key identity type", t, func() {
		path := "/admin/payments/bulk-refunds"
		req, err := http.NewRequest("POST", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{})
		req.Header.Set("Eric-Identity", "authorised_identity")
		req.Header.Set("Eric-Identity-Type", "invalid")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "noroles")
		// Pass no ID (identity)
		authUserDetails := authentication.AuthUserDetails{}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		w := httptest.NewRecorder()
		test := PaymentAdminAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("Invalid user details in context", t, func() {
		path := "/admin/payments/bulk-refunds"
		req, err := http.NewRequest("POST", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{})
		req.Header.Set("Eric-Identity", "authorised_identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "noroles")
		// The details have to be in a authUserDetails struct, so pass a different struct to fail
		authUserDetails := models.PaymentResourceRest{
			Status: "test",
		}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		w := httptest.NewRecorder()
		test := PaymentAdminAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("No authorised identity", t, func() {
		path := "/admin/payments/bulk-refunds"
		req, err := http.NewRequest("POST", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{})
		req.Header.Set("Eric-Identity", "authorised_identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "noroles")
		// Pass no ID (identity)
		authUserDetails := authentication.AuthUserDetails{}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		w := httptest.NewRecorder()
		test := PaymentAdminAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("User has admin role and request is GET", t, func() {
		path := "/admin/payments/bulk-refunds"
		req, err := http.NewRequest("GET", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{})
		req.Header.Set("Eric-Identity", "identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "/admin/payments-bulk-refunds")
		authUserDetails := authentication.AuthUserDetails{
			ID: "identity",
		}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		test := PaymentAdminAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("Success - User has admin role and request is POST", t, func() {
		path := "/admin/payments/bulk-refunds"
		req, err := http.NewRequest("POST", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{})
		req.Header.Set("Eric-Identity", "identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "/admin/payments-bulk-refunds")
		authUserDetails := authentication.AuthUserDetails{
			ID: "identity",
		}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		test := PaymentAdminAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusOK)
	})

	Convey("Success - User has elevated privileges", t, func() {
		path := "/admin/payments/bulk-refunds"
		req, err := http.NewRequest("GET", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "identity")
		req.Header.Set("Eric-Identity-Type", "key")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Key-Roles", "*")
		authUserDetails := authentication.AuthUserDetails{
			ID: "api-key-user",
		}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		test := PaymentAdminAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusOK)
	})

	Convey("Success - User has payment privileges", t, func() {
		path := "/admin/payments/bulk-refunds"
		req, err := http.NewRequest("POST", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{})
		req.Header.Set("Eric-Identity", "identity")
		req.Header.Set("Eric-Identity-Type", "key")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Key-Privileges", "payment")
		authUserDetails := authentication.AuthUserDetails{
			ID: "api-key-user",
		}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		test := PaymentAdminAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusOK)
	})
}
