package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/golang/mock/gomock"

	"github.com/companieshouse/payments.api.ch.gov.uk/dao"

	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	. "github.com/companieshouse/payments.api.ch.gov.uk/service"
	. "github.com/smartystreets/goconvey/convey"
)

func serveHandleCreateExternalPaymentJourney(mockExternalProviderService ExternalPaymentProvidersService, req *http.Request) *httptest.ResponseRecorder {
	handler := HandleCreateExternalPaymentJourney(&mockExternalProviderService)

	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	return res
}

func CreateMockExternalPaymentProvidersService(mockPayPalService PayPalService, mockGovPayService GovPayService) ExternalPaymentProvidersService {
	return ExternalPaymentProvidersService{
		GovPayService: mockGovPayService,
		PayPalService: mockPayPalService,
	}
}

func TestUnitHandleCreateExternalPaymentJourney(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
	mockPayPalSDK := NewMockPaypalSDK(mockCtrl)

	// Generate a mock external provider service using mocks for both PayPal and GovPay
	mockExternalProviderService := CreateMockExternalPaymentProvidersService(
		PayPalService{
			Client:         mockPayPalSDK,
			PaymentService: *mockPaymentService,
		},
		GovPayService{
			PaymentService: *mockPaymentService,
		})

	Convey("Invalid PaymentResourceRest in Request", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		res := serveHandleCreateExternalPaymentJourney(mockExternalProviderService, req)
		So(res.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Error creating external payment journey", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		paymentResource := models.PaymentResourceRest{
			Status: service.InProgress.String(),
		}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &paymentResource)
		res := serveHandleCreateExternalPaymentJourney(mockPaypalService, req.WithContext(ctx))
		So(res.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Error creating external payment journey - bad request", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &models.PaymentResourceRest{})
		res := serveHandleCreateExternalPaymentJourney(mockPaypalService, req.WithContext(ctx))
		So(res.Code, ShouldEqual, http.StatusBadRequest)
	})

}
