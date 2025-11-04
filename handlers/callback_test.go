package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/plutov/paypal/v4"

	"github.com/companieshouse/chs.go/avro"
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
	AvailablePaymentMethods: []string{"credit-card"},
	ClassOfPayment:          []string{"data-maintenance"},
	Description:             "desc",
	DescriptionIdentifier:   "identifier",
	ProductType:             "productType",
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

type CustomError struct {
	message string
}

func (e CustomError) Error() string {
	return e.message
}

// Mock function for erroring when preparing and sending kafka message
func mockProduceKafkaMessageError(path string) error {
	return CustomError{"hello"}
}

// Mock function for successful preparing and sending of kafka message
func mockProduceKafkaMessage(path string) error {
	return nil
}

func TestUnitHandleGovPayCallback(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockPaymentProvidersService := service.NewMockPaymentProviderService(mockCtrl)

	cfg, _ := config.Get()
	cfg.DomainAllowList = "http://dummy-url"

	Convey("Payment ID not supplied", t, func() {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler := HandleGovPayCallback(mockPaymentProvidersService)
		handler.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Error getting payment session", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		paymentService = createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(nil, fmt.Errorf("error"))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		w := httptest.NewRecorder()

		handler := HandleGovPayCallback(mockPaymentProvidersService)
		handler.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Payment session not found", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		paymentService = createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		w := httptest.NewRecorder()

		handler := HandleGovPayCallback(mockPaymentProvidersService)
		handler.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusNotFound)
	})

	Convey("Payment session is already paid", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		paymentService = createMockPaymentService(mock, cfg)
		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount: "10.00",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				Status: service.Paid.String(),
			},
		}
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		w := httptest.NewRecorder()

		handler := HandleGovPayCallback(mockPaymentProvidersService)
		handler.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Invalid expiry time", t, func() {
		cfg.ExpiryTimeInMinutes = "invalid"

		mock := dao.NewMockDAO(mockCtrl)
		paymentService = createMockPaymentService(mock, cfg)
		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount: "10.00",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt: time.Now().Add(time.Hour * -2),
			},
		}
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		w := httptest.NewRecorder()

		handler := HandleGovPayCallback(mockPaymentProvidersService)
		handler.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	cfg.ExpiryTimeInMinutes = "60"
	Convey("Payment session expired", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		paymentService = createMockPaymentService(mock, cfg)
		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount: "10.00",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt: time.Now().Add(time.Hour * -2),
			},
		}
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mock.EXPECT().PatchPaymentResource("1234", gomock.Any()).Return(nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		w := httptest.NewRecorder()

		handler := HandleGovPayCallback(mockPaymentProvidersService)
		handler.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusForbidden)
	})

	Convey("Payment session expired and patch failed", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		paymentService = createMockPaymentService(mock, cfg)
		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount: "10.00",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt: time.Now().Add(time.Hour * -2),
			},
		}
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mock.EXPECT().PatchPaymentResource("1234", gomock.Any()).Return(fmt.Errorf("error"))

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		w := httptest.NewRecorder()

		handler := HandleGovPayCallback(mockPaymentProvidersService)
		handler.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
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
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		w := httptest.NewRecorder()

		handler := HandleGovPayCallback(mockPaymentProvidersService)
		handler.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusPreconditionFailed)
	})

	Convey("Error getting payment status from credit-card", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		paymentService = createMockPaymentService(mock, cfg)
		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount:        "10.00",
				PaymentMethod: "credit-card",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt: time.Now(),
			},
		}
		statusResponse := models.StatusResponse{
			Status: paypal.OrderStatusVoided,
		}
		mockPaymentProvidersService.EXPECT().CheckPaymentProviderStatus(gomock.Any()).Return(&statusResponse, "123", service.Success, nil)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		w := httptest.NewRecorder()

		handler := HandleGovPayCallback(mockPaymentProvidersService)
		handler.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Error setting payment status", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		paymentService = createMockPaymentService(mock, cfg)
		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount:        "10.00",
				PaymentMethod: "credit-card",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt: time.Now(),
			},
		}
		statusResponse := models.StatusResponse{
			Status: paypal.OrderStatusVoided,
		}
		mockPaymentProvidersService.EXPECT().CheckPaymentProviderStatus(gomock.Any()).Return(&statusResponse, "123", service.Success, nil)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		govPayResponse := models.IncomingGovPayResponse{}
		govPayJSONResponse, _ := httpmock.NewJsonResponder(http.StatusOK, govPayResponse)
		httpmock.RegisterResponder(http.MethodGet, cfg.GovPayURL, govPayJSONResponse)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		w := httptest.NewRecorder()

		handler := HandleGovPayCallback(mockPaymentProvidersService)
		handler.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Error sending kafka message", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		paymentService = createMockPaymentService(mock, cfg)
		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount:        "10.00",
				PaymentMethod: "credit-card",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt: time.Now(),
			},
		}
		statusResponse := models.StatusResponse{
			Status: paypal.OrderStatusVoided,
		}
		mockPaymentProvidersService.EXPECT().CheckPaymentProviderStatus(gomock.Any()).Return(&statusResponse, "123", service.Success, nil)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jSONResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jSONResponse)

		govPayResponse := models.IncomingGovPayResponse{}
		govPayJSONResponse, _ := httpmock.NewJsonResponder(http.StatusOK, govPayResponse)
		httpmock.RegisterResponder(http.MethodGet, cfg.GovPayURL, govPayJSONResponse)

		handlePaymentMessage = mockProduceKafkaMessageError

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		w := httptest.NewRecorder()

		handler := HandleGovPayCallback(mockPaymentProvidersService)
		handler.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Successful callback with redirect", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		paymentService = createMockPaymentService(mock, cfg)
		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount:        "10.00",
				PaymentMethod: "credit-card",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt:   time.Now(),
				CompletedAt: time.Now(),
			},
			ExternalPaymentStatusURI: "http://dummy-url",
		}
		statusResponse := models.StatusResponse{
			Status: paypal.OrderStatusVoided,
		}
		mockPaymentProvidersService.EXPECT().CheckPaymentProviderStatus(gomock.Any()).Return(&statusResponse, "123", service.Success, nil)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jSONResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jSONResponse)

		govPayJSONResponse, _ := httpmock.NewJsonResponder(
			http.StatusOK,
			models.IncomingGovPayResponse{
				State: models.State{
					Finished: true,
					Status:   "success",
				},
			},
		)

		httpmock.RegisterResponder(http.MethodGet, cfg.GovPayURL, govPayJSONResponse)

		handlePaymentMessage = mockProduceKafkaMessage

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		w := httptest.NewRecorder()

		handler := HandleGovPayCallback(mockPaymentProvidersService)
		handler.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusSeeOther)
		So(paymentSession.Data.CompletedAt, ShouldNotBeZeroValue)
	})

	Convey("Created callback with redirect", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		paymentService = createMockPaymentService(mock, cfg)
		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount:        "10.00",
				PaymentMethod: "credit-card",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt: time.Now(),
			},
			ExternalPaymentStatusURI: "http://dummy-url",
		}
		statusResponse := models.StatusResponse{
			Status: "in-progress",
		}
		mockPaymentProvidersService.EXPECT().CheckPaymentProviderStatus(gomock.Any()).Return(&statusResponse, "123", service.Created, nil)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jSONResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jSONResponse)

		govPayJSONResponse, _ := httpmock.NewJsonResponder(
			http.StatusOK,
			models.IncomingGovPayResponse{
				State: models.State{
					Finished: false,
					Status:   "created",
				},
			},
		)

		httpmock.RegisterResponder(http.MethodGet, cfg.GovPayURL, govPayJSONResponse)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		w := httptest.NewRecorder()

		handler := HandleGovPayCallback(mockPaymentProvidersService)
		handler.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusSeeOther)
		So(paymentSession.Data.CompletedAt, ShouldBeZeroValue)
	})

	Convey("Successful message preparation with prepareKafkaMessage", t, func() {
		paymentID := "12345"
		refundID := "54321"

		// This is the schema that is used by the producer
		schema := `{
				"type": "record",
				"name": "payment_processed",
				"namespace": "payments",
				"fields": [
				{
					"name": "payment_resource_id",
					"type": "string"
				},
				{
					"name": "refund_id",
					"type": "string"
				}
				]
			}`

		producerSchema := &avro.Schema{
			Definition: schema,
		}
		// Here we test that after preparing the message, the message represents the original message. We provide
		// the schema and message (paymentID), prepare the message (which includes marshalling), then unmarshal to
		// ensure the data being sent to the payments-processed topic has not been modified in any way

		message, pkmError := prepareKafkaMessage(paymentID, refundID, *producerSchema)
		unmarshalledPaymentProcessed := paymentProcessed{}
		psError := producerSchema.Unmarshal(message.Value, &unmarshalledPaymentProcessed)

		So(pkmError, ShouldEqual, nil)
		So(psError, ShouldEqual, nil)
		So(unmarshalledPaymentProcessed.PaymentSessionID, ShouldEqual, "12345")
		So(unmarshalledPaymentProcessed.RefundId, ShouldEqual, "54321")
	})

	Convey("Unsuccessful message preparation with prepareKafkaMessage", t, func() {
		paymentID := "12345"
		refundID := "54321"

		// This is the schema that is used by the producer, the type is in the incorrect type, so should error when marshalling
		schema := `{
				"type": "record",
				"name": "payment_processed",
				"namespace": "payments",
				"fields": [
				{
					"name": "payment_resource_id",
					"type": "int"
				},
	{
					"name": "refund_id",
					"type": "int"
				}
				]
			}`

		producerSchema := &avro.Schema{
			Definition: schema,
		}

		_, err := prepareKafkaMessage(paymentID, refundID, *producerSchema)
		So(err, ShouldNotBeEmpty)
	})
}

func serveHandlePayPalCallback(externalPaymentSvc service.PaymentProviderService, paymentIDSet bool) *httptest.ResponseRecorder {
	path := "/callback/payments/paypal/orders/1234"
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if paymentIDSet {
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
	}
	//if orderIDSet {
	//	params := req.URL.Query()
	//	params.Set("token", "5678")
	//	req.URL.RawQuery = params.Encode()
	//}
	res := httptest.NewRecorder()

	handler := HandlePayPalCallback(externalPaymentSvc)
	handler.ServeHTTP(res, req)

	return res
}

func TestUnitHandlePayPalCallback(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Generate a mock external provider service using mocks for both PayPal and GovPay
	mockExternalPaymentProvidersService := service.NewMockPaymentProviderService(mockCtrl)

	Convey("Payment ID not supplied", t, func() {
		res := serveHandlePayPalCallback(mockExternalPaymentProvidersService, false) //
		So(res.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Error getting payment session", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		cfg, _ := config.Get()
		paymentService = createMockPaymentService(mock, cfg)

		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(nil, fmt.Errorf("error"))

		res := serveHandlePayPalCallback(mockExternalPaymentProvidersService, true)
		So(res.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Payment session not found", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		cfg, _ := config.Get()
		paymentService = createMockPaymentService(mock, cfg)

		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(nil, nil)

		res := serveHandlePayPalCallback(mockExternalPaymentProvidersService, true)
		So(res.Code, ShouldEqual, http.StatusNotFound)
	})

	Convey("Payment session is already paid", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		cfg, _ := config.Get()
		paymentService = createMockPaymentService(mock, cfg)
		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount: "10.00",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				Status: service.Paid.String(),
			},
		}
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		res := serveHandlePayPalCallback(mockExternalPaymentProvidersService, true)
		So(res.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Invalid expiry time", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		cfg, _ := config.Get()
		cfg.ExpiryTimeInMinutes = "invalid"
		paymentService = createMockPaymentService(mock, cfg)

		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount: "10.00",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt: time.Now().Add(time.Hour * -2),
			},
		}

		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		res := serveHandlePayPalCallback(mockExternalPaymentProvidersService, true)
		So(res.Code, ShouldEqual, http.StatusInternalServerError)

	})

	Convey("Error setting payment status of expired payment session", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		cfg, _ := config.Get()
		cfg.ExpiryTimeInMinutes = "60"
		paymentService = createMockPaymentService(mock, cfg)

		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount: "10.00",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt: time.Now().Add(time.Hour * -2),
			},
		}

		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(fmt.Errorf("error"))

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		res := serveHandlePayPalCallback(mockExternalPaymentProvidersService, true)
		So(res.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Payment session is expired", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		cfg, _ := config.Get()
		cfg.ExpiryTimeInMinutes = "60"
		paymentService = createMockPaymentService(mock, cfg)

		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount: "10.00",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt: time.Now().Add(time.Hour * -2),
			},
		}

		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		res := serveHandlePayPalCallback(mockExternalPaymentProvidersService, true)
		So(res.Code, ShouldEqual, http.StatusForbidden)
	})

	Convey("Payment method not recognised", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		cfg, _ := config.Get()
		cfg.ExpiryTimeInMinutes = "60"
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
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		res := serveHandlePayPalCallback(mockExternalPaymentProvidersService, true)
		So(res.Code, ShouldEqual, http.StatusPreconditionFailed)
	})

	Convey("Error checking paypal order status", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		cfg, _ := config.Get()
		cfg.ExpiryTimeInMinutes = "60"
		paymentService = createMockPaymentService(mock, cfg)

		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount:        "10.00",
				PaymentMethod: "paypal",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt: time.Now(),
			},
		}

		statusResponse := models.StatusResponse{
			Status: paypal.OrderStatusVoided,
		}

		mockExternalPaymentProvidersService.EXPECT().CheckPaymentProviderStatus(gomock.Any()).Return(&statusResponse, "123", service.Success, fmt.Errorf("error"))
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		res := serveHandlePayPalCallback(mockExternalPaymentProvidersService, true)
		So(res.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Error - paypal payment status not approved", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		cfg, _ := config.Get()
		cfg.ExpiryTimeInMinutes = "60"
		paymentService = createMockPaymentService(mock, cfg)

		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount:        "10.00",
				PaymentMethod: "paypal",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt: time.Now(),
			},
		}

		statusResponse := models.StatusResponse{
			Status: paypal.OrderStatusVoided,
		}

		mockExternalPaymentProvidersService.EXPECT().CheckPaymentProviderStatus(gomock.Any()).Return(&statusResponse, "", service.Success, nil)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		res := serveHandlePayPalCallback(mockExternalPaymentProvidersService, true)
		So(res.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Error capturing payment", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		cfg, _ := config.Get()
		cfg.ExpiryTimeInMinutes = "60"
		paymentService = createMockPaymentService(mock, cfg)

		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount:        "10.00",
				PaymentMethod: "paypal",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt: time.Now(),
			},
		}

		statusResponse := models.StatusResponse{
			Status: paypal.OrderStatusApproved,
		}

		mockExternalPaymentProvidersService.EXPECT().CheckPaymentProviderStatus(gomock.Any()).Return(&statusResponse, "", service.Success, nil)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mockExternalPaymentProvidersService.EXPECT().CapturePayment(gomock.Any()).Return(nil, fmt.Errorf("error"))

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		res := serveHandlePayPalCallback(mockExternalPaymentProvidersService, true)
		So(res.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Error setting successful payment status", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		cfg, _ := config.Get()
		cfg.ExpiryTimeInMinutes = "60"
		paymentService = createMockPaymentService(mock, cfg)

		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount:        "10.00",
				PaymentMethod: "paypal",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt: time.Now(),
			},
		}

		statusResponse := models.StatusResponse{
			Status: paypal.OrderStatusApproved,
		}

		captureResponse := paypal.CaptureOrderResponse{
			PurchaseUnits: []paypal.CapturedPurchaseUnit{
				{
					Payments: &paypal.CapturedPayments{
						Captures: []paypal.CaptureAmount{
							{
								Status: "COMPLETED",
							},
						},
					},
				},
			},
		}

		mockExternalPaymentProvidersService.EXPECT().CheckPaymentProviderStatus(gomock.Any()).Return(&statusResponse, "", service.Success, nil)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mockExternalPaymentProvidersService.EXPECT().CapturePayment(gomock.Any()).Return(&captureResponse, nil)
		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(fmt.Errorf("error"))

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		res := serveHandlePayPalCallback(mockExternalPaymentProvidersService, true)
		So(res.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Error sending kafka message", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		cfg, _ := config.Get()
		cfg.ExpiryTimeInMinutes = "60"
		paymentService = createMockPaymentService(mock, cfg)

		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount:        "10.00",
				PaymentMethod: "paypal",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt: time.Now(),
			},
		}

		statusResponse := models.StatusResponse{
			Status: paypal.OrderStatusApproved,
		}

		captureResponse := paypal.CaptureOrderResponse{
			PurchaseUnits: []paypal.CapturedPurchaseUnit{
				{
					Payments: &paypal.CapturedPayments{
						Captures: []paypal.CaptureAmount{
							{
								Status: "COMPLETED",
							},
						},
					},
				},
			},
		}

		mockExternalPaymentProvidersService.EXPECT().CheckPaymentProviderStatus(gomock.Any()).Return(&statusResponse, "", service.Success, nil)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mockExternalPaymentProvidersService.EXPECT().CapturePayment(gomock.Any()).Return(&captureResponse, nil)
		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		handlePaymentMessage = mockProduceKafkaMessageError

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		res := serveHandlePayPalCallback(mockExternalPaymentProvidersService, true)
		So(res.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Successful redirect if payment is cancelled", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		cfg, _ := config.Get()
		cfg.ExpiryTimeInMinutes = "60"
		paymentService = createMockPaymentService(mock, cfg)

		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount:        "10.00",
				PaymentMethod: "paypal",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt:   time.Now(),
				CompletedAt: time.Now(),
			},
		}

		statusResponse := models.StatusResponse{
			Status: paypal.OrderStatusCreated,
		}

		mockExternalPaymentProvidersService.EXPECT().CheckPaymentProviderStatus(gomock.Any()).Return(&statusResponse, "", service.Success, nil)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		handlePaymentMessage = mockProduceKafkaMessage

		res := serveHandlePayPalCallback(mockExternalPaymentProvidersService, true)
		So(res.Code, ShouldEqual, http.StatusSeeOther)
	})

	Convey("Successful PayPal callback with redirect - paypal payment declined", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		cfg, _ := config.Get()
		cfg.ExpiryTimeInMinutes = "60"
		paymentService = createMockPaymentService(mock, cfg)

		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount:        "10.00",
				PaymentMethod: "paypal",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt:   time.Now(),
				CompletedAt: time.Now(),
			},
		}

		statusResponse := models.StatusResponse{
			Status: paypal.OrderStatusApproved,
		}

		captureResponse := paypal.CaptureOrderResponse{
			PurchaseUnits: []paypal.CapturedPurchaseUnit{
				{
					Payments: &paypal.CapturedPayments{
						Captures: []paypal.CaptureAmount{
							{
								Status: "DECLINED",
							},
						},
					},
				},
			},
		}

		mockExternalPaymentProvidersService.EXPECT().CheckPaymentProviderStatus(gomock.Any()).Return(&statusResponse, "", service.Success, nil)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mockExternalPaymentProvidersService.EXPECT().CapturePayment(gomock.Any()).Return(&captureResponse, nil)
		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		handlePaymentMessage = mockProduceKafkaMessage

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		res := serveHandlePayPalCallback(mockExternalPaymentProvidersService, true)
		So(res.Code, ShouldEqual, http.StatusSeeOther)
		So(paymentSession.Data.CompletedAt, ShouldNotBeZeroValue)
	})

	Convey("Successful PayPal callback with redirect - paypal payment failed", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		cfg, _ := config.Get()
		cfg.ExpiryTimeInMinutes = "60"
		paymentService = createMockPaymentService(mock, cfg)

		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount:        "10.00",
				PaymentMethod: "paypal",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt:   time.Now(),
				CompletedAt: time.Now(),
			},
		}

		statusResponse := models.StatusResponse{
			Status: paypal.OrderStatusApproved,
		}

		captureResponse := paypal.CaptureOrderResponse{
			PurchaseUnits: []paypal.CapturedPurchaseUnit{
				{
					Payments: &paypal.CapturedPayments{
						Captures: []paypal.CaptureAmount{
							{
								Status: "FAILED",
							},
						},
					},
				},
			},
		}

		mockExternalPaymentProvidersService.EXPECT().CheckPaymentProviderStatus(gomock.Any()).Return(&statusResponse, "", service.Success, nil)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mockExternalPaymentProvidersService.EXPECT().CapturePayment(gomock.Any()).Return(&captureResponse, nil)
		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		handlePaymentMessage = mockProduceKafkaMessage

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		res := serveHandlePayPalCallback(mockExternalPaymentProvidersService, true)
		So(res.Code, ShouldEqual, http.StatusSeeOther)
		So(paymentSession.Data.CompletedAt, ShouldNotBeZeroValue)
	})

	Convey("Successful PayPal callback with redirect", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		cfg, _ := config.Get()
		cfg.ExpiryTimeInMinutes = "60"
		paymentService = createMockPaymentService(mock, cfg)

		paymentSession := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount:        "10.00",
				PaymentMethod: "paypal",
				Links: models.PaymentLinksDB{
					Resource: "http://dummy-url",
				},
				CreatedAt:   time.Now(),
				CompletedAt: time.Now(),
			},
		}

		statusResponse := models.StatusResponse{
			Status: paypal.OrderStatusApproved,
		}

		captureResponse := paypal.CaptureOrderResponse{
			PurchaseUnits: []paypal.CapturedPurchaseUnit{
				{
					Payments: &paypal.CapturedPayments{
						Captures: []paypal.CaptureAmount{
							{
								Status: "COMPLETED",
							},
						},
					},
				},
			},
		}

		mockExternalPaymentProvidersService.EXPECT().CheckPaymentProviderStatus(gomock.Any()).Return(&statusResponse, "", service.Success, nil)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mockExternalPaymentProvidersService.EXPECT().CapturePayment(gomock.Any()).Return(&captureResponse, nil)
		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		handlePaymentMessage = mockProduceKafkaMessage

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(http.StatusOK, defaultCosts)
		httpmock.RegisterResponder(http.MethodGet, "http://dummy-url", jsonResponse)

		res := serveHandlePayPalCallback(mockExternalPaymentProvidersService, true)
		So(res.Code, ShouldEqual, http.StatusSeeOther)
		So(paymentSession.Data.CompletedAt, ShouldNotBeZeroValue)
	})
}
