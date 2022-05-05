// Code generated by MockGen. DO NOT EDIT.
// Source: dao/dao.go

// Package dao is a generated GoMock package.
package dao

import (
	reflect "reflect"

	models "github.com/companieshouse/payments.api.ch.gov.uk/models"
	gomock "github.com/golang/mock/gomock"
)

// MockDAO is a mock of DAO interface.
type MockDAO struct {
	ctrl     *gomock.Controller
	recorder *MockDAOMockRecorder
}

// MockDAOMockRecorder is the mock recorder for MockDAO.
type MockDAOMockRecorder struct {
	mock *MockDAO
}

// NewMockDAO creates a new mock instance.
func NewMockDAO(ctrl *gomock.Controller) *MockDAO {
	mock := &MockDAO{ctrl: ctrl}
	mock.recorder = &MockDAOMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDAO) EXPECT() *MockDAOMockRecorder {
	return m.recorder
}

// CreateBulkRefund mocks base method.
func (m *MockDAO) CreateBulkRefund(providerID string, bulkRefund models.BulkRefundDB) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateBulkRefund", providerID, bulkRefund)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateBulkRefund indicates an expected call of CreateBulkRefund.
func (mr *MockDAOMockRecorder) CreateBulkRefund(providerID, bulkRefund interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateBulkRefund", reflect.TypeOf((*MockDAO)(nil).CreateBulkRefund), providerID, bulkRefund)
}

// CreatePaymentResource mocks base method.
func (m *MockDAO) CreatePaymentResource(paymentResource *models.PaymentResourceDB) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreatePaymentResource", paymentResource)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreatePaymentResource indicates an expected call of CreatePaymentResource.
func (mr *MockDAOMockRecorder) CreatePaymentResource(paymentResource interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreatePaymentResource", reflect.TypeOf((*MockDAO)(nil).CreatePaymentResource), paymentResource)
}

// GetPaymentResource mocks base method.
func (m *MockDAO) GetPaymentResource(arg0 string) (*models.PaymentResourceDB, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPaymentResource", arg0)
	ret0, _ := ret[0].(*models.PaymentResourceDB)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPaymentResource indicates an expected call of GetPaymentResource.
func (mr *MockDAOMockRecorder) GetPaymentResource(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPaymentResource", reflect.TypeOf((*MockDAO)(nil).GetPaymentResource), arg0)
}

// GetPaymentResourceByProviderID mocks base method.
func (m *MockDAO) GetPaymentResourceByProviderID(providerID string) (*models.PaymentResourceDB, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPaymentResourceByProviderID", providerID)
	ret0, _ := ret[0].(*models.PaymentResourceDB)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPaymentResourceByProviderID indicates an expected call of GetPaymentResourceByProviderID.
func (mr *MockDAOMockRecorder) GetPaymentResourceByProviderID(providerID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPaymentResourceByProviderID", reflect.TypeOf((*MockDAO)(nil).GetPaymentResourceByProviderID), providerID)
}

// GetPaymentsWithRefundStatus mocks base method.
func (m *MockDAO) GetPaymentsWithRefundStatus() ([]models.PaymentResourceDB, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPaymentsWithRefundStatus")
	ret0, _ := ret[0].([]models.PaymentResourceDB)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPaymentsWithRefundStatus indicates an expected call of GetPaymentsWithRefundStatus.
func (mr *MockDAOMockRecorder) GetPaymentsWithRefundStatus() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPaymentsWithRefundStatus", reflect.TypeOf((*MockDAO)(nil).GetPaymentsWithRefundStatus))
}

// PatchPaymentResource mocks base method.
func (m *MockDAO) PatchPaymentResource(id string, paymentUpdate *models.PaymentResourceDB) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PatchPaymentResource", id, paymentUpdate)
	ret0, _ := ret[0].(error)
	return ret0
}

// PatchPaymentResource indicates an expected call of PatchPaymentResource.
func (mr *MockDAOMockRecorder) PatchPaymentResource(id, paymentUpdate interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PatchPaymentResource", reflect.TypeOf((*MockDAO)(nil).PatchPaymentResource), id, paymentUpdate)
}
