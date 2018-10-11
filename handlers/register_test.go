package handlers

import (
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
		So(router.GetRoute("create-paymentjourney"), ShouldNotBeNil)
	})
}
