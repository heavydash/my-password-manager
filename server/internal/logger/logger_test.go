package logger

import "go.uber.org/zap"

type TestLogger struct{}

func NewTestLogger() Logger {
	return &TestLogger{}
}

func (TestLogger) Debug(string, ...zap.Field) {}
func (TestLogger) Info(string, ...zap.Field)  {}
func (TestLogger) Warn(string, ...zap.Field)  {}
func (TestLogger) Error(string, ...zap.Field) {}
func (TestLogger) With(...zap.Field) Logger   { return TestLogger{} }
func (TestLogger) Sync() error                { return nil }
