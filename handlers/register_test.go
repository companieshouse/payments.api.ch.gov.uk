package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/gorilla/mux"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitRegisterRoutes(t *testing.T) {
	Convey("Register routes", t, func() {
		router := mux.NewRouter()
		cfg, _ := config.Get()
		cfg.SecureAppCostsRegex = "\\/secure-app-regex-test\\/"
		Register(router, *cfg)
		So(router.GetRoute("get-healthcheck"), ShouldNotBeNil)
		So(router.GetRoute("create-payment"), ShouldNotBeNil)
		So(router.GetRoute("get-payment"), ShouldNotBeNil)
		So(router.GetRoute("get-payment-details"), ShouldNotBeNil)
		So(router.GetRoute("patch-payment"), ShouldNotBeNil)
		So(router.GetRoute("create-external-payment-journey"), ShouldNotBeNil)
		So(router.GetRoute("handle-govpay-callback"), ShouldNotBeNil)
		So(router.GetRoute("create-refund"), ShouldNotBeNil)
		So(router.GetRoute("update-refund"), ShouldNotBeNil)
	})
}

func TestUnitGetHealthCheck(t *testing.T) {
	Convey("Get HealthCheck", t, func() {
		req, err := http.NewRequest("GET", "", nil)
		So(err, ShouldBeNil)
		w := httptest.NewRecorder()
		healthCheck(w, req)
		So(w.Code, ShouldEqual, 200)
	})
}
