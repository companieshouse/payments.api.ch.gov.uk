package handlers

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/jarcoal/httpmock.v1"
)

func TestUnitCreatePaymentSession(t *testing.T) {
	cfg, _ := config.Get()
	reqBody := []byte("{\"redirect_uri\": \"dummy-redirect-uri\",\"resource\": \"http://dummy-resource\",\"state\": \"dummy-state\",\"reference\": \"dummy-reference\"}")

	Convey("Empty Request Body", t, func() {
		req, err := http.NewRequest("GET", "", nil)
		So(err, ShouldBeNil)
		w := httptest.NewRecorder()
		createPaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Invalid Request Body", t, func() {
		req, err := http.NewRequest("GET", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader([]byte("invalid_body")))
		w := httptest.NewRecorder()
		createPaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	Convey("Invalid Resource Domain", t, func() {
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		w := httptest.NewRecorder()
		createPaymentSession(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	cfg.DomainWhitelist = "dummy-resource"

	Convey("Error gettting cost resource", t, func() {
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET", "http://dummy-resource", httpmock.NewErrorResponder(fmt.Errorf("error")))
		createPaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("Error reading cost resource", t, func() {
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET", "http://dummy-resource", httpmock.NewStringResponder(500, "string"))
		createPaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("Invalid user header", t, func() {
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
		req.Header.Set("Eric-Authorised-User", "test@companieshouse.gov.uk; forename=f; surname=s; invalid=invalid")
		w := httptest.NewRecorder()
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		var paymentResource createPaymentResource
		jsonResponse, _ := httpmock.NewJsonResponder(500, paymentResource)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		createPaymentSession(w, req)
		So(w.Code, ShouldEqual, 500)
	})

}
