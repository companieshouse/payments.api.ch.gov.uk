package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/jarcoal/httpmock"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

func CreateMockExternalPaymentProvidersService(mockPayPalService PayPalService, mockGovPayService GovPayService) ExternalPaymentProvidersService {
	return ExternalPaymentProvidersService{
		GovPayService: mockGovPayService,
		PayPalService: mockPayPalService,
	}
}

func TestUnitCreateExternalPayment(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	cfg.GovPayURL = "http://dummy-govpay-url"

	mockDao := dao.NewMockDAO(mockCtrl)
	mockPaymentService := createMockPaymentService(mockDao, cfg)
	mockPayPalSDK := NewMockPayPalSDK(mockCtrl)

	// Generate a mock external provider service using mocks for both PayPal and GovPay
	mockExternalPaymentProvidersService := CreateMockExternalPaymentProvidersService(
		PayPalService{
			Client:         mockPayPalSDK,
			PaymentService: mockPaymentService,
		},
		GovPayService{
			PaymentService: mockPaymentService,
		})

	Convey("Payment session not in progress", t, func() {

		path := fmt.Sprintf("/payments/%s", "1234")
		req, err := http.NewRequest("Get", path, nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		So(err, ShouldBeNil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)
		externalPaymentJourney, responseType, err := mockPaymentService.CreateExternalPaymentJourney(req, &models.PaymentResourceRest{}, mockExternalPaymentProvidersService)

		So(externalPaymentJourney, ShouldBeNil)
		So(responseType.String(), ShouldEqual, InvalidData.String())
		So(err.Error(), ShouldEqual, "payment session is not in progress")
	})

	Convey("Class Of Payment different on same cost resource", t, func() {

		req := httptest.NewRequest("", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty", "data-maintenance", "orderable-item"},
			Description:    "mismatched cost resource",
		}

		paymentSession := models.PaymentResourceRest{
			PaymentMethod: "credit-card",
			Amount:        "3",
			Status:        InProgress.String(),
			Costs:         []models.CostResourceRest{costResource},
		}

		externalPaymentJourney, responseType, err := mockPaymentService.CreateExternalPaymentJourney(req, &paymentSession, mockExternalPaymentProvidersService)
		So(externalPaymentJourney, ShouldBeNil)
		So(responseType.String(), ShouldEqual, InvalidData.String())
		So(err.Error(), ShouldEqual, fmt.Sprintf("Two or more class of payments are different on the same cost resource: [%v] ", costResource.Description))
	})

	Convey("Class Of Payment different on different cost resources", t, func() {

		req := httptest.NewRequest("", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		costResource1 := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
			Description:    "penalty cost resource",
		}

		costResource2 := models.CostResourceRest{
			ClassOfPayment: []string{"data-maintenance"},
			Description:    "data-maintenance cost resource",
		}

		paymentSession := models.PaymentResourceRest{
			PaymentMethod: "credit-card",
			Amount:        "3",
			Status:        InProgress.String(),
			Costs:         []models.CostResourceRest{costResource1, costResource2},
		}

		externalPaymentJourney, responseType, err := mockPaymentService.CreateExternalPaymentJourney(req, &paymentSession, mockExternalPaymentProvidersService)
		So(externalPaymentJourney, ShouldBeNil)
		So(responseType.String(), ShouldEqual, InvalidData.String())
		So(err.Error(), ShouldEqual, fmt.Sprintf("Two or more class of payments are different on the same transaction: [%v] and [%v] ",
			costResource1.Description, costResource2.Description))
	})

	Convey("Error communicating with GOV.UK Pay", t, func() {

		req := httptest.NewRequest("", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		paymentSession := models.PaymentResourceRest{
			PaymentMethod: "credit-card",
			Status:        InProgress.String(),
		}

		externalPaymentJourney, responseType, err := mockPaymentService.CreateExternalPaymentJourney(req, &paymentSession, mockExternalPaymentProvidersService)
		So(externalPaymentJourney, ShouldBeNil)
		So(responseType.String(), ShouldEqual, Error.String())
		So(err.Error(), ShouldEqual, `error communicating with GovPay: [error converting amount to pay to pence: [strconv.Atoi: parsing "": invalid syntax]]`)
	})

	Convey("No NextURL received from GOV.UK Pay", t, func() {

		mockDao.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		req := httptest.NewRequest("", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusCreated, &models.IncomingGovPayResponse{})
		httpmock.RegisterResponder("POST", cfg.GovPayURL, jsonResponse)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
		}

		paymentSession := models.PaymentResourceRest{
			PaymentMethod: "credit-card",
			Amount:        "3",
			Status:        InProgress.String(),
			Costs:         []models.CostResourceRest{costResource},
		}

		externalPaymentJourney, responseType, err := mockPaymentService.CreateExternalPaymentJourney(req, &paymentSession, mockExternalPaymentProvidersService)
		So(externalPaymentJourney, ShouldBeNil)
		So(responseType.String(), ShouldEqual, Error.String())
		So(err.Error(), ShouldEqual, "no NextURL returned from GovPay")
	})

	Convey("Create External GovPay Payment Journey - success", t, func() {

		mockDao.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

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
			PaymentMethod: "credit-card",
			Amount:        "4",
			Status:        InProgress.String(),
			Costs:         []models.CostResourceRest{costResource},
		}

		externalPaymentJourney, responseType, err := mockPaymentService.CreateExternalPaymentJourney(req, &paymentSession, mockExternalPaymentProvidersService)
		So(err, ShouldBeNil)
		So(responseType.String(), ShouldEqual, Success.String())
		So(externalPaymentJourney.NextURL, ShouldEqual, "response_url")
	})

	Convey("Create External GovPay Payment Journey for orderable-item - success", t, func() {

		mockDao.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

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
			ClassOfPayment: []string{"orderable-item"},
		}

		paymentSession := models.PaymentResourceRest{
			PaymentMethod: "credit-card",
			Amount:        "3",
			Status:        InProgress.String(),
			Costs:         []models.CostResourceRest{costResource},
		}

		externalPaymentJourney, responseType, err := mockPaymentService.CreateExternalPaymentJourney(req, &paymentSession, mockExternalPaymentProvidersService)
		So(err, ShouldBeNil)
		So(responseType.String(), ShouldEqual, Success.String())
		So(externalPaymentJourney.NextURL, ShouldEqual, "response_url")
	})

	Convey("Error communicating with Paypal", t, func() {

		mockPayPalSDK.EXPECT().CreateOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))

		req := httptest.NewRequest("", "/test", nil)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"orderable-item"},
		}

		paymentSession := models.PaymentResourceRest{
			PaymentMethod: "PayPal",
			Amount:        "3",
			Status:        InProgress.String(),
			Costs:         []models.CostResourceRest{costResource},
			Links: models.PaymentLinksRest{
				Self: "payments/1234",
			},
		}

		externalPaymentJourney, responseType, err := mockPaymentService.CreateExternalPaymentJourney(req, &paymentSession, mockExternalPaymentProvidersService)
		So(externalPaymentJourney, ShouldBeNil)
		So(responseType.String(), ShouldEqual, Error.String())
		So(err.Error(), ShouldEqual, "error communicating with PayPal API: [error creating order: [error]]")

	})

	Convey("No NextURL received from Paypal", t, func() {

		paypalResponse := CreatePayPalOrderResponse("")
		mockPayPalSDK.EXPECT().CreateOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&paypalResponse, nil)
		mockDao.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		req := httptest.NewRequest("", "/test", nil)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"orderable-item"},
		}

		paymentSession := models.PaymentResourceRest{
			PaymentMethod: "PayPal",
			Amount:        "3",
			Status:        InProgress.String(),
			Costs:         []models.CostResourceRest{costResource},
			Links: models.PaymentLinksRest{
				Self: "payments/1234",
			},
		}

		externalPaymentJourney, responseType, err := mockPaymentService.CreateExternalPaymentJourney(req, &paymentSession, mockExternalPaymentProvidersService)
		So(externalPaymentJourney, ShouldBeNil)
		So(responseType.String(), ShouldEqual, Error.String())
		So(err.Error(), ShouldEqual, "approve link not returned in paypal order response")
	})

	Convey("Create an External PayPal Payment Journey for orderable-item - success", t, func() {

		paypalResponse := CreatePayPalOrderResponse("response_url")
		mockPayPalSDK.EXPECT().CreateOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&paypalResponse, nil)
		mockDao.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		req := httptest.NewRequest("", "/test", nil)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"orderable-item"},
		}

		paymentSession := models.PaymentResourceRest{
			PaymentMethod: "PayPal",
			Amount:        "3",
			Status:        InProgress.String(),
			Costs:         []models.CostResourceRest{costResource},
			Links: models.PaymentLinksRest{
				Self: "payments/1234",
			},
		}

		externalPaymentJourney, responseType, err := mockPaymentService.CreateExternalPaymentJourney(req, &paymentSession, mockExternalPaymentProvidersService)
		So(err, ShouldBeNil)
		So(responseType.String(), ShouldEqual, Success.String())
		So(externalPaymentJourney.NextURL, ShouldEqual, "response_url")
	})

	Convey("Invalid Payment Method", t, func() {

		req := httptest.NewRequest("", "/test", nil)

		paymentSession := models.PaymentResourceRest{
			PaymentMethod: "invalid",
			Status:        InProgress.String(),
		}

		externalPaymentJourney, responseType, err := mockPaymentService.CreateExternalPaymentJourney(req, &paymentSession, mockExternalPaymentProvidersService)
		So(externalPaymentJourney, ShouldBeNil)
		So(responseType.String(), ShouldEqual, Error.String())
		So(err.Error(), ShouldEqual, "payment method [invalid] for resource [] not recognised")
	})
}
