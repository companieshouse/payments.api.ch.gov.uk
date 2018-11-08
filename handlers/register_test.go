package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/gorilla/pat"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitRegisterRoutes(t *testing.T) {
	Convey("Register routes", t, func() {
		router := pat.New()
		cfg, _ := config.Get()
		Register(router, *cfg)
		So(router.GetRoute("get-healthcheck"), ShouldNotBeNil)
		So(router.GetRoute("create-payment"), ShouldNotBeNil)
		So(router.GetRoute("get-payment"), ShouldNotBeNil)
		So(router.GetRoute("create-payment-journey"), ShouldNotBeNil)
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