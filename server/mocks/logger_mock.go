// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/mattermost/mattermost/server/public/pluginapi/experimental/bot/logger (interfaces: Logger)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	logger "github.com/mattermost/mattermost/server/public/pluginapi/experimental/bot/logger"
)

// MockLogger is a mock of Logger interface.
type MockLogger struct {
	ctrl     *gomock.Controller
	recorder *MockLoggerMockRecorder
}

// MockLoggerMockRecorder is the mock recorder for MockLogger.
type MockLoggerMockRecorder struct {
	mock *MockLogger
}

// NewMockLogger creates a new mock instance.
func NewMockLogger(ctrl *gomock.Controller) *MockLogger {
	mock := &MockLogger{ctrl: ctrl}
	mock.recorder = &MockLoggerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockLogger) EXPECT() *MockLoggerMockRecorder {
	return m.recorder
}

// Context mocks base method.
func (m *MockLogger) Context() logger.LogContext {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(logger.LogContext)
	return ret0
}

// Context indicates an expected call of Context.
func (mr *MockLoggerMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockLogger)(nil).Context))
}

// Debugf mocks base method.
func (m *MockLogger) Debugf(arg0 string, arg1 ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Debugf", varargs...)
}

// Debugf indicates an expected call of Debugf.
func (mr *MockLoggerMockRecorder) Debugf(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Debugf", reflect.TypeOf((*MockLogger)(nil).Debugf), varargs...)
}

// Errorf mocks base method.
func (m *MockLogger) Errorf(arg0 string, arg1 ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Errorf", varargs...)
}

// Errorf indicates an expected call of Errorf.
func (mr *MockLoggerMockRecorder) Errorf(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Errorf", reflect.TypeOf((*MockLogger)(nil).Errorf), varargs...)
}

// Infof mocks base method.
func (m *MockLogger) Infof(arg0 string, arg1 ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Infof", varargs...)
}

// Infof indicates an expected call of Infof.
func (mr *MockLoggerMockRecorder) Infof(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Infof", reflect.TypeOf((*MockLogger)(nil).Infof), varargs...)
}

// Timed mocks base method.
func (m *MockLogger) Timed() logger.Logger {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Timed")
	ret0, _ := ret[0].(logger.Logger)
	return ret0
}

// Timed indicates an expected call of Timed.
func (mr *MockLoggerMockRecorder) Timed() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Timed", reflect.TypeOf((*MockLogger)(nil).Timed))
}

// Warnf mocks base method.
func (m *MockLogger) Warnf(arg0 string, arg1 ...interface{}) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Warnf", varargs...)
}

// Warnf indicates an expected call of Warnf.
func (mr *MockLoggerMockRecorder) Warnf(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Warnf", reflect.TypeOf((*MockLogger)(nil).Warnf), varargs...)
}

// With mocks base method.
func (m *MockLogger) With(arg0 logger.LogContext) logger.Logger {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "With", arg0)
	ret0, _ := ret[0].(logger.Logger)
	return ret0
}

// With indicates an expected call of With.
func (mr *MockLoggerMockRecorder) With(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "With", reflect.TypeOf((*MockLogger)(nil).With), arg0)
}

// WithError mocks base method.
func (m *MockLogger) WithError(arg0 error) logger.Logger {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WithError", arg0)
	ret0, _ := ret[0].(logger.Logger)
	return ret0
}

// WithError indicates an expected call of WithError.
func (mr *MockLoggerMockRecorder) WithError(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WithError", reflect.TypeOf((*MockLogger)(nil).WithError), arg0)
}