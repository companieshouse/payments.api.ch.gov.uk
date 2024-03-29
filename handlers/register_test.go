package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitRegisterRoutes(t *testing.T) {
	Convey("Register routes", t, func() {
		router := mux.NewRouter()
		cfg, _ := config.Get()
		cfg.SecureAppCostsRegex = "\\/secure-app-regex-test\\/"

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockDao := dao.NewMockDAO(mockCtrl)
		service.SetEmptyPaypalClientForUnitTests()

		Register(router, *cfg, mockDao)
		So(router.GetRoute("get-healthcheck"), ShouldNotBeNil)
		So(router.GetRoute("create-payment"), ShouldNotBeNil)
		So(router.GetRoute("get-payment"), ShouldNotBeNil)
		So(router.GetRoute("get-payment-details"), ShouldNotBeNil)
		So(router.GetRoute("check-payment-status"), ShouldNotBeNil)
		So(router.GetRoute("create-refund"), ShouldNotBeNil)
		So(router.GetRoute("get-refunds"), ShouldNotBeNil)
		So(router.GetRoute("update-refund"), ShouldNotBeNil)
		So(router.GetRoute("patch-payment"), ShouldNotBeNil)
		So(router.GetRoute("create-external-payment-journey"), ShouldNotBeNil)
		So(router.GetRoute("handle-govpay-callback"), ShouldNotBeNil)
		So(router.GetRoute("handle-paypal-callback"), ShouldNotBeNil)
		So(router.GetRoute("bulk-refund-govpay"), ShouldNotBeNil)
		So(router.GetRoute("bulk-refund-paypal"), ShouldNotBeNil)
		So(router.GetRoute("get-refund-statuses"), ShouldNotBeNil)
		So(router.GetRoute("process-bulk-refund"), ShouldNotBeNil)
		So(router.GetRoute("process-pending-refunds"), ShouldNotBeNil)
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
