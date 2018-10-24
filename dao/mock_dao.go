// Code generated by MockGen. DO NOT EDIT.
// Source: dao.go

// Package dao is a generated GoMock package.
package dao

import (
	models "github.com/companieshouse/payments.api.ch.gov.uk/models"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockDAO is a mock of DAO interface
type MockDAO struct {
	ctrl     *gomock.Controller
	recorder *MockDAOMockRecorder
}

// MockDAOMockRecorder is the mock recorder for MockDAO
type MockDAOMockRecorder struct {
	mock *MockDAO
}

// NewMockDAO creates a new mock instance
func NewMockDAO(ctrl *gomock.Controller) *MockDAO {
	mock := &MockDAO{ctrl: ctrl}
	mock.recorder = &MockDAOMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockDAO) EXPECT() *MockDAOMockRecorder {
	return m.recorder
}

// CreatePaymentResource mocks base method
func (m *MockDAO) CreatePaymentResource(arg0 *models.PaymentResource) error {
	ret := m.ctrl.Call(m, "CreatePaymentResource", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreatePaymentResource indicates an expected call of CreatePaymentResource
func (mr *MockDAOMockRecorder) CreatePaymentResource(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreatePaymentResource", reflect.TypeOf((*MockDAO)(nil).CreatePaymentResource), arg0)
}

// GetPaymentResource mocks base method
func (m *MockDAO) GetPaymentResource(arg0 string) (models.PaymentResource, error) {
	ret := m.ctrl.Call(m, "GetPaymentResource", arg0)
	ret0, _ := ret[0].(models.PaymentResource)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPaymentResource indicates an expected call of GetPaymentResource
func (mr *MockDAOMockRecorder) GetPaymentResource(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPaymentResource", reflect.TypeOf((*MockDAO)(nil).GetPaymentResource), arg0)
}
