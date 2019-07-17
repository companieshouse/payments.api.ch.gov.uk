package handlers

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
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
	cfg.DomainWhitelist = "http://dummy-url"
	cfg.ExpiryTimeInMinutes = "90"
	Convey("Invalid PaymentResourceRest", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		HandleGetPaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})
	Convey("Payment session expired", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &models.PaymentResourceRest{CreatedAt: time.Now().Add(-time.Hour * 2)})
		w := httptest.NewRecorder()
		Register(mux.NewRouter(), *cfg)
		HandleGetPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 200)
	})
	Convey("Valid PaymentResourceRest", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &models.PaymentResourceRest{CreatedAt: time.Now()})
		w := httptest.NewRecorder()
		Register(mux.NewRouter(), *cfg)
		HandleGetPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 200)
	})
}

func TestUnitHandlePatchPaymentSession(t *testing.T) {
	cfg, _ := config.Get()
	cfg.DomainWhitelist = "http://dummy-url"
	cfg.ExpiryTimeInMinutes = "90"
	Convey("Request Body empty", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &models.PaymentResourceRest{CreatedAt: time.Now()})
		req.Body = nil
		w := httptest.NewRecorder()
		Register(mux.NewRouter(), *cfg)
		HandlePatchPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Request Body invalid", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &models.PaymentResourceRest{CreatedAt: time.Now()})
		w := httptest.NewRecorder()
		Register(mux.NewRouter(), *cfg)
		HandlePatchPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Invalid PaymentResourceRest due to no context", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		HandlePatchPaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("Payment method not supplied", t, func() {
		reqBody := []byte(`{"amount":"12"}`)
		req := httptest.NewRequest("GET", "/test", ioutil.NopCloser(bytes.NewReader(reqBody)))
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &models.PaymentResourceRest{CreatedAt: time.Now()})
		w := httptest.NewRecorder()
		Register(mux.NewRouter(), *cfg)
		HandlePatchPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Payment session expired", t, func() {
		reqBody := []byte(`{"amount":"12", "payment_method": "GovPay"}`)
		req := httptest.NewRequest("GET", "/test", ioutil.NopCloser(bytes.NewReader(reqBody)))
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &models.PaymentResourceRest{CreatedAt: time.Now().Add(-time.Hour * 2)})
		w := httptest.NewRecorder()
		Register(mux.NewRouter(), *cfg)
		HandlePatchPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 403)
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

	Convey("Error getting payment status from Gov Pay", t, func() {

		paymentResource := models.PaymentResourceRest{
			Status: service.InProgress.String(),
			Costs:  []models.CostResourceRest{{ClassOfPayment: []string{"class"}}},
		}

		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &paymentResource)
		w := httptest.NewRecorder()
		HandleGetPaymentDetails(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})
}
