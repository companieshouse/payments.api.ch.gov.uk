package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitHandleCreateRefund(t *testing.T) {

	Convey("Request Body Empty", t, func() {
		req, _ := http.NewRequest("POST", "/payments/123/refunds", nil)
		w := httptest.NewRecorder()
		HandleCreateRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})
}

func TestUnitHandleProcessPendingRefunds(t *testing.T) {

	Convey("Successful request", t, func() {
		req, _ := http.NewRequest("POST", "/payments/refunds/process-pending", nil)
		w := httptest.NewRecorder()
		HandleProcessPendingRefunds(w, req)
		So(w.Code, ShouldEqual, http.StatusOK)
	})
}
