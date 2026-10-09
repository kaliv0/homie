package log

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// resetLog restores default package logger state after the test.
func resetLog(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		_ = defaultLogger.Close()
		defaultLogger.Configure(false, "")
	})
}

func readLog(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestConfigureVerbose(t *testing.T) {
	resetLog(t)

	Configure(true, "")
	if !Verbose() || defaultLogger.Std() == nil {
		t.Fatal("expected verbose logger")
	}

	Configure(false, "")
	if Verbose() || defaultLogger.Std() == nil {
		t.Fatal("expected non-verbose logger")
	}
}

func TestConfigure_WritesToFile(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		log     func(path string)
		want    []string
	}{
		{"println", false, func(string) { Println("info-line") }, []string{logPrefix, "info-line"}},
		{"tee when verbose", true, func(string) { Println("tee-line") }, []string{logPrefix, "tee-line"}},
		{"package helpers", false, func(string) {
			Printf("printf-%s\n", "line")
			Println("println-line")
		}, []string{logPrefix, "printf-line", "println-line"}},
		{"same path reused", false, func(path string) {
			Println("first")
			Configure(true, path)
			Println("second")
		}, []string{"first", "second"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetLog(t)
			path := filepath.Join(t.TempDir(), "homie.log")
			Configure(tt.verbose, path)
			tt.log(path)

			got := readLog(t, path)
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("log = %q, want %q", got, want)
				}
			}
		})
	}
}

func TestConfigure_FileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	resetLog(t)

	path := filepath.Join(t.TempDir(), "homie.log")
	Configure(false, path)

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Fatalf("log file mode = %#o, want 0600", got)
	}
}

func TestConfigure_FileFlags(t *testing.T) {
	resetLog(t)

	t.Run("no call site when not verbose", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "homie.log")
		Configure(false, path)
		Println("marker")

		if got := readLog(t, path); got != logPrefix+"marker\n" {
			t.Fatalf("log = %q, want %q", got, logPrefix+"marker\n")
		}
	})

	t.Run("longfile reports caller when verbose", func(t *testing.T) {
		helpers := []struct {
			name string
			log  func()
		}{
			{"Printf", func() { Printf("marker\n") }},
			{"Println", func() { Println("marker") }},
		}
		for _, h := range helpers {
			t.Run(h.name, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "homie.log")
				Configure(true, path)
				h.log()

				got := readLog(t, path)
				rest, ok := strings.CutPrefix(got, logPrefix)
				if !ok {
					t.Fatalf("log = %q, want prefix %q", got, logPrefix)
				}
				site, _, ok := strings.Cut(rest, ": marker")
				if !ok {
					t.Fatalf("log = %q, want file:line before marker", got)
				}
				if !strings.Contains(site, "log_test.go") {
					t.Fatalf("call site %q, want log_test.go", site)
				}
				if !strings.ContainsRune(site, filepath.Separator) {
					t.Fatalf("call site %q, want path separator for Llongfile", site)
				}
			})
		}
	})
}

func TestConfigure_OpenFailureFallsBackToStderr(t *testing.T) {
	l := New(false, filepath.Join(t.TempDir(), "missing-dir", "homie.log"))
	if l.logFile != nil || l.logPath != "" {
		t.Fatalf("logFile=%v logPath=%q, want none after open failure", l.logFile, l.logPath)
	}
	if l.Std() == nil || l.Std().Writer() != os.Stderr {
		t.Fatal("expected stderr logger after open failure")
	}
}

func TestClose_ReleasesFile(t *testing.T) {
	l := New(false, "")
	path := filepath.Join(t.TempDir(), "homie.log")
	l.Configure(false, path)
	l.Std().Print("before")

	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	before := readLog(t, path)
	l.Std().Print("after-close")
	if after := readLog(t, path); after != before {
		t.Fatalf("log file changed after Close: before=%q after=%q", before, after)
	}
	if !strings.Contains(before, "before") {
		t.Fatalf("missing before line: %q", before)
	}
	if l.Std().Writer() != os.Stderr {
		t.Fatal("expected stderr logger after Close")
	}
}

func TestClose_NoFile(t *testing.T) {
	l := New(false, "")
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestClose_PreservesVerbose(t *testing.T) {
	l := New(true, "")
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if !l.Verbose() {
		t.Fatal("expected verbose preserved after Close")
	}
}

func TestFatal(t *testing.T) {
	if os.Getenv("HOMIE_TEST_FATAL") == "1" {
		path := os.Getenv("HOMIE_TEST_FATAL_LOG")
		Configure(true, path)
		Fatal("fatal-line")
		return
	}

	path := filepath.Join(t.TempDir(), "homie.log")
	cmd := exec.Command(os.Args[0], "-test.run=^TestFatal$")
	cmd.Env = append(os.Environ(),
		"HOMIE_TEST_FATAL=1",
		"HOMIE_TEST_FATAL_LOG="+path,
	)
	err := cmd.Run()
	var ee *exec.ExitError
	if !errors.As(err, &ee) || ee.ExitCode() != 1 {
		t.Fatalf("Fatal exit = %v, want exit status 1", err)
	}

	got := readLog(t, path)
	if !strings.Contains(got, "fatal-line") || !strings.Contains(got, logPrefix) {
		t.Fatalf("log = %q, want fatal-line with prefix", got)
	}
	if !strings.Contains(got, "log_test.go") {
		t.Fatalf("log = %q, want caller log_test.go", got)
	}
}
