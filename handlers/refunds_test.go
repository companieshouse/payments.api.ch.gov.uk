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
	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitHandleCreateRefund(t *testing.T) {

	Convey("Request Body Empty", t, func() {
		req, _ := http.NewRequest("POST", "/payments/123/refunds", nil)
		w := httptest.NewRecorder()
		HandleCreateRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
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

	paymentsPaidStatus := []models.PaymentResourceDB{}
	paymentsPaidStatus = append(paymentsPaidStatus, paymentPaidStatus)

	Convey("Successful request - with payment refunds", t, func() {
		mockDao.EXPECT().GetPaymentsWithRefundPendingStatus().Return(paymentsPaidStatus, nil)

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

func TestUnitHandleProcessPendingRefundsWithResponseTypeSuccess(t *testing.T) {
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

	paymentsPaidStatus := []models.PaymentResourceDB{}
	paymentsPaidStatus = append(paymentsPaidStatus, paymentPaidStatus)

	Convey("Successful request - with response type success", t, func() {
		mockDao.EXPECT().GetPaymentsWithRefundPendingStatus().Return(paymentsPaidStatus, nil)

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

		res, resType, _ := refundService.ProcessPendingRefunds(req)

		So(resType, ShouldEqual, service.Success)
		So(len(res), ShouldEqual, 1)
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

	paymentsPaidStatus := []models.PaymentResourceDB{}
	paymentsPaidStatus = append(paymentsPaidStatus, paymentPaidStatus)

	Convey("Failed request - with response type Error", t, func() {
		errorReponse := errors.New("error retrieving payments with pending refunds status")
		errorList = append(errorList, errorReponse)

		mockDao.EXPECT().GetPaymentsWithRefundPendingStatus().Return(nil, errorReponse)

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

		So(respondedError[0].Error(), ShouldEqual, "no payments with paid status found")
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

	paymentsPaidStatus := []models.PaymentResourceDB{}
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

		So(respondedError[0].Error(), ShouldEqual, "no payments with paid status found")
		So(resType, ShouldEqual, service.Success)
		So(len(res), ShouldEqual, 0)

	})

}
