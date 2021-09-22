package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/golang/mock/gomock"

	"github.com/companieshouse/payments.api.ch.gov.uk/dao"

	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	. "github.com/companieshouse/payments.api.ch.gov.uk/service"
	. "github.com/smartystreets/goconvey/convey"
)

func serveHandleCreateExternalPaymentJourney(paypalSvc PayPalService, req *http.Request) *httptest.ResponseRecorder {
	handler := HandleCreateExternalPaymentJourney(&paypalSvc)

	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	return res
}

func createMockPaypalService(sdk PayPalSDK, service *PaymentService) PayPalService {
	return PayPalService{
		Client:         sdk,
		PaymentService: *service,
	}
}

func TestUnitHandleCreateExternalPaymentJourney(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
	mockPayPalSDK := NewMockPaypalSDK(mockCtrl)
	mockPaypalService := createMockPaypalService(mockPayPalSDK, mockPaymentService)

	Convey("Invalid PaymentResourceRest in Request", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		res := serveHandleCreateExternalPaymentJourney(mockPaypalService, req)
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
