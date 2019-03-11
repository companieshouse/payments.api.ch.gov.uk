package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/jarcoal/httpmock.v1"
)

func TestUnitCreateExternalPayment(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	cfg.GovPayURL = "http://dummy-govpay-url"

	Convey("Payment session not in progress", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)

		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		externalPaymentJourney, responseType, err := mockPaymentService.CreateExternalPaymentJourney(req, &models.PaymentResourceRest{})

		So(externalPaymentJourney, ShouldBeNil)
		So(responseType.String(), ShouldEqual, InvalidData.String())
		So(err.Error(), ShouldEqual, "payment session is not in progress")
	})

	Convey("Error communicating with GOV.UK Pay", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)

		req := httptest.NewRequest("", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET", cfg.GovPayURL, httpmock.NewStringResponder(400, "error"))

		paymentSession := models.PaymentResourceRest{
			PaymentMethod: "GovPay",
			Status:        InProgress.String(),
		}

		externalPaymentJourney, responseType, err := mockPaymentService.CreateExternalPaymentJourney(req, &paymentSession)
		So(externalPaymentJourney, ShouldBeNil)
		So(responseType.String(), ShouldEqual, Error.String())
		So(err.Error(), ShouldEqual, `error communicating with GovPay: [error converting amount to pay to pence: [strconv.Atoi: parsing "": invalid syntax]]`)
	})

	Convey("No NextURL received from GOV.UK Pay", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		req := httptest.NewRequest("", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusCreated, &models.IncomingGovPayResponse{})
		httpmock.RegisterResponder("POST", cfg.GovPayURL, jsonResponse)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentSession := models.PaymentResourceRest{
			PaymentMethod: "GovPay",
			Amount:        "3",
			Status:        InProgress.String(),
			Costs:         []models.CostResourceRest{costResource},
		}

		externalPaymentJourney, responseType, err := mockPaymentService.CreateExternalPaymentJourney(req, &paymentSession)
		So(externalPaymentJourney, ShouldBeNil)
		So(responseType.String(), ShouldEqual, Error.String())
		So(err.Error(), ShouldEqual, "no NextURL returned from GovPay")
	})

	Convey("Create External Payment Journey - success", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		req := httptest.NewRequest("", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusCreated, &models.IncomingGovPayResponse{
			GovPayLinks: models.GovPayLinks{
				NextURL: models.NextURL{
					HREF: "response_url",
				},
			},
		})
		httpmock.RegisterResponder("POST", cfg.GovPayURL, jsonResponse)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentSession := models.PaymentResourceRest{
			PaymentMethod: "GovPay",
			Amount:        "4",
			Status:        InProgress.String(),
			Costs:         []models.CostResourceRest{costResource},
		}

		externalPaymentJourney, responseType, err := mockPaymentService.CreateExternalPaymentJourney(req, &paymentSession)
		So(err, ShouldBeNil)
		So(responseType.String(), ShouldEqual, Success.String())
		So(externalPaymentJourney.NextURL, ShouldEqual, "response_url")
	})

	Convey("Invalid Payment Method", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		req := httptest.NewRequest("", "/test", nil)
		paymentSession := models.PaymentResourceRest{
			PaymentMethod: "invalid",
			Status:        InProgress.String(),
		}

		externalPaymentJourney, responseType, err := mockPaymentService.CreateExternalPaymentJourney(req, &paymentSession)
		So(externalPaymentJourney, ShouldBeNil)
		So(responseType.String(), ShouldEqual, Error.String())
		So(err.Error(), ShouldEqual, "payment method [invalid] for resource [] not recognised")
	})

}
