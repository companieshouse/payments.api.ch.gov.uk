package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/fixtures"
	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	xmlFilePath        = "test_files/bulk_refund.xml"
	errorXmlPath       = "test_files/bulk_refund_error.xml"
	invalidXmlPath     = "test_files/bulk_refund_invalid.xml"
	invalidDataXmlPath = "test_files/bulk_refund_invalid_data.xml"
)

func getBodyWithFile(filePath string) (*bytes.Buffer, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	defer file.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.SetBoundary("test_boundary")
	part, err := writer.CreateFormFile("file", filePath)
	if err != nil {
		writer.Close()
		return nil, err
	}
	io.Copy(part, file)
	writer.Close()
	return body, nil
}

func TestUnitHandleBulkRefund(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	Convey("File not supplied", t, func() {
		req := httptest.NewRequest("POST", "/admin/payments/bulk-refunds/govpay", nil)
		w := httptest.NewRecorder()
		HandleGovPayBulkRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Error bulk refund file", t, func() {
		body, err := getBodyWithFile(errorXmlPath)
		if err != nil {
			t.Error(err)
		}

		req := httptest.NewRequest("POST", "/admin/payments/bulk-refunds/govpay", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=test_boundary")
		w := httptest.NewRecorder()

		HandleGovPayBulkRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Invalid bulk refund file", t, func() {
		body, err := getBodyWithFile(invalidXmlPath)
		if err != nil {
			t.Error(err)
		}

		req := httptest.NewRequest("POST", "/admin/payments/bulk-refunds/govpay", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=test_boundary")
		w := httptest.NewRecorder()

		HandleGovPayBulkRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusUnprocessableEntity)
	})

	Convey("Failed to upload bulk refund file - invalid data compared to paymentSession in DB", t, func() {
		body, err := getBodyWithFile(invalidDataXmlPath)
		if err != nil {
			t.Error(err)
		}

		req := httptest.NewRequest("POST", "/admin/payments/bulk-refunds/govpay", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=test_boundary")
		w := httptest.NewRecorder()

		paymentSession := generatePaymentSession()
		cfg, _ := config.Get()

		mockDao := dao.NewMockDAO(mockCtrl)
		mockGovPayService := service.NewMockPaymentProviderService(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDao, cfg)

		refundService = &service.RefundService{
			GovPayService:  mockGovPayService,
			PaymentService: mockPaymentService,
			DAO:            mockDao,
			Config:         *cfg,
		}
		mockDao.EXPECT().GetPaymentResourceByProviderID(gomock.Any()).Return(&paymentSession, nil).AnyTimes()

		HandleGovPayBulkRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusBadRequest)
	})

	Convey("Failed to upload bulk refund file - error returned by service", t, func() {
		body, err := getBodyWithFile(invalidDataXmlPath)
		if err != nil {
			t.Error(err)
		}

		req := httptest.NewRequest("POST", "/admin/payments/bulk-refunds/govpay", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=test_boundary")
		w := httptest.NewRecorder()

		cfg, _ := config.Get()

		mockDao := dao.NewMockDAO(mockCtrl)
		mockGovPayService := service.NewMockPaymentProviderService(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDao, cfg)

		refundService = &service.RefundService{
			GovPayService:  mockGovPayService,
			PaymentService: mockPaymentService,
			DAO:            mockDao,
			Config:         *cfg,
		}
		mockDao.EXPECT().GetPaymentResourceByProviderID(gomock.Any()).Return(nil, fmt.Errorf("error")).AnyTimes()

		HandleGovPayBulkRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Failed to upload bulk refund file - no userID in context", t, func() {
		body, err := getBodyWithFile(xmlFilePath)
		if err != nil {
			t.Error(err)
		}

		req := httptest.NewRequest("POST", "/admin/payments/bulk-refunds/govpay", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=test_boundary")
		w := httptest.NewRecorder()

		cfg, _ := config.Get()
		paymentSession := generatePaymentSession()

		mockDao := dao.NewMockDAO(mockCtrl)
		mockGovPayService := service.NewMockPaymentProviderService(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDao, cfg)

		refundService = &service.RefundService{
			GovPayService:  mockGovPayService,
			PaymentService: mockPaymentService,
			DAO:            mockDao,
			Config:         *cfg,
		}
		mockDao.EXPECT().GetPaymentResourceByProviderID(gomock.Any()).Return(&paymentSession, nil).AnyTimes()

		HandleGovPayBulkRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Failed to upload bulk refund file - error returned from service when updating DB", t, func() {
		body, err := getBodyWithFile(xmlFilePath)
		if err != nil {
			t.Error(err)
		}

		req := httptest.NewRequest("POST", "/admin/payments/bulk-refunds/govpay", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=test_boundary")
		w := httptest.NewRecorder()

		cfg, _ := config.Get()
		paymentSession := generatePaymentSession()

		mockDao := dao.NewMockDAO(mockCtrl)
		mockGovPayService := service.NewMockPaymentProviderService(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDao, cfg)

		refundService = &service.RefundService{
			GovPayService:  mockGovPayService,
			PaymentService: mockPaymentService,
			DAO:            mockDao,
			Config:         *cfg,
		}
		mockDao.EXPECT().GetPaymentResourceByProviderID(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mockDao.EXPECT().CreateBulkRefundByProviderID(gomock.Any()).Return(fmt.Errorf("error")).AnyTimes()

		HandleGovPayBulkRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Error updating DB", t, func() {
		body, err := getBodyWithFile(xmlFilePath)
		if err != nil {
			t.Error(err)
		}

		req := httptest.NewRequest("POST", "/admin/payments/bulk-refunds/govpay", bytes.NewReader(body.Bytes()))
		ctx := req.Context()
		ctx = context.WithValue(ctx, helpers.ContextKeyUserID, "userID")
		req = req.WithContext(ctx)
		req.Header.Set("Content-Type", "multipart/form-data; boundary=test_boundary")
		w := httptest.NewRecorder()

		paymentSession := generatePaymentSession()
		cfg, _ := config.Get()

		mockDao := dao.NewMockDAO(mockCtrl)
		mockGovPayService := service.NewMockPaymentProviderService(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDao, cfg)

		refundService = &service.RefundService{
			GovPayService:  mockGovPayService,
			PaymentService: mockPaymentService,
			DAO:            mockDao,
			Config:         *cfg,
		}

		mockDao.EXPECT().GetPaymentResourceByProviderID(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mockDao.EXPECT().CreateBulkRefundByProviderID(gomock.Any()).Return(fmt.Errorf("err")).AnyTimes()

		HandleGovPayBulkRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Success uploading bulk refund file - Gov Pay", t, func() {
		body, err := getBodyWithFile(xmlFilePath)
		if err != nil {
			t.Error(err)
		}

		req := httptest.NewRequest("POST", "/admin/payments/bulk-refunds/govpay", bytes.NewReader(body.Bytes()))
		ctx := req.Context()
		ctx = context.WithValue(ctx, helpers.ContextKeyUserID, "userID")
		req = req.WithContext(ctx)
		req.Header.Set("Content-Type", "multipart/form-data; boundary=test_boundary")
		w := httptest.NewRecorder()

		paymentSession := generatePaymentSession()
		cfg, _ := config.Get()

		mockDao := dao.NewMockDAO(mockCtrl)
		mockGovPayService := service.NewMockPaymentProviderService(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDao, cfg)

		refundService = &service.RefundService{
			GovPayService:  mockGovPayService,
			PaymentService: mockPaymentService,
			DAO:            mockDao,
			Config:         *cfg,
		}
		mockDao.EXPECT().GetPaymentResourceByProviderID(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mockDao.EXPECT().CreateBulkRefundByProviderID(gomock.Any()).Return(nil).AnyTimes()

		HandleGovPayBulkRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusCreated)
	})

	Convey("Success uploading bulk refund file - PayPal", t, func() {
		body, err := getBodyWithFile(xmlFilePath)
		if err != nil {
			t.Error(err)
		}

		req := httptest.NewRequest("POST", "/admin/payments/bulk-refunds/paypal", bytes.NewReader(body.Bytes()))
		ctx := req.Context()
		ctx = context.WithValue(ctx, helpers.ContextKeyUserID, "userID")
		req = req.WithContext(ctx)
		req.Header.Set("Content-Type", "multipart/form-data; boundary=test_boundary")
		w := httptest.NewRecorder()

		paymentSession := generatePaymentSession()
		paymentSession.Data.PaymentMethod = "PayPal"
		cfg, _ := config.Get()

		mockDao := dao.NewMockDAO(mockCtrl)
		mockGovPayService := service.NewMockPaymentProviderService(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDao, cfg)

		refundService = &service.RefundService{
			GovPayService:  mockGovPayService,
			PaymentService: mockPaymentService,
			DAO:            mockDao,
			Config:         *cfg,
		}

		mockDao.EXPECT().GetPaymentResourceByExternalPaymentTransactionID(gomock.Any()).Return(&paymentSession, nil).AnyTimes()
		mockDao.EXPECT().CreateBulkRefundByExternalPaymentTransactionID(gomock.Any()).Return(nil).AnyTimes()

		HandlePayPalBulkRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusCreated)
	})
}

func TestUnitHandleProcessBulkPendingRefunds(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	Convey("Error returned by service", t, func() {
		cfg, _ := config.Get()

		mockDao := dao.NewMockDAO(mockCtrl)
		mockGovPayService := service.NewMockPaymentProviderService(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDao, cfg)

		refundService = &service.RefundService{
			GovPayService:  mockGovPayService,
			PaymentService: mockPaymentService,
			DAO:            mockDao,
			Config:         *cfg,
		}

		mockDao.EXPECT().GetPaymentsWithRefundStatus().Return(nil, fmt.Errorf("error"))

		req := httptest.NewRequest("POST", "/admin/payments/bulk-refunds/process-pending", nil)
		w := httptest.NewRecorder()

		HandleProcessBulkPendingRefunds(w, req)

		So(w.Code, ShouldEqual, http.StatusAccepted)
		So(w.Body.String(), ShouldContainSubstring, "error retrieving payments with refund-pending status")
	})

	Convey("Multiple errors returned by service", t, func() {
		cfg, _ := config.Get()

		mockDao := dao.NewMockDAO(mockCtrl)
		mockGovPayService := service.NewMockPaymentProviderService(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDao, cfg)

		refundService = &service.RefundService{
			GovPayService:  mockGovPayService,
			PaymentService: mockPaymentService,
			DAO:            mockDao,
			Config:         *cfg,
		}

		paymentSession := generatePaymentSession()
		bulkRefund := models.BulkRefundDB{
			Status:           "refund-pending",
			UploadedFilename: "name",
			UploadedAt:       "time",
			UploadedBy:       "Name",
		}
		paymentSession.BulkRefund = append(paymentSession.BulkRefund, bulkRefund)
		paymentSession1 := generatePaymentSession()
		paymentSession1.ID = "1122"
		paymentSession1.BulkRefund = append(paymentSession1.BulkRefund, bulkRefund)
		pList := []models.PaymentResourceDB{paymentSession, paymentSession1}

		mockDao.EXPECT().GetPaymentsWithRefundStatus().Return(pList, nil)

		req := httptest.NewRequest("POST", "/admin/payments/bulk-refunds/process-pending", nil)
		w := httptest.NewRecorder()

		HandleProcessBulkPendingRefunds(w, req)

		So(w.Code, ShouldEqual, http.StatusAccepted)
		So(w.Body.String(), ShouldContainSubstring, "error converting amount string to int for payment with id [1234],error converting amount string to int for payment with id [1122]")
	})

	Convey("Successfully process pending refund", t, func() {
		cfg, _ := config.Get()

		mockDao := dao.NewMockDAO(mockCtrl)
		mockGovPayService := service.NewMockPaymentProviderService(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDao, cfg)

		refundService = &service.RefundService{
			GovPayService:  mockGovPayService,
			PaymentService: mockPaymentService,
			DAO:            mockDao,
			Config:         *cfg,
		}

		body := fixtures.GetRefundRequest(1000)
		refundSummary := fixtures.GetRefundSummary(1000)
		paymentResource := &models.PaymentResourceRest{}
		refundRequest := fixtures.GetCreateRefundGovPayRequest(body.Amount, refundSummary.AmountAvailable)
		paymentSession := generatePaymentSession()
		response := fixtures.GetCreateRefundGovPayResponse()

		bulkRefund := models.BulkRefundDB{
			Status:           "refund-pending",
			UploadedFilename: "name",
			UploadedAt:       "time",
			UploadedBy:       "Name",
			Amount:           "10.00",
		}
		paymentSession.BulkRefund = append(paymentSession.BulkRefund, bulkRefund)
		pList := []models.PaymentResourceDB{paymentSession}
		mockDao.EXPECT().GetPaymentsWithRefundStatus().Return(pList, nil)
		mockGovPayService.EXPECT().GetRefundSummary(gomock.Any(), gomock.Any()).Return(paymentResource, refundSummary, service.Success, nil)
		mockGovPayService.EXPECT().CreateRefund(paymentResource, refundRequest).Return(response, service.Success, nil)
		mockDao.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil)

		req := httptest.NewRequest("POST", "/admin/payments/bulk-refunds/process-pending", nil)
		w := httptest.NewRecorder()

		HandleProcessBulkPendingRefunds(w, req)

		So(w.Code, ShouldEqual, http.StatusAccepted)
		So(w.Body.String(), ShouldEqual, "\"\"\n")
	})

	Convey("Successfully process multiple pending refunds", t, func() {
		cfg, _ := config.Get()

		mockDao := dao.NewMockDAO(mockCtrl)
		mockGovPayService := service.NewMockPaymentProviderService(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDao, cfg)

		refundService = &service.RefundService{
			GovPayService:  mockGovPayService,
			PaymentService: mockPaymentService,
			DAO:            mockDao,
			Config:         *cfg,
		}

		body := fixtures.GetRefundRequest(1000)
		refundSummary := fixtures.GetRefundSummary(1000)
		paymentResource := &models.PaymentResourceRest{}
		refundRequest := fixtures.GetCreateRefundGovPayRequest(body.Amount, refundSummary.AmountAvailable)
		paymentSession := generatePaymentSession()
		response := fixtures.GetCreateRefundGovPayResponse()
		paymentSession1 := generatePaymentSession()
		bulkRefund := models.BulkRefundDB{
			Status:           "refund-pending",
			UploadedFilename: "name",
			UploadedAt:       "time",
			UploadedBy:       "Name",
			Amount:           "10.00",
		}
		paymentSession.BulkRefund = append(paymentSession.BulkRefund, bulkRefund)
		paymentSession1.BulkRefund = append(paymentSession1.BulkRefund, bulkRefund)
		pList := []models.PaymentResourceDB{paymentSession, paymentSession1}
		mockDao.EXPECT().GetPaymentsWithRefundStatus().Return(pList, nil)
		mockGovPayService.EXPECT().GetRefundSummary(gomock.Any(), gomock.Any()).Return(paymentResource, refundSummary, service.Success, nil).Times(2)
		mockGovPayService.EXPECT().CreateRefund(paymentResource, refundRequest).Return(response, service.Success, nil).Times(2)
		mockDao.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(nil).Times(2)

		req := httptest.NewRequest("POST", "/admin/payments/bulk-refunds/process-pending", nil)
		w := httptest.NewRecorder()

		HandleProcessBulkPendingRefunds(w, req)

		So(w.Code, ShouldEqual, http.StatusAccepted)
		So(w.Body.String(), ShouldEqual, "\"\"\n")
	})
}

func TestUnitHandleGetRefundStatuses(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	Convey("Error retrieving payments with pending refund status from DB", t, func() {
		cfg, _ := config.Get()

		mockDao := dao.NewMockDAO(mockCtrl)
		mockGovPayService := service.NewMockPaymentProviderService(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDao, cfg)

		refundService = &service.RefundService{
			GovPayService:  mockGovPayService,
			PaymentService: mockPaymentService,
			DAO:            mockDao,
			Config:         *cfg,
		}

		mockDao.EXPECT().GetPaymentsWithRefundStatus().Return(nil, fmt.Errorf("error"))

		req := httptest.NewRequest("GET", "/admin/payments/bulk-refunds", nil)
		w := httptest.NewRecorder()

		HandleGetRefundStatuses(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Successful request getting payments with pending refund status from DB", t, func() {
		cfg, _ := config.Get()
		pendingRefunds := fixtures.GetPendingRefundPayments()

		mockDao := dao.NewMockDAO(mockCtrl)
		mockGovPayService := service.NewMockPaymentProviderService(mockCtrl)
		mockPaymentService := createMockPaymentService(mockDao, cfg)

		refundService = &service.RefundService{
			GovPayService:  mockGovPayService,
			PaymentService: mockPaymentService,
			DAO:            mockDao,
			Config:         *cfg,
		}

		mockDao.EXPECT().GetPaymentsWithRefundStatus().Return(pendingRefunds, nil)

		req := httptest.NewRequest("GET", "/admin/payments/bulk-refunds", nil)
		w := httptest.NewRecorder()

		HandleGetRefundStatuses(w, req)

		So(w.Code, ShouldEqual, http.StatusOK)
	})
}

func generatePaymentSession() models.PaymentResourceDB {
	return models.PaymentResourceDB{
		ID:                       "1234",
		RedirectURI:              "/internal/redirect",
		State:                    "state",
		ExternalPaymentStatusURI: "/external/status",
		ExternalPaymentStatusID:  "1122",
		Data: models.PaymentResourceDataDB{
			Amount:        "10.00",
			PaymentMethod: "credit-card",
			Status:        service.Paid.String(),
		},
		Refunds: nil,
	}

}
