package interceptors

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/chs.go/authentication"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitElevatedOrPaymentPrivilegesIntercept(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	Convey("Not API Key", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		test := ElevatedOrPaymentPrivilegesIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("API Key not authorised", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("ERIC-Identity-Type", authentication.APIKeyIdentityType)
		w := httptest.NewRecorder()

		test := ElevatedOrPaymentPrivilegesIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("API Key with Internal Privilege", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("ERIC-Identity-Type", authentication.APIKeyIdentityType)
		req.Header.Set("ERIC-Authorised-Key-Roles", "*")
		w := httptest.NewRecorder()

		test := ElevatedOrPaymentPrivilegesIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusOK)
	})

	Convey("API Key with Payment Privilege", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("ERIC-Identity-Type", authentication.APIKeyIdentityType)
		req.Header.Set("ERIC-Authorised-Key-Privileges", "payment")
		w := httptest.NewRecorder()

		test := ElevatedOrPaymentPrivilegesIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusOK)
	})
}
