package handlers

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	xmlFilePath = "bulk_refund.xml"
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

func TestUnitHandleGovPayBulkRefund(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	Convey("File not supplied", t, func() {
		req := httptest.NewRequest("POST", "/admin/payments/bulk-refunds/govpay", nil)
		w := httptest.NewRecorder()
		HandleBulkRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Success uploading bulk refund file", t, func() {
		body, err := getBodyWithFile(xmlFilePath)
		if err != nil {
			t.Error(err)
		}

		req := httptest.NewRequest("POST", "//admin/payments/bulk-refunds/govpay", bytes.NewReader(body.Bytes()))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=test_boundary")
		w := httptest.NewRecorder()

		HandleBulkRefund(w, req)
		So(w.Code, ShouldEqual, http.StatusCreated)
	})
}
