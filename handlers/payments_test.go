package handlers

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
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

	Convey("Invalid Request", t, func() {
		reqBody := []byte(`{"redirect_uri": "", "reference": "", "resource": "", "state": ""}`)
		req := httptest.NewRequest("GET", "/test", ioutil.NopCloser(bytes.NewReader(reqBody)))
		w := httptest.NewRecorder()
		HandleCreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Error Creating Payment Resource", t, func() {
		reqBody := []byte(`{"redirect_uri": "uri", "reference": "ref", "resource": "ref", "state": "state"}`)
		req := httptest.NewRequest("GET", "/test", ioutil.NopCloser(bytes.NewReader(reqBody)))
		w := httptest.NewRecorder()
		HandleCreatePaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

}

func TestUnitHandleGetPaymentSession(t *testing.T) {
	Convey("Invalid PaymentResourceRest", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		HandleGetPaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})
	Convey("Valid PaymentResourceRest", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &models.PaymentResourceRest{CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute * 60)})
		w := httptest.NewRecorder()
		HandleGetPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 200)
	})
}

func TestUnitHandlePatchPaymentSession(t *testing.T) {
	Convey("Request Body empty", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &models.PaymentResourceRest{CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute * 60)})
		req.Body = nil
		w := httptest.NewRecorder()
		HandlePatchPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Request Body invalid", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &models.PaymentResourceRest{CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute * 60)})
		w := httptest.NewRecorder()
		HandlePatchPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Payment method not supplied", t, func() {
		reqBody := []byte(`{"amount":"12"}`)
		req := httptest.NewRequest("GET", "/test", ioutil.NopCloser(bytes.NewReader(reqBody)))
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &models.PaymentResourceRest{CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute * 60)})
		w := httptest.NewRecorder()
		HandlePatchPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Payment session expired", t, func() {
		reqBody := []byte(`{"amount":"12", "payment_method": "GovPay"}`)
		req := httptest.NewRequest("GET", "/test", ioutil.NopCloser(bytes.NewReader(reqBody)))
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &models.PaymentResourceRest{CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute * time.Duration(-60))})
		w := httptest.NewRecorder()
		HandlePatchPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 403)
	})
}
