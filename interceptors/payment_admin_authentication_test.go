package interceptors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/chs.go/authentication"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/jarcoal/httpmock"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitAdminUserPaymentInterceptor(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	cfg.DomainAllowList = resourceURL

	Convey("No oauth2 or API key identity type", t, func() {
		path := "/admin/payments/bulk-refunds"
		req, err := http.NewRequest("POST", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{})
		req.Header.Set("Eric-Identity", "authorised_identity")
		req.Header.Set("Eric-Identity-Type", "invalid")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "noroles")
		// Pass no ID (identity)
		authUserDetails := authentication.AuthUserDetails{}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		w := httptest.NewRecorder()
		test := PaymentAdminAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("Invalid user details in context", t, func() {
		path := "/admin/payments/bulk-refunds"
		req, err := http.NewRequest("POST", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{})
		req.Header.Set("Eric-Identity", "authorised_identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "noroles")
		// The details have to be in a authUserDetails struct, so pass a different struct to fail
		authUserDetails := models.PaymentResourceRest{
			Status: "test",
		}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		w := httptest.NewRecorder()
		test := PaymentAdminAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("No authorised identity", t, func() {
		path := "/admin/payments/bulk-refunds"
		req, err := http.NewRequest("POST", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{})
		req.Header.Set("Eric-Identity", "authorised_identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "noroles")
		// Pass no ID (identity)
		authUserDetails := authentication.AuthUserDetails{}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		w := httptest.NewRecorder()
		test := PaymentAdminAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("User has admin role and request is GET", t, func() {
		path := "/admin/payments/bulk-refunds"
		req, err := http.NewRequest("GET", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{})
		req.Header.Set("Eric-Identity", "identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "/admin/payments-bulk-refunds")
		authUserDetails := authentication.AuthUserDetails{
			ID: "identity",
		}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		test := PaymentAdminAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusUnauthorized)
	})

	Convey("Success - User has admin role and request is POST", t, func() {
		path := "/admin/payments/bulk-refunds"
		req, err := http.NewRequest("POST", path, nil)
		So(err, ShouldBeNil)
		req = mux.SetURLVars(req, map[string]string{})
		req.Header.Set("Eric-Identity", "identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "/admin/payments-bulk-refunds")
		authUserDetails := authentication.AuthUserDetails{
			ID: "identity",
		}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		test := PaymentAdminAuthenticationIntercept(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusOK)
	})
}
