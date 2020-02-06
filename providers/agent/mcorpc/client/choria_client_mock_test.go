// Code generated by MockGen. DO NOT EDIT.
// Source: client.go

package client

import (
	context "context"
	choria "github.com/choria-io/go-choria/choria"
	client "github.com/choria-io/go-choria/client/client"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockRequestResult is a mock of RequestResult interface
type MockRequestResult struct {
	ctrl     *gomock.Controller
	recorder *MockRequestResultMockRecorder
}

// MockRequestResultMockRecorder is the mock recorder for MockRequestResult
type MockRequestResultMockRecorder struct {
	mock *MockRequestResult
}

// NewMockRequestResult creates a new mock instance
func NewMockRequestResult(ctrl *gomock.Controller) *MockRequestResult {
	mock := &MockRequestResult{ctrl: ctrl}
	mock.recorder = &MockRequestResultMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockRequestResult) EXPECT() *MockRequestResultMockRecorder {
	return m.recorder
}

// Stats mocks base method
func (m *MockRequestResult) Stats() *Stats {
	ret := m.ctrl.Call(m, "Stats")
	ret0, _ := ret[0].(*Stats)
	return ret0
}

// Stats indicates an expected call of Stats
func (mr *MockRequestResultMockRecorder) Stats() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stats", reflect.TypeOf((*MockRequestResult)(nil).Stats))
}

// MockChoriaClient is a mock of ChoriaClient interface
type MockChoriaClient struct {
	ctrl     *gomock.Controller
	recorder *MockChoriaClientMockRecorder
}

// MockChoriaClientMockRecorder is the mock recorder for MockChoriaClient
type MockChoriaClientMockRecorder struct {
	mock *MockChoriaClient
}

// NewMockChoriaClient creates a new mock instance
func NewMockChoriaClient(ctrl *gomock.Controller) *MockChoriaClient {
	mock := &MockChoriaClient{ctrl: ctrl}
	mock.recorder = &MockChoriaClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockChoriaClient) EXPECT() *MockChoriaClientMockRecorder {
	return m.recorder
}

// Request mocks base method
func (m *MockChoriaClient) Request(ctx context.Context, msg *choria.Message, handler client.Handler) error {
	ret := m.ctrl.Call(m, "Request", ctx, msg, handler)
	ret0, _ := ret[0].(error)
	return ret0
}

// Request indicates an expected call of Request
func (mr *MockChoriaClientMockRecorder) Request(ctx, msg, handler interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Request", reflect.TypeOf((*MockChoriaClient)(nil).Request), ctx, msg, handler)
}
