package interceptors

import (
	"context"
	"fmt"
	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitUserPaymentInterceptor(t *testing.T) {

	Convey("No payment ID in request", t, func() {
		path := fmt.Sprintf("/payments/")
		req, err := http.NewRequest("Get", path, nil)
		req.Header.Set("Eric-Identity", "authorised_identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "notauth2")
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		test := PaymentAuthenticationInterceptor(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Invalid user details in context", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "authorised_identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "notauth2")
		// The details have to be in a authUserDetails struct, so pass a different struct to fail
		authUserDetails := models.PaymentResourceData{
			Status: "test",
		}
		ctx := context.WithValue(req.Context(), helpers.UserDetailsKey, authUserDetails)
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		test := PaymentAuthenticationInterceptor(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 500)
	})

	Convey("No authorised identity", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "authorised_identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "notauth2")
		// Pass no ID (identity)
		authUserDetails := models.AuthUserDetails{

		}
		ctx := context.WithValue(req.Context(), helpers.UserDetailsKey, authUserDetails)
		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		test := PaymentAuthenticationInterceptor(GetTestHandler())
		test.ServeHTTP(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, 401)
	})
}