package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
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
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		HandleGetPaymentDetails(w, req)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})
}
