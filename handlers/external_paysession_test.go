package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitHandleCreateExternalPaymentJourney(t *testing.T) {
	Convey("Invalid PaymentResourceRest in Request", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		HandleCreateExternalPaymentJourney(w, req)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Error creating external payment journey", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		paymentResource := models.PaymentResourceRest{
			Status: service.InProgress.String(),
		}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &paymentResource)
		w := httptest.NewRecorder()
		HandleCreateExternalPaymentJourney(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Error creating external payment journey - bad request", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), helpers.ContextKeyPaymentSession, &models.PaymentResourceRest{})
		w := httptest.NewRecorder()
		HandleCreateExternalPaymentJourney(w, req.WithContext(ctx))
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})

}
