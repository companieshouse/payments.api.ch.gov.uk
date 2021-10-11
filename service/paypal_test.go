package service

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/golang/mock/gomock"
	"github.com/plutov/paypal/v4"
	. "github.com/smartystreets/goconvey/convey"
)

func CreatePayPalOrderResponse(nextUrl string) paypal.Order {
	return paypal.Order{
		ID:            "123",
		Status:        paypal.OrderStatusCreated,
		Intent:        "",
		Payer:         nil,
		PurchaseUnits: nil,
		Links: []paypal.Link{
			{
				Href:        nextUrl,
				Rel:         "approve",
				Method:      "GET",
				Description: "Approve an order",
				Enctype:     "1",
			},
		},
		CreateTime: nil,
		UpdateTime: nil,
	}
}

func CreateMockPayPalService(sdk PayPalSDK, service PaymentService) PayPalService {
	return PayPalService{
		Client:         sdk,
		PaymentService: service,
	}
}

func TestUnitCreatePaymentAndGenerateNextURL(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	mockDao := dao.NewMockDAO(mockCtrl)
	mockPaymentService := createMockPaymentService(mockDao, cfg)
	mockPayPalSDK := NewMockPayPalSDK(mockCtrl)
	mockPayPalService := CreateMockPayPalService(mockPayPalSDK, mockPaymentService)

	Convey("Error when creating an order resource in PayPal", t, func() {
		req := httptest.NewRequest("", "/test", nil)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
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

		mockPayPalSDK.EXPECT().CreateOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))

		url, resType, err := mockPayPalService.CreatePaymentAndGenerateNextURL(req, &paymentSession)

		So(url, ShouldBeEmpty)
		So(resType, ShouldEqual, Error)
		So(err.Error(), ShouldContainSubstring, "error creating order: [error]")
	})

	Convey("Order status is not created - unsuccessful", t, func() {
		req := httptest.NewRequest("", "/test", nil)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
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

		order := paypal.Order{
			ID:     "123",
			Status: paypal.OrderStatusVoided,
		}

		mockPayPalSDK.EXPECT().CreateOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&order, nil)

		url, resType, err := mockPayPalService.CreatePaymentAndGenerateNextURL(req, &paymentSession)

		So(url, ShouldBeEmpty)
		So(resType, ShouldEqual, Error)
		So(err.Error(), ShouldContainSubstring, "failed to correctly create paypal order")
	})

	Convey("Successfully create paypal order", t, func() {
		req := httptest.NewRequest("", "/test", nil)

		costResource := models.CostResourceRest{
			ClassOfPayment: []string{"penalty"},
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

		order := paypal.Order{
			ID:            "123",
			Status:        paypal.OrderStatusCreated,
			Intent:        "",
			Payer:         nil,
			PurchaseUnits: nil,
			Links: []paypal.Link{
				{
					Href:        "return_url",
					Rel:         "approve",
					Method:      "GET",
					Description: "Approve an order",
					Enctype:     "1",
				},
			},
			CreateTime: nil,
			UpdateTime: nil,
		}

		mockPayPalSDK.EXPECT().CreateOrder(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&order, nil)

		url, resType, err := mockPayPalService.CreatePaymentAndGenerateNextURL(req, &paymentSession)

		So(url, ShouldEqual, "return_url")
		So(resType, ShouldEqual, Success)
		So(err, ShouldBeNil)
	})
}

func TestUnitCaptureOrder(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	mockDao := dao.NewMockDAO(mockCtrl)
	mockPaymentService := createMockPaymentService(mockDao, cfg)
	mockPayPalSDK := NewMockPayPalSDK(mockCtrl)
	mockPayPalService := CreateMockPayPalService(mockPayPalSDK, mockPaymentService)

	Convey("Capture order in paypal", t, func() {
		captureOrder := paypal.CaptureOrderResponse{
			ID:     "123",
			Status: paypal.OrderStatusCompleted,
		}

		mockPayPalSDK.EXPECT().CaptureOrder(gomock.Any(), gomock.Any(), gomock.Any()).Return(&captureOrder, fmt.Errorf("error"))

		res, err := mockPayPalService.CapturePayment("123")

		So(res, ShouldEqual, &captureOrder)
		So(err.Error(), ShouldEqual, "error")
	})
}
