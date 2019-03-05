package interceptors

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	. "github.com/smartystreets/goconvey/convey"
)

// GetTestHandler returns a http.HandlerFunc for testing http middleware
func GetTestHandler() http.HandlerFunc {
	fn := func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	return http.HandlerFunc(fn)
}

func TestUnitUserAuthenticationInterceptor(t *testing.T) {

	Convey("Incorrect identity type", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "authorised_identity")
		req.Header.Set("Eric-Identity-Type", "notoauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "notauth2")

		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		test := UserAuthenticationInterceptor(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, 401)
	})

	Convey("No identity in request", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "notauth2")

		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		test := UserAuthenticationInterceptor(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, 401)
	})

	Convey("No authorised user in request", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "authorised_identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-Roles", "notauth2")

		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		test := UserAuthenticationInterceptor(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, 401)
	})

	Convey("Happy path with populated headers", t, func() {
		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		req.Header.Set("Eric-Identity", "authorised_identity")
		req.Header.Set("Eric-Identity-Type", "oauth2")
		req.Header.Set("ERIC-Authorised-User", "test@test.com;test;user")
		req.Header.Set("ERIC-Authorised-Roles", "notauth2")

		So(err, ShouldBeNil)

		w := httptest.NewRecorder()
		test := UserAuthenticationInterceptor(GetTestHandler())
		test.ServeHTTP(w, req)
		So(w.Code, ShouldEqual, 200)
	})
}
