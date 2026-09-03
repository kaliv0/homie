package log

import (
	stdlog "log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// resetLog restores default package logger state after the test.
func resetLog(t *testing.T) {
	t.Helper()
	t.Cleanup(restoreDefaultLogger)
}

func restoreDefaultLogger() {
	mu.Lock()
	defer mu.Unlock()
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
		logPath = ""
	}
	verbose = false
	std = stdlog.New(os.Stderr, logPrefix, stdlog.Llongfile)
}

func TestConfigureVerbose(t *testing.T) {
	resetLog(t)

	Configure(true, "")
	if !Verbose() || Logger() == nil {
		t.Fatal("expected verbose logger")
	}

	Configure(false, "")
	if Verbose() || Logger() == nil {
		t.Fatal("expected non-verbose logger")
	}
}

func TestConfigureLogFile(t *testing.T) {
	resetLog(t)

	path := filepath.Join(t.TempDir(), "homie.log")
	Configure(false, path)
	Logger().Printf("info-line\n")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "D'OH: ") || !strings.Contains(got, "info-line") {
		t.Fatalf("log file contents = %q, want homie: prefix, file:line, and message", got)
	}

	if runtime.GOOS != "windows" {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := fi.Mode().Perm() & 0o777; got != 0o600 {
			t.Fatalf("log file mode = %#o, want 0600", got)
		}
	}
}

func TestConfigureSameLogPathReused(t *testing.T) {
	resetLog(t)

	path := filepath.Join(t.TempDir(), "homie.log")
	Configure(false, path)
	Logger().Printf("first\n")
	Configure(true, path)
	Logger().Printf("second\n")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "first") || !strings.Contains(got, "second") {
		t.Fatalf("log file = %q, want both first and second", got)
	}
}

func TestConfigure_TeeToFile(t *testing.T) {
	resetLog(t)
	path := filepath.Join(t.TempDir(), "homie.log")

	Configure(true, path)
	Logger().Printf("tee-line\n")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "tee-line") {
		t.Fatalf("expected tee line in file, got %q", string(data))
	}
}
