package handlers

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

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
		ctx := req.Context()
		ctx = context.WithValue(ctx, helpers.ContextKeyPaymentSession, &models.PaymentResourceRest{})
		w := httptest.NewRecorder()
		HandleGetPaymentSession(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 200)
	})
}

func TestUnitHandlePatchPaymentSession(t *testing.T) {
	Convey("Payment ID not supplied", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		HandlePatchPaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Request Body empty", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		req.Body = nil
		w := httptest.NewRecorder()
		HandlePatchPaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Request Body invalid", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		w := httptest.NewRecorder()
		HandlePatchPaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Payment method not supplied", t, func() {
		reqBody := []byte(`{"amount":"12"}`)
		req := httptest.NewRequest("GET", "/test", ioutil.NopCloser(bytes.NewReader(reqBody)))
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		w := httptest.NewRecorder()
		HandlePatchPaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})
}
