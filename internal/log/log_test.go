package log

import (
	"bytes"
	stdlog "log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/kaliv0/homie/internal/config"
)

func viperFromYAML(t *testing.T, yaml string) *viper.Viper {
	t.Helper()
	v := viper.New()
	if yaml == "" {
		return v
	}
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewBufferString(yaml)); err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	return v
}

// configureFromConfig mirrors cmd/root.go PersistentPreRunE logging setup.
func configureFromConfig(t *testing.T, yaml string) {
	t.Helper()
	cfg, err := config.Parse(viperFromYAML(t, yaml))
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}
	Configure(cfg.Verbose, cfg.LogFile)
}

// resetLog restores default package logger state.
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

func TestConfigureFromConfig_UsesVerbose(t *testing.T) {
	resetLog(t)

	configureFromConfig(t, "verbose: true\n")
	if !Verbose() {
		t.Fatal("Verbose() = false, want true")
	}
}

func TestConfigureFromConfig_ExpandsHomeInLogFile(t *testing.T) {
	resetLog(t)
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configureFromConfig(t, "verbose: true\nlog_file: ~/homie-configure-from-command.log\n")
	Logger().Printf("hello\n")

	path := filepath.Join(tmpDir, "homie-configure-from-command.log")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected expanded log file at %q: %v", path, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "hello") {
		t.Fatalf("log file = %q, want hello", string(data))
	}
}

func TestConfigureFromConfig_TeeToFile(t *testing.T) {
	resetLog(t)
	path := filepath.Join(t.TempDir(), "homie.log")

	configureFromConfig(t, "verbose: true\nlog_file: "+path+"\n")
	Logger().Printf("tee-line\n")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "tee-line") {
		t.Fatalf("expected tee line in file, got %q", string(data))
	}
}
