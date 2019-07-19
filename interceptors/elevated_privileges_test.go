package interceptors

import (
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitElevatedPrivilegesInterceptor(t *testing.T) {

	Convey("Incorrect identity type", t, func() {
		req, _ := http.NewRequest("GET", "/payments", nil)
		w := httptest.NewRecorder()
		ElevatedPrivilegesInterceptor(GetTestHandler()).ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("No elevated privileges", t, func() {
		req, _ := http.NewRequest("GET", "/payments", nil)
		req.Header.Set("ERIC-Identity-Type", "key")
		w := httptest.NewRecorder()
		ElevatedPrivilegesInterceptor(GetTestHandler()).ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("Elevated privileges", t, func() {
		req, _ := http.NewRequest("GET", "/payments", nil)
		req.Header.Set("ERIC-Identity-Type", "key")
		req.Header.Set("ERIC-Authorised-Key-Roles", "*")
		w := httptest.NewRecorder()
		ElevatedPrivilegesInterceptor(GetTestHandler()).ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusOK)
	})

}
