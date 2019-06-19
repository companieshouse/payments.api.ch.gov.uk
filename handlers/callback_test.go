package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	AvailablePaymentMethods: []string{"GovPay"},
	ClassOfPayment:          []string{"class"},
	Description:             "desc",
	DescriptionIdentifier:   "identifier",
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
func mockProduceKafaMessageError(path string) error {
	return CustomError{"hello"}
}

// Mock function for successful preparing and sending of kafka message
func mockProduceKafaMessage(path string) error {
	return nil
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
		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-url", jsonResponse)

		req := httptest.NewRequest("GET", "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		w := httptest.NewRecorder()
		HandleGovPayCallback(w, req)
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
		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-url", jsonResponse)

		req := httptest.NewRequest("GET", "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "1234"})
		w := httptest.NewRecorder()
		HandleGovPayCallback(w, req)
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

	Convey("Error sending kafka message", t, func() {
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

		handleKafkaMessage = mockProduceKafaMessageError

		req := httptest.NewRequest("GET", "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		w := httptest.NewRecorder()
		HandleGovPayCallback(w, req)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Successful callback with redirect", t, func() {
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

		handleKafkaMessage = mockProduceKafaMessage

		req := httptest.NewRequest("GET", "/test", nil)
		req = mux.SetURLVars(req, map[string]string{"payment_id": "123"})
		w := httptest.NewRecorder()
		HandleGovPayCallback(w, req)
		So(w.Code, ShouldEqual, http.StatusSeeOther)
		So(paymentSession.Data.CompletedAt, ShouldNotBeNil)
	})

	Convey("Successful message preparation with prepareKafkaMessage", t, func() {
		paymentID := "12345"

		// This is the schema that is used by the producer
		schema := `{
			"type": "record",
			"name": "payment_processed",
			"namespace": "payments",
			"fields": [
			{
				"name": "payment_resource_id",
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

		message, pkmError := prepareKafkaMessage(paymentID, *producerSchema)
		unmarshalledPaymentProcessed := paymentProcessed{}
		psError := producerSchema.Unmarshal(message.Value, &unmarshalledPaymentProcessed)

		So(pkmError, ShouldEqual, nil)
		So(psError, ShouldEqual, nil)
		So(unmarshalledPaymentProcessed.PaymentSessionID, ShouldEqual, "12345")
	})

	Convey("Unsuccessful message preparation with prepareKafkaMessage", t, func() {
		paymentID := "12345"

		// This is the schema that is used by the producer, the type is in the incorrect type, so should error when marshalling
		schema := `{
			"type": "record",
			"name": "payment_processed",
			"namespace": "payments",
			"fields": [
			{
				"name": "payment_resource_id",
				"type": "int"
			}
			]
		}`

		producerSchema := &avro.Schema{
			Definition: schema,
		}

		_, err := prepareKafkaMessage(paymentID, *producerSchema)
		So(err, ShouldNotBeEmpty)
	})
}
