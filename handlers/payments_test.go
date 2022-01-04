package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/companieshouse/chs.go/authentication"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/golang/mock/gomock"
	"github.com/jarcoal/httpmock"
	"github.com/plutov/paypal/v4"
	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitHandleCreatePaymentSession(t *testing.T) {
	Convey("Request Body Empty", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		HandleCreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Request Body Invalid", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		HandleCreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Error creating payment resource - invalid data", t, func() {
		b := []byte(`{"redirect_uri":"invalid", "reference":"invalid", "resource": "invalid", "state": "invalid"}`)
		req := httptest.NewRequest("GET", "/test", bytes.NewReader(b))
		w := httptest.NewRecorder()

		HandleCreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Error creating payment resource - no authentication details", t, func() {
		paymentService = &service.PaymentService{
			Config: config.Config{DomainAllowList: "http://www.companieshouse.gov.uk"},
		}

		b := []byte(`{"redirect_uri":"http://www.companieshouse.gov.uk", "reference":"invalid", "resource": "http://www.companieshouse.gov.uk", "state": "invalid"}`)
		req := httptest.NewRequest("GET", "/test", bytes.NewReader(b))
		w := httptest.NewRecorder()

		HandleCreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Create payment resource - success", t, func() {
		mockDao := dao.NewMockDAO(gomock.NewController(t))
		mockDao.EXPECT().CreatePaymentResource(gomock.Any()).Return(nil)

		paymentService = &service.PaymentService{
			DAO:    mockDao,
			Config: config.Config{DomainAllowList: "https://www.companieshouse.gov.uk"},
		}

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "https://www.companieshouse.gov.uk", jsonResponse)

		b := []byte(`{"redirect_uri":"https://www.companieshouse.gov.uk", "reference":"invalid", "resource": "https://www.companieshouse.gov.uk", "state": "invalid"}`)
		req := httptest.NewRequest("GET", "/test", bytes.NewReader(b))
		w := httptest.NewRecorder()

		var userDetails = authentication.AuthUserDetails{
			Email:    "email@companieshouse.gov.uk",
			Forename: "forename",
			ID:       "id",
			Surname:  "surname",
		}

		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, userDetails)

		HandleCreatePaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusCreated)
	})

}

func TestUnitHandleGetPaymentSession(t *testing.T) {

	Convey("Invalid PaymentResourceRest", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		HandleGetPaymentSession(w, req)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Error checking expiry status", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)

		paymentResource := models.PaymentResourceRest{}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &paymentResource)

		w := httptest.NewRecorder()
		HandleGetPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Payment session expired", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		twoHours, _ := time.ParseDuration("2h")
		paymentResource := models.PaymentResourceRest{
			CreatedAt: time.Now().Add(-twoHours),
			Status:    service.InProgress.String(),
		}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &paymentResource)

		cfg, _ := config.Get()
		cfg.ExpiryTimeInMinutes = "90"
		paymentService = &service.PaymentService{
			DAO:    dao.NewMockDAO(gomock.NewController(t)),
			Config: *cfg,
		}

		w := httptest.NewRecorder()
		HandleGetPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusOK)

		decoder := json.NewDecoder(w.Body)
		var rest models.PaymentResourceRest
		decoder.Decode(&rest)
		So(rest.Status, ShouldEqual, service.Expired.String())
	})
}

func TestUnitHandlePatchPaymentSession(t *testing.T) {
	cfg, _ := config.Get()
	mockCtrl := gomock.NewController(t)

	Convey("Invalid PaymentResourceRest due to no context", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		HandlePatchPaymentSession(w, req)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Error checking expiry status - config not set", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)

		paymentResource := models.PaymentResourceRest{}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &paymentResource)
		paymentService.Config.ExpiryTimeInMinutes = ""

		w := httptest.NewRecorder()
		HandlePatchPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Patch Payment Session - Payment session expired", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		twoHours, _ := time.ParseDuration("2h")
		paymentResource := models.PaymentResourceRest{
			CreatedAt: time.Now().Add(-twoHours),
			Status:    service.InProgress.String(),
		}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &paymentResource)

		paymentService.Config.ExpiryTimeInMinutes = "90"

		w := httptest.NewRecorder()
		HandlePatchPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusForbidden)
	})

	Convey("Invalid request body", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		paymentResource := models.PaymentResourceRest{
			CreatedAt: time.Now(),
		}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &paymentResource)

		w := httptest.NewRecorder()
		HandlePatchPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Neither method nor status supplied", t, func() {
		b := []byte(`{"status":"", "payment_method": ""}`)
		req := httptest.NewRequest("GET", "/test", bytes.NewReader(b))
		paymentResource := models.PaymentResourceRest{
			CreatedAt: time.Now(),
		}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &paymentResource)

		w := httptest.NewRecorder()
		HandlePatchPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Payment method not supplied", t, func() {
		b := []byte(`{"status":"pending", "payment_method": ""}`)
		req := httptest.NewRequest("GET", "/test", bytes.NewReader(b))
		paymentResource := models.PaymentResourceRest{
			CreatedAt: time.Now(),
		}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &paymentResource)

		w := httptest.NewRecorder()
		HandlePatchPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Error patching payment", t, func() {
		b := []byte(`{"status":"pending", "payment_method": "credit-card"}`)
		req := httptest.NewRequest("GET", "/test", bytes.NewReader(b))
		paymentResource := models.PaymentResourceRest{
			CreatedAt: time.Now(),
		}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &paymentResource)

		mockDao := dao.NewMockDAO(mockCtrl)
		payment := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{Status: "pending"},
		}
		mockDao.EXPECT().GetPaymentResource(gomock.Any()).Return(&payment, nil)
		paymentService = &service.PaymentService{
			DAO:    mockDao,
			Config: *cfg,
		}

		w := httptest.NewRecorder()
		HandlePatchPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Successful patch", t, func() {
		b := []byte(`{"status":"pending", "payment_method": "credit-card"}`)
		req := httptest.NewRequest("GET", "/test", bytes.NewReader(b))
		paymentResource := models.PaymentResourceRest{
			CreatedAt: time.Now(),
		}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &paymentResource)

		payment := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount: "10.00",
				Status: "pending",
				Links:  models.PaymentLinksDB{Resource: "companieshouse.gov.uk"},
			},
		}

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "companieshouse.gov.uk", jsonResponse)

		mockDao := dao.NewMockDAO(mockCtrl)
		mockDao.EXPECT().GetPaymentResource(gomock.Any()).Return(&payment, nil)
		mockDao.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)
		paymentService = &service.PaymentService{
			DAO:    mockDao,
			Config: *cfg,
		}

		w := httptest.NewRecorder()
		HandlePatchPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusOK)
	})

}

func TestUnitHandleGetPaymentDetails(t *testing.T) {

	cfg, _ := config.Get()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	Convey("Invalid request", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		mockPayPalSDK := service.NewMockPayPalSDK(mockCtrl)

		svc := service.ExternalPaymentProvidersService{
			GovPayService: service.GovPayService{
				PaymentService: *mockPaymentService,
			},
			PayPalService: service.PayPalService{
				Client:         mockPayPalSDK,
				PaymentService: *mockPaymentService,
			},
		}
		handler := HandleGetPaymentDetails(&svc)

		res := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		handler.ServeHTTP(res, req)

		HandleGetPaymentDetails(&svc)
		So(res.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Payment method not recognised", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		mockPayPalSDK := service.NewMockPayPalSDK(mockCtrl)

		svc := service.ExternalPaymentProvidersService{
			GovPayService: service.GovPayService{
				PaymentService: *mockPaymentService,
			},
			PayPalService: service.PayPalService{
				Client:         mockPayPalSDK,
				PaymentService: *mockPaymentService,
			},
		}
		handler := HandleGetPaymentDetails(&svc)

		res := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		paymentResource := models.PaymentResourceRest{}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &paymentResource)
		handler.ServeHTTP(res, req.WithContext(ctx))

		HandleGetPaymentDetails(&svc)
		So(res.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Error getting payment details from external provider", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		mockPayPalSDK := service.NewMockPayPalSDK(mockCtrl)

		svc := service.ExternalPaymentProvidersService{
			GovPayService: service.GovPayService{
				PaymentService: *mockPaymentService,
			},
			PayPalService: service.PayPalService{
				Client:         mockPayPalSDK,
				PaymentService: *mockPaymentService,
			},
		}
		handler := HandleGetPaymentDetails(&svc)

		res := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		paymentResource := models.PaymentResourceRest{PaymentMethod: "credit-card"}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &paymentResource)
		handler.ServeHTTP(res, req.WithContext(ctx))

		HandleGetPaymentDetails(&svc)
		So(res.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Get Payment Details - Credit Card Success", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		mockPayPalSDK := service.NewMockPayPalSDK(mockCtrl)

		svc := service.ExternalPaymentProvidersService{
			GovPayService: service.GovPayService{
				PaymentService: *mockPaymentService,
			},
			PayPalService: service.PayPalService{
				Client:         mockPayPalSDK,
				PaymentService: *mockPaymentService,
			},
		}
		handler := HandleGetPaymentDetails(&svc)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "companieshouse.gov.uk", jsonResponse)

		res := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		paymentResource := models.PaymentResourceRest{
			PaymentMethod: "credit-card",
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusURI: "companieshouse.gov.uk",
			},
			Costs: []models.CostResourceRest{
				{
					ClassOfPayment: []string{"penalty"},
				},
			},
		}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &paymentResource)
		handler.ServeHTTP(res, req.WithContext(ctx))

		HandleGetPaymentDetails(&svc)
		So(res.Code, ShouldEqual, http.StatusOK)
	})

	Convey("Get Payment Details - PayPal Success", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		mockPayPalSDK := service.NewMockPayPalSDK(mockCtrl)

		createTime, _ := time.Parse("2006-01-02", "2003-02-01")
		paypalStatus := paypal.Order{
			CreateTime: &createTime,
			Status:     "COMPLETED",
		}

		mockPayPalSDK.EXPECT().GetOrder(gomock.Any(), "123456").Return(&paypalStatus, nil)

		svc := service.ExternalPaymentProvidersService{
			GovPayService: service.GovPayService{
				PaymentService: *mockPaymentService,
			},
			PayPalService: service.PayPalService{
				Client:         mockPayPalSDK,
				PaymentService: *mockPaymentService,
			},
		}
		handler := HandleGetPaymentDetails(&svc)

		res := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		paymentResource := models.PaymentResourceRest{
			PaymentMethod: "PayPal",
			MetaData: models.PaymentResourceMetaDataRest{
				ExternalPaymentStatusID: "123456",
			},
			Costs: []models.CostResourceRest{
				{
					ClassOfPayment: []string{"penalty"},
				},
			},
		}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &paymentResource)
		handler.ServeHTTP(res, req.WithContext(ctx))

		HandleGetPaymentDetails(&svc)
		So(res.Code, ShouldEqual, http.StatusOK)
	})

}
