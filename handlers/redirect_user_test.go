package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/smartystreets/goconvey/convey"
)

func TestRedirectUser_NoDuplicateQuery(t *testing.T) {
	convey.Convey("redirectUser should preserve existing query params without duplication", t, func() {

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)

		redirectURI := "http://dummy-url?lang=cy"

		params := models.RedirectParams{
			State:  "abc",
			Ref:    "123",
			Status: "paid",
		}

		redirectUser(w, r, redirectURI, params)

		location := w.Header().Get("Location")

		convey.So(strings.Count(location, "?"), convey.ShouldEqual, 1)
		convey.So(strings.Count(location, "lang=cy"), convey.ShouldEqual, 1)
	})
}
