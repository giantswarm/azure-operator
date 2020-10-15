// Code generated by MockGen. DO NOT EDIT.
// Source: internal/azure/spec.go

// Package mock_azure is a generated GoMock package.
package mock_azure

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"

	azure "github.com/giantswarm/azure-operator/v5/service/controller/resource/workermigration/internal/azure"
)

// MockAPI is a mock of API interface
type MockAPI struct {
	ctrl     *gomock.Controller
	recorder *MockAPIMockRecorder
}

// MockAPIMockRecorder is the mock recorder for MockAPI
type MockAPIMockRecorder struct {
	mock *MockAPI
}

// NewMockAPI creates a new mock instance
func NewMockAPI(ctrl *gomock.Controller) *MockAPI {
	mock := &MockAPI{ctrl: ctrl}
	mock.recorder = &MockAPIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAPI) EXPECT() *MockAPIMockRecorder {
	return m.recorder
}

// GetVMSS mocks base method
func (m *MockAPI) GetVMSS(ctx context.Context, resourceGroupName, vmssName string) (azure.VMSS, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVMSS", ctx, resourceGroupName, vmssName)
	ret0, _ := ret[0].(azure.VMSS)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVMSS indicates an expected call of GetVMSS
func (mr *MockAPIMockRecorder) GetVMSS(ctx, resourceGroupName, vmssName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVMSS", reflect.TypeOf((*MockAPI)(nil).GetVMSS), ctx, resourceGroupName, vmssName)
}

// DeleteVMSS mocks base method
func (m *MockAPI) DeleteVMSS(ctx context.Context, resourceGroupName, vmssName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteVMSS", ctx, resourceGroupName, vmssName)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteVMSS indicates an expected call of DeleteVMSS
func (mr *MockAPIMockRecorder) DeleteVMSS(ctx, resourceGroupName, vmssName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteVMSS", reflect.TypeOf((*MockAPI)(nil).DeleteVMSS), ctx, resourceGroupName, vmssName)
}

// ListVMSSNodes mocks base method
func (m *MockAPI) ListVMSSNodes(ctx context.Context, resourceGroupName, vmssName string) (azure.VMSSNodes, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListVMSSNodes", ctx, resourceGroupName, vmssName)
	ret0, _ := ret[0].(azure.VMSSNodes)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListVMSSNodes indicates an expected call of ListVMSSNodes
func (mr *MockAPIMockRecorder) ListVMSSNodes(ctx, resourceGroupName, vmssName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListVMSSNodes", reflect.TypeOf((*MockAPI)(nil).ListVMSSNodes), ctx, resourceGroupName, vmssName)
}