package interceptors

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/chs.go/authentication"
	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitInternalOrPaymentPrivilegesIntercept(t *testing.T) {

	Convey("Not API Key", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		test := InternalOrPaymentPrivilegesIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("API Key not authorised", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("ERIC-Identity-Type", authentication.APIKeyIdentityType)
		w := httptest.NewRecorder()

		test := InternalOrPaymentPrivilegesIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("API Key with Internal Privilege", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("ERIC-Identity-Type", authentication.APIKeyIdentityType)
		req.Header.Set("ERIC-Authorised-Key-Roles", "*")
		w := httptest.NewRecorder()

		test := InternalOrPaymentPrivilegesIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusOK)
	})

	Convey("API Key with Payment Privilege", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("ERIC-Identity-Type", authentication.APIKeyIdentityType)
		req.Header.Set("ERIC-Authorised-Key-Privileges", "payment")
		w := httptest.NewRecorder()

		test := InternalOrPaymentPrivilegesIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusOK)
	})
}

func TestUnitOauth2OrPaymentPrivilegesIntercept(t *testing.T) {

	Convey("Not API Key or OAuth", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		test := Oauth2OrPaymentPrivilegesIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("API Key without payment privilege", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("ERIC-Identity-Type", authentication.APIKeyIdentityType)
		w := httptest.NewRecorder()

		test := Oauth2OrPaymentPrivilegesIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
		fmt.Println("ENM test end")
	})

	Convey("API Key with payment privilege", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("ERIC-Identity-Type", authentication.APIKeyIdentityType)
		req.Header.Set("ERIC-Authorised-Key-Privileges", "payment")
		w := httptest.NewRecorder()

		test := Oauth2OrPaymentPrivilegesIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusOK)
	})

	Convey("OAuth2", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("ERIC-Identity-Type", authentication.Oauth2IdentityType)
		w := httptest.NewRecorder()

		test := Oauth2OrPaymentPrivilegesIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusOK)
	})
}
