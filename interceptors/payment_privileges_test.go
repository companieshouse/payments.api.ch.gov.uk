package interceptors

import (
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
		req.Header.Set("ERIC-Identity", "123")
		w := httptest.NewRecorder()

		test := InternalOrPaymentPrivilegesIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusOK)
	})

	Convey("API Key with Payment Privilege", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("ERIC-Identity-Type", authentication.APIKeyIdentityType)
		req.Header.Set("ERIC-Authorised-Key-Privileges", "payment")
		req.Header.Set("ERIC-Identity", "123")
		w := httptest.NewRecorder()

		test := InternalOrPaymentPrivilegesIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusOK)
	})

	Convey("API Key without Payment Privilege", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("ERIC-Identity-Type", authentication.APIKeyIdentityType)
		req.Header.Set("ERIC-Identity", "123")
		w := httptest.NewRecorder()

		test := InternalOrPaymentPrivilegesIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
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

func TestUnitUserPaymentAuthenticationIntercept(t *testing.T) {

	Convey("Not API Key or OAuth", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		test := UserPaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("No authorised identity", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("ERIC-Identity-Type", authentication.APIKeyIdentityType)
		w := httptest.NewRecorder()

		test := UserPaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("No authorised user", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("ERIC-Identity-Type", authentication.Oauth2IdentityType)
		req.Header.Set("ERIC-Identity", "123")
		w := httptest.NewRecorder()

		test := UserPaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("Authorised user length 1", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("ERIC-Identity-Type", authentication.Oauth2IdentityType)
		req.Header.Set("ERIC-Identity", "123")
		req.Header.Set("ERIC-Authorised-User", "test@companieshouse.gov.uk")
		w := httptest.NewRecorder()

		test := UserPaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusOK)
	})

	Convey("Authorised user length 2", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("ERIC-Identity-Type", authentication.Oauth2IdentityType)
		req.Header.Set("ERIC-Identity", "123")
		req.Header.Set("ERIC-Authorised-User", "test@companieshouse.gov.uk;forename")
		w := httptest.NewRecorder()

		test := UserPaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusOK)
	})

	Convey("Authorised user length 3", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("ERIC-Identity-Type", authentication.Oauth2IdentityType)
		req.Header.Set("ERIC-Identity", "123")
		req.Header.Set("ERIC-Authorised-User", "test@companieshouse.gov.uk;forename;surname")
		w := httptest.NewRecorder()

		test := UserPaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusOK)
	})

	Convey("API Key user", t, func() {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("ERIC-Identity-Type", authentication.APIKeyIdentityType)
		req.Header.Set("ERIC-Identity", "123")
		w := httptest.NewRecorder()

		test := UserPaymentAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, http.StatusOK)
	})
}
