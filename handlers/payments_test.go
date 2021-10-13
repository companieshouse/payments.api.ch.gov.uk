package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitHandleCreatePaymentSession(t *testing.T) {
	Convey("Request Body Empty", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		HandleCreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Request Body Invalid", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		HandleCreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

}

func TestUnitHandleGetPaymentSession(t *testing.T) {
	cfg, _ := config.Get()
	cfg.DomainAllowList = "http://dummy-url"
	cfg.ExpiryTimeInMinutes = "90"
	cfg.PaypalEnv = "test"
	cfg.PaypalClientID = "123"
	cfg.PaypalSecret = "123"

	Convey("Invalid PaymentResourceRest", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		HandleGetPaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})
}

func TestUnitHandlePatchPaymentSession(t *testing.T) {
	cfg, _ := config.Get()
	cfg.DomainAllowList = "http://dummy-url"
	cfg.ExpiryTimeInMinutes = "90"

	Convey("Invalid PaymentResourceRest due to no context", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		HandlePatchPaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})
}

func TestUnitHandleGetPaymentDetails(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	Convey("Error getting payment session", t, func() {

		cfg, _ := config.Get()
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
		handler := HandleCreateExternalPaymentJourney(&svc)

		res := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		handler.ServeHTTP(res, req)

		HandleGetPaymentDetails(&svc)
		So(res.Code, ShouldEqual, http.StatusBadRequest)
	})
}
