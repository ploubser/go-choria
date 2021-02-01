// Code generated by MockGen. DO NOT EDIT.
// Source: filter.go

// Package protocol is a generated GoMock package.
package protocol

import (
	json "encoding/json"
	ddl "github.com/choria-io/go-choria/providers/data/ddl"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockLogger is a mock of Logger interface
type MockLogger struct {
	ctrl     *gomock.Controller
	recorder *MockLoggerMockRecorder
}

// MockLoggerMockRecorder is the mock recorder for MockLogger
type MockLoggerMockRecorder struct {
	mock *MockLogger
}

// NewMockLogger creates a new mock instance
func NewMockLogger(ctrl *gomock.Controller) *MockLogger {
	mock := &MockLogger{ctrl: ctrl}
	mock.recorder = &MockLoggerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockLogger) EXPECT() *MockLoggerMockRecorder {
	return m.recorder
}

// Warnf mocks base method
func (m *MockLogger) Warnf(format string, args ...interface{}) {
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Warnf", varargs...)
}

// Warnf indicates an expected call of Warnf
func (mr *MockLoggerMockRecorder) Warnf(format interface{}, args ...interface{}) *gomock.Call {
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Warnf", reflect.TypeOf((*MockLogger)(nil).Warnf), varargs...)
}

// Debugf mocks base method
func (m *MockLogger) Debugf(format string, args ...interface{}) {
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Debugf", varargs...)
}

// Debugf indicates an expected call of Debugf
func (mr *MockLoggerMockRecorder) Debugf(format interface{}, args ...interface{}) *gomock.Call {
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Debugf", reflect.TypeOf((*MockLogger)(nil).Debugf), varargs...)
}

// Errorf mocks base method
func (m *MockLogger) Errorf(format string, args ...interface{}) {
	varargs := []interface{}{format}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Errorf", varargs...)
}

// Errorf indicates an expected call of Errorf
func (mr *MockLoggerMockRecorder) Errorf(format interface{}, args ...interface{}) *gomock.Call {
	varargs := append([]interface{}{format}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Errorf", reflect.TypeOf((*MockLogger)(nil).Errorf), varargs...)
}

// MockServerInfoSource is a mock of ServerInfoSource interface
type MockServerInfoSource struct {
	ctrl     *gomock.Controller
	recorder *MockServerInfoSourceMockRecorder
}

// MockServerInfoSourceMockRecorder is the mock recorder for MockServerInfoSource
type MockServerInfoSourceMockRecorder struct {
	mock *MockServerInfoSource
}

// NewMockServerInfoSource creates a new mock instance
func NewMockServerInfoSource(ctrl *gomock.Controller) *MockServerInfoSource {
	mock := &MockServerInfoSource{ctrl: ctrl}
	mock.recorder = &MockServerInfoSourceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockServerInfoSource) EXPECT() *MockServerInfoSourceMockRecorder {
	return m.recorder
}

// Classes mocks base method
func (m *MockServerInfoSource) Classes() []string {
	ret := m.ctrl.Call(m, "Classes")
	ret0, _ := ret[0].([]string)
	return ret0
}

// Classes indicates an expected call of Classes
func (mr *MockServerInfoSourceMockRecorder) Classes() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Classes", reflect.TypeOf((*MockServerInfoSource)(nil).Classes))
}

// Facts mocks base method
func (m *MockServerInfoSource) Facts() json.RawMessage {
	ret := m.ctrl.Call(m, "Facts")
	ret0, _ := ret[0].(json.RawMessage)
	return ret0
}

// Facts indicates an expected call of Facts
func (mr *MockServerInfoSourceMockRecorder) Facts() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Facts", reflect.TypeOf((*MockServerInfoSource)(nil).Facts))
}

// Identity mocks base method
func (m *MockServerInfoSource) Identity() string {
	ret := m.ctrl.Call(m, "Identity")
	ret0, _ := ret[0].(string)
	return ret0
}

// Identity indicates an expected call of Identity
func (mr *MockServerInfoSourceMockRecorder) Identity() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Identity", reflect.TypeOf((*MockServerInfoSource)(nil).Identity))
}

// KnownAgents mocks base method
func (m *MockServerInfoSource) KnownAgents() []string {
	ret := m.ctrl.Call(m, "KnownAgents")
	ret0, _ := ret[0].([]string)
	return ret0
}

// KnownAgents indicates an expected call of KnownAgents
func (mr *MockServerInfoSourceMockRecorder) KnownAgents() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "KnownAgents", reflect.TypeOf((*MockServerInfoSource)(nil).KnownAgents))
}

// DataFuncMap mocks base method
func (m *MockServerInfoSource) DataFuncMap() (ddl.FuncMap, error) {
	ret := m.ctrl.Call(m, "DataFuncMap")
	ret0, _ := ret[0].(ddl.FuncMap)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DataFuncMap indicates an expected call of DataFuncMap
func (mr *MockServerInfoSourceMockRecorder) DataFuncMap() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DataFuncMap", reflect.TypeOf((*MockServerInfoSource)(nil).DataFuncMap))
}
