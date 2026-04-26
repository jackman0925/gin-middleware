package log

import (
	"fmt"
	"testing"
)

func TestSetStdLogger(t *testing.T) {
	// Should not panic
	SetStdLogger(LevelDebug)
	SetStdLogger(LevelInfo)
	SetStdLogger(LevelWarn)
	SetStdLogger(LevelError)
}

func TestSetLoggerNil(t *testing.T) {
	SetLogger(nil, LevelDebug)
	// Should not panic — discard logger is used
	Errorf("test")
	Warnf("test")
	Infof("test")
	Debugf("test")
}

func TestSetLoggerCustom(t *testing.T) {
	custom := &testLogger{}
	SetLogger(custom, LevelInfo)

	Infof("hello %s", "world")
	if len(custom.infoLogs) != 1 {
		t.Fatalf("expected 1 info log, got %d", len(custom.infoLogs))
	}
	if custom.infoLogs[0] != "hello world" {
		t.Fatalf("expected 'hello world', got '%s'", custom.infoLogs[0])
	}

	Debugf("should be dropped")
	if len(custom.debugLogs) != 0 {
		t.Fatalf("expected 0 debug logs at LevelInfo, got %d", len(custom.debugLogs))
	}
}

func TestLoggerReturnsCurrent(t *testing.T) {
	SetStdLogger(LevelInfo)
	l := GetLogger()
	if l == nil {
		t.Fatal("Logger() should not return nil")
	}
}

func TestGlobalFunctions(t *testing.T) {
	SetStdLogger(LevelDebug)
	// Should not panic
	Errorf("error: %d", 1)
	Warnf("warn: %d", 2)
	Infof("info: %d", 3)
	Debugf("debug: %d", 4)
}

type testLogger struct {
	errorLogs  []string
	warnLogs   []string
	infoLogs   []string
	debugLogs  []string
}

func (l *testLogger) Errorf(format string, v ...any) {
	l.errorLogs = append(l.errorLogs, sprintf(format, v...))
}

func (l *testLogger) Warnf(format string, v ...any) {
	l.warnLogs = append(l.warnLogs, sprintf(format, v...))
}

func (l *testLogger) Infof(format string, v ...any) {
	l.infoLogs = append(l.infoLogs, sprintf(format, v...))
}

func (l *testLogger) Debugf(format string, v ...any) {
	l.debugLogs = append(l.debugLogs, sprintf(format, v...))
}

func sprintf(format string, v ...any) string {
	return fmt.Sprintf(format, v...)
}
