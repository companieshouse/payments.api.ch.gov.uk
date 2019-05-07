package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/jarcoal/httpmock"
	. "github.com/smartystreets/goconvey/convey"
)

var defaultCost = models.CostResourceRest{
	Amount:                  "10",
	AvailablePaymentMethods: []string{"GovPay"},
	ClassOfPayment:          []string{"class"},
	Description:             "desc",
	DescriptionIdentifier:   "identifier",
	Links: models.CostLinksRest{Self: "self"},
}

var defaultCosts = models.CostsRest{
	Description: "costs_desc",
	Costs:       []models.CostResourceRest{defaultCost},
}

func createMockPaymentService(mockDAO *dao.MockDAO, cfg *config.Config) *service.PaymentService {
	return &service.PaymentService{
		DAO:    mockDAO,
		Config: *cfg,
	}
}

func TestUnitHandleGovPayCallback(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	cfg.DomainWhitelist = "http://dummy-url"
	Convey("Payment ID not supplied", t, func() {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		HandleGovPayCallback(w, req)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Error getting payment session", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		paymentService = createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(nil, fmt.Errorf("error"))

		req := httptest.NewRequest("GET", "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		w := httptest.NewRecorder()
		HandleGovPayCallback(w, req)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Payment session not found", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		paymentService = createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(nil, nil)

		req := httptest.NewRequest("GET", "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		w := httptest.NewRecorder()
		HandleGovPayCallback(w, req)
		So(w.Code, ShouldEqual, http.StatusNotFound)
	})

	Convey("Payment method not recognised", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		paymentService = createMockPaymentService(mock, cfg)
		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount:        "10.00",
				PaymentMethod: "invalid",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt: time.Now(),
			},
		}
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-url", jsonResponse)

		req := httptest.NewRequest("GET", "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		w := httptest.NewRecorder()
		HandleGovPayCallback(w, req)
		So(w.Code, ShouldEqual, http.StatusPreconditionFailed)
	})

	Convey("Error getting payment status from GovPay", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		paymentService = createMockPaymentService(mock, cfg)
		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount:        "10.00",
				PaymentMethod: "GovPay",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt: time.Now(),
			},
		}
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-url", jsonResponse)

		req := httptest.NewRequest("GET", "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		w := httptest.NewRecorder()
		HandleGovPayCallback(w, req)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Error setting payment status", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		paymentService = createMockPaymentService(mock, cfg)
		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount:        "10.00",
				PaymentMethod: "GovPay",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt: time.Now(),
			},
		}
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(fmt.Errorf("error"))

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-url", jsonResponse)

		govPayResponse := models.IncomingGovPayResponse{}
		govPayJSONResponse, _ := httpmock.NewJsonResponder(200, govPayResponse)
		httpmock.RegisterResponder("GET", cfg.GovPayURL, govPayJSONResponse)

		req := httptest.NewRequest("GET", "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		w := httptest.NewRecorder()
		HandleGovPayCallback(w, req)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Error setting payment status", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		paymentService = createMockPaymentService(mock, cfg)
		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount:        "10.00",
				PaymentMethod: "GovPay",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt: time.Now(),
			},
		}
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jSONResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-url", jSONResponse)

		govPayResponse := models.IncomingGovPayResponse{}
		govPayJSONResponse, _ := httpmock.NewJsonResponder(200, govPayResponse)
		httpmock.RegisterResponder("GET", cfg.GovPayURL, govPayJSONResponse)

		req := httptest.NewRequest("GET", "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		w := httptest.NewRecorder()
		HandleGovPayCallback(w, req)
		So(w.Code, ShouldEqual, http.StatusSeeOther)
	})
}
