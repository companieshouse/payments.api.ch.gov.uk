package service

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitCreateExternalPayment(t *testing.T) {
	Convey("Empty Request Body", t, func() {
		req, err := http.NewRequest("GET", "", nil)
		So(err, ShouldBeNil)
		w := httptest.NewRecorder()
		CreateExternalPaymentJourney(w, req)
		So(w.Code, ShouldEqual, 400)

	})

	Convey("Invalid Request Body", t, func() {
		req, err := http.NewRequest("GET", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader([]byte("invalid_body")))
		w := httptest.NewRecorder()
		CreateExternalPaymentJourney(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	reqBodyWrongPayment := []byte("{\"payment_method\": \"PayPal\",\"resource\": \"http://dummy-resource\"}")

	Convey("Invalid Payment Method", t, func() {
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBodyWrongPayment))
		w := httptest.NewRecorder()
		CreateExternalPaymentJourney(w, req)
		So(w.Code, ShouldEqual, 400)
	})

	reqBodyCorrectPayment := []byte("{\"payment_method\": \"GovPay\",\"resource\": \"http://dummy-resource\"}")
	Convey("Valid External Payment Request", t, func() {
		req, err := http.NewRequest("Get", "", nil)
		So(err, ShouldBeNil)
		req.Body = ioutil.NopCloser(bytes.NewReader(reqBodyCorrectPayment))
		w := httptest.NewRecorder()
		CreateExternalPaymentJourney(w, req)
		So(w.Code, ShouldEqual, 200)
	})
}
