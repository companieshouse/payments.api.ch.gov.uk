package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitHandleCreateRefund(t *testing.T) {

	Convey("Request Body Empty", t, func() {
		req, _ := http.NewRequest("POST", "/payments/123/refunds", nil)
		w := httptest.NewRecorder()
		HandleCreateRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Payment ID not supplied", t, func() {
		req, _ := http.NewRequest("POST", "/payments/123/refunds", strings.NewReader("Body"))
		w := httptest.NewRecorder()
		HandleCreateRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Invalid request body", t, func() {
		req, _ := http.NewRequest("POST", "/payments/123/refunds", strings.NewReader("invalid_body"))
		req = mux.SetURLVars(req, map[string]string{"paymentId": "abc"})
		w := httptest.NewRecorder()
		HandleCreateRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})
}

func TestUnitGetPaymentRefunds(t *testing.T) {
	cfg, _ := config.Get()
	mockDao := dao.NewMockDAO(gomock.NewController(t))
	Convey("Request Body Empty", t, func() {
		req, _ := http.NewRequest("GET", "/payments/123/refunds", nil)
		w := httptest.NewRecorder()
		HandleGetRefunds(w, req)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Payment ID not supplied", t, func() {
		req, _ := http.NewRequest("GET", "/payments/123/refunds", nil)
		vars := map[string]string{"paymentId": ""}
		w := httptest.NewRecorder()

		req = mux.SetURLVars(req, vars)

		HandleGetRefunds(w, req)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Successful request - with the paymentId for refunds", t, func() {
		refundData := models.RefundResourceDB{
			RefundId:          "sasaswewq23wsw",
			CreatedAt:         "2020-11-19T12:57:30.Z06Z",
			Amount:            800.0,
			Status:            "pending",
			ExternalRefundUrl: "https://pulicapi.payments.service.gov.uk",
		}
		refundDatas := []models.RefundResourceDB{refundData}

		paymentService = &service.PaymentService{
			DAO:    mockDao,
			Config: *cfg,
		}

		refundService = &service.RefundService{
			PaymentService: paymentService,
			DAO:            mockDao,
			Config:         *cfg,
		}

		req, _ := http.NewRequest("GET", "/payments/123/refunds", nil)
		vars := map[string]string{"paymentId": "123"}
		w := httptest.NewRecorder()

		req = mux.SetURLVars(req, vars)

		mockDao.EXPECT().GetPaymentRefunds(gomock.Any()).Return(refundDatas, nil)

		HandleGetRefunds(w, req)
		So(w.Code, ShouldEqual, http.StatusOK)
	})
}

func TestUnitHandleProcessPendingRefundsWithPaymentRefunds(t *testing.T) {
	cfg, _ := config.Get()
	mockDao := dao.NewMockDAO(gomock.NewController(t))
	paymentResourceDataDB := models.PaymentResourceDataDB{}
	refundData := models.RefundResourceDB{
		RefundId:          "sasaswewq23wsw",
		CreatedAt:         "2020-11-19T12:57:30.Z06Z",
		Amount:            800.0,
		Status:            "pending",
		ExternalRefundUrl: "https://pulicapi.payments.service.gov.uk",
	}
	refundDatas := []models.RefundResourceDB{refundData}
	paymentPaidStatus := models.PaymentResourceDB{
		ID:                           "xVzfvN3TlKWAAPp",
		RedirectURI:                  "https://www.google.com",
		State:                        "application-nonce-value",
		ExternalPaymentStatusURI:     "https://publicapi.payments/5212tt8usgl6k574f6",
		ExternalPaymentStatusID:      "tPM43f6ck5212tt8usgl6k574f6",
		ExternalPaymentTransactionID: "",
		Data:                         paymentResourceDataDB,
		Refunds:                      refundDatas,
		BulkRefund:                   nil,
	}

	var paymentsPaidStatus []models.PaymentResourceDB
	paymentsPaidStatus = append(paymentsPaidStatus, paymentPaidStatus)

	Convey("Successful request - with payment refunds", t, func() {
		mockDao.EXPECT().GetPaymentsWithRefundPendingStatus().Return(nil, nil)

		paymentService = &service.PaymentService{
			DAO:    mockDao,
			Config: *cfg,
		}

		refundService = &service.RefundService{
			PaymentService: paymentService,
			DAO:            mockDao,
			Config:         *cfg,
		}

		req, _ := http.NewRequest("POST", "/payments/refunds/process-pending", strings.NewReader("Body"))
		w := httptest.NewRecorder()

		HandleProcessPendingRefunds(w, req)

		So(w.Code, ShouldEqual, http.StatusOK)
	})
}

func TestUnitHandleProcessPendingRefundsWithResponseTypeError(t *testing.T) {
	var errorList []error
	cfg, _ := config.Get()
	mockDao := dao.NewMockDAO(gomock.NewController(t))

	paymentResourceDataDB := models.PaymentResourceDataDB{}
	refundData := models.RefundResourceDB{
		RefundId:          "sasaswewq23wsw",
		CreatedAt:         "2020-11-19T12:57:30.Z06Z",
		Amount:            800.0,
		Status:            "pending",
		ExternalRefundUrl: "https://pulicapi.payments.service.gov.uk",
	}
	refundDatas := []models.RefundResourceDB{refundData}
	paymentPaidStatus := models.PaymentResourceDB{
		ID:                           "xVzfvN3TlKWAAPp",
		RedirectURI:                  "https://www.google.com",
		State:                        "application-nonce-value",
		ExternalPaymentStatusURI:     "https://publicapi.payments/5212tt8usgl6k574f6",
		ExternalPaymentStatusID:      "tPM43f6ck5212tt8usgl6k574f6",
		ExternalPaymentTransactionID: "",
		Data:                         paymentResourceDataDB,
		Refunds:                      refundDatas,
		BulkRefund:                   nil,
	}

	var paymentsPaidStatus []models.PaymentResourceDB
	paymentsPaidStatus = append(paymentsPaidStatus, paymentPaidStatus)

	Convey("Failed request - with response type Error", t, func() {
		errorResponse := errors.New("error retrieving payments with pending refunds status")
		errorList = append(errorList, errorResponse)

		mockDao.EXPECT().GetPaymentsWithRefundPendingStatus().Return(nil, errorResponse)

		paymentService = &service.PaymentService{
			DAO:    mockDao,
			Config: *cfg,
		}

		govPayService := &service.GovPayService{PaymentService: *paymentService}

		refundService = &service.RefundService{
			GovPayService:  govPayService,
			PaymentService: paymentService,
			DAO:            mockDao,
			Config:         *cfg,
		}

		req, _ := http.NewRequest("POST", "/payments/refunds/process-pending", strings.NewReader("Body"))
		httptest.NewRecorder()

		res, resType, respondedError := refundService.ProcessPendingRefunds(req)

		So(respondedError[0].Error(), ShouldEqual, "error retrieving payments with refund pending status")
		So(resType, ShouldEqual, service.Error)
		So(len(res), ShouldEqual, 0)
	})

	Convey("Successful request - with no payment response", t, func() {
		mockDao.EXPECT().GetPaymentsWithRefundPendingStatus().Return(nil, nil)

		paymentService = &service.PaymentService{
			DAO:    mockDao,
			Config: *cfg,
		}

		refundService = &service.RefundService{
			PaymentService: paymentService,
			DAO:            mockDao,
			Config:         *cfg,
		}

		req, _ := http.NewRequest("POST", "/payments/refunds/process-pending", strings.NewReader("Body"))
		httptest.NewRecorder()

		res, resType, respondedError := refundService.ProcessPendingRefunds(req)

		So(respondedError[0].Error(), ShouldEqual, "no payments with refund pending status found")
		So(resType, ShouldEqual, service.Success)
		So(len(res), ShouldEqual, 0)
	})
}

func TestUnitHandleProcessPendingRefundsWithNoPayment(t *testing.T) {
	cfg, _ := config.Get()
	mockDao := dao.NewMockDAO(gomock.NewController(t))
	paymentResourceDataDB := models.PaymentResourceDataDB{}
	refundData := models.RefundResourceDB{
		RefundId:          "sasaswewq23wsw",
		CreatedAt:         "2020-11-19T12:57:30.Z06Z",
		Amount:            800.0,
		Status:            "pending",
		ExternalRefundUrl: "https://pulicapi.payments.service.gov.uk",
	}
	refundDatas := []models.RefundResourceDB{refundData}
	paymentPaidStatus := models.PaymentResourceDB{
		ID:                           "xVzfvN3TlKWAAPp",
		RedirectURI:                  "https://www.google.com",
		State:                        "application-nonce-value",
		ExternalPaymentStatusURI:     "https://publicapi.payments/5212tt8usgl6k574f6",
		ExternalPaymentStatusID:      "tPM43f6ck5212tt8usgl6k574f6",
		ExternalPaymentTransactionID: "",
		Data:                         paymentResourceDataDB,
		Refunds:                      refundDatas,
		BulkRefund:                   nil,
	}

	var paymentsPaidStatus []models.PaymentResourceDB
	paymentsPaidStatus = append(paymentsPaidStatus, paymentPaidStatus)

	Convey("Successful request - with no payment response", t, func() {
		mockDao.EXPECT().GetPaymentsWithRefundPendingStatus().Return(nil, nil)

		paymentService = &service.PaymentService{
			DAO:    mockDao,
			Config: *cfg,
		}

		refundService = &service.RefundService{
			PaymentService: paymentService,
			DAO:            mockDao,
			Config:         *cfg,
		}

		req, _ := http.NewRequest("POST", "/payments/refunds/process-pending", strings.NewReader("Body"))
		httptest.NewRecorder()

		res, resType, respondedError := refundService.ProcessPendingRefunds(req)

		So(respondedError[0].Error(), ShouldEqual, "no payments with refund pending status found")
		So(resType, ShouldEqual, service.Success)
		So(len(res), ShouldEqual, 0)
	})
}
