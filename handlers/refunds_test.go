package handlers

import (
	"bytes"
	"encoding/json"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUnitHandleCreateRefund(t *testing.T) {
	cfg, _ := config.Get()

	Convey("Request Body Empty", t, func() {
		req, _ := http.NewRequest("POST", "/payments/123/refunds", nil)
		w := httptest.NewRecorder()
		HandleCreateRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("No PaymentId", t, func() {
		refundRequest := models.CreateRefundRequest{}
		requestBody, _ := json.Marshal(refundRequest)

		req := httptest.NewRequest("POST", "/payments/123/refunds", bytes.NewBuffer(requestBody))
		Register(mux.NewRouter(), *cfg)
		w := httptest.NewRecorder()
		HandleCreateRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
		print(w.Body)
	})

	Convey("Invalid request body", t, func() {
		refundRequest := "string"
		requestBody, _ := json.Marshal(refundRequest)

		req := httptest.NewRequest("POST", "/payments/123/refunds", bytes.NewBuffer(requestBody))
		req = mux.SetURLVars(req, map[string]string{"paymentId": "123"})
		Register(mux.NewRouter(), *cfg)
		w := httptest.NewRecorder()
		HandleCreateRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})
}
