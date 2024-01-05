package logging

import "github.com/stretchr/testify/mock"

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(args ...interface{}) {
	m.Called(args)
}

func (m *MockLogger) Debugf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Info(args ...interface{}) {
	m.Called(args)
}

func (m *MockLogger) Infof(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Warn(args ...interface{}) {
	m.Called(args)
}

func (m *MockLogger) Warnf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Error(args ...interface{}) {
	m.Called(args)
}

func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.Called(format, args)
}

type NilLogger struct {}
func (m *NilLogger) Debug(args ...interface{}) {}
func (m *NilLogger) Debugf(format string, args ...interface{}) {}
func (m *NilLogger) Info(args ...interface{}) {}
func (m *NilLogger) Infof(format string, args ...interface{}) {}
func (m *NilLogger) Warn(args ...interface{}) {}
func (m *NilLogger) Warnf(format string, args ...interface{}) {}
func (m *NilLogger) Error(args ...interface{}) {}
func (m *NilLogger) Errorf(format string, args ...interface{}) {}