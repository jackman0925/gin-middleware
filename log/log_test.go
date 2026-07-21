package log

import (
	"bytes"
	"fmt"
	stdlog "log"
	"sync"
	"testing"
)

func resetLogger(t *testing.T) {
	t.Helper()
	SetLogger(nil, LevelError)
	t.Cleanup(func() { SetLogger(nil, LevelError) })
}

func TestSetStdLogger(t *testing.T) {
	resetLogger(t)
	// Should not panic
	SetStdLogger(LevelDebug)
	SetStdLogger(LevelInfo)
	SetStdLogger(LevelWarn)
	SetStdLogger(LevelError)
}

func TestSetLoggerNil(t *testing.T) {
	resetLogger(t)
	SetLogger(nil, LevelDebug)
	// Should not panic — discard logger is used
	Errorf("test")
	Warnf("test")
	Infof("test")
	Debugf("test")
}

func TestLoggingIsDisabledByDefault(t *testing.T) {
	resetLogger(t)
	var output bytes.Buffer
	previousOutput := stdlog.Writer()
	stdlog.SetOutput(&output)
	t.Cleanup(func() { stdlog.SetOutput(previousOutput) })

	if IsEnabled() {
		t.Fatal("logging should be disabled when no logger is configured")
	}
	// Package-level calls are safe no-ops before configuration.
	Infof("discarded")
	if output.Len() != 0 {
		t.Fatalf("unconfigured logging produced output: %q", output.String())
	}
}

func TestConfiguredLoggerIsEnabledByDefaultAndCanBeToggled(t *testing.T) {
	resetLogger(t)
	custom := &testLogger{}
	SetLogger(custom, LevelDebug)

	if !IsEnabled() {
		t.Fatal("a configured logger should be enabled by default")
	}
	Infof("enabled")

	SetEnabled(false)
	if IsEnabled() {
		t.Fatal("logging should be disabled after SetEnabled(false)")
	}
	Infof("disabled")

	SetEnabled(true)
	if !IsEnabled() {
		t.Fatal("logging should be enabled after SetEnabled(true)")
	}
	Infof("enabled again")

	if got := len(custom.infoLogs); got != 2 {
		t.Fatalf("expected 2 emitted logs, got %d", got)
	}
}

func TestEnableWithoutConfiguredLoggerDoesNothing(t *testing.T) {
	resetLogger(t)
	SetEnabled(true)
	if IsEnabled() {
		t.Fatal("logging cannot be enabled without a configured logger")
	}
}

func TestConcurrentConfigurationAndLogging(t *testing.T) {
	resetLogger(t)
	custom := &lockedTestLogger{}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for n := 0; n < 100; n++ {
				if worker%2 == 0 {
					SetLogger(custom, LevelInfo)
					SetEnabled(n%2 == 0)
				} else {
					Infof("worker=%d n=%d", worker, n)
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestSetLoggerCustom(t *testing.T) {
	resetLogger(t)
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
	resetLogger(t)
	SetStdLogger(LevelInfo)
	l := GetLogger()
	if l == nil {
		t.Fatal("Logger() should not return nil")
	}
}

func TestGlobalFunctions(t *testing.T) {
	resetLogger(t)
	SetStdLogger(LevelDebug)
	// Should not panic
	Errorf("error: %d", 1)
	Warnf("warn: %d", 2)
	Infof("info: %d", 3)
	Debugf("debug: %d", 4)
}

type lockedTestLogger struct {
	mu sync.Mutex
}

func (l *lockedTestLogger) add() {
	l.mu.Lock()
	l.mu.Unlock()
}

func (l *lockedTestLogger) Errorf(string, ...any) { l.add() }
func (l *lockedTestLogger) Warnf(string, ...any)  { l.add() }
func (l *lockedTestLogger) Infof(string, ...any)  { l.add() }
func (l *lockedTestLogger) Debugf(string, ...any) { l.add() }

type testLogger struct {
	errorLogs []string
	warnLogs  []string
	infoLogs  []string
	debugLogs []string
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
