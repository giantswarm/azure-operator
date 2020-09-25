// Code generated by MockGen. DO NOT EDIT.
// Source: spec.go

// Package mock_tenantcluster is a generated GoMock package.
package mock_tenantcluster

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// MockFactory is a mock of Factory interface
type MockFactory struct {
	ctrl     *gomock.Controller
	recorder *MockFactoryMockRecorder
}

// MockFactoryMockRecorder is the mock recorder for MockFactory
type MockFactoryMockRecorder struct {
	mock *MockFactory
}

// NewMockFactory creates a new mock instance
func NewMockFactory(ctrl *gomock.Controller) *MockFactory {
	mock := &MockFactory{ctrl: ctrl}
	mock.recorder = &MockFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockFactory) EXPECT() *MockFactoryMockRecorder {
	return m.recorder
}

// GetClient mocks base method
func (m *MockFactory) GetClient(ctx context.Context, cr *v1alpha3.Cluster) (client.Client, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetClient", ctx, cr)
	ret0, _ := ret[0].(client.Client)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetClient indicates an expected call of GetClient
func (mr *MockFactoryMockRecorder) GetClient(ctx, cr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetClient", reflect.TypeOf((*MockFactory)(nil).GetClient), ctx, cr)
}
