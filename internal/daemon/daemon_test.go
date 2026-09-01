package daemon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"

	"github.com/kaliv0/homie/internal/config"
)

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Parse(viper.New())
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}
	return cfg
}

func testPIDFile(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)
	return filepath.Join(tmpDir, "homie.pid")
}

func writePIDFile(t *testing.T, path string, pid int) {
	t.Helper()
	if err := os.WriteFile(path, []byte(fmt.Sprintf("%d\n", pid)), 0600); err != nil {
		t.Fatalf("failed to write pidfile: %v", err)
	}
}

func TestAcquire_Release(t *testing.T) {
	testPIDFile(t)
	cfg := testConfig(t)

	lock, err := Acquire(cfg)
	if err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}

	running, pid, err := Status(cfg)
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}
	if !running || pid != os.Getpid() {
		t.Fatalf("expected running with pid %d, got running=%v pid=%d", os.Getpid(), running, pid)
	}

	if err := lock.Release(); err != nil {
		t.Fatalf("Release() failed: %v", err)
	}

	running, _, err = Status(cfg)
	if err != nil {
		t.Fatalf("Status() after release failed: %v", err)
	}
	if running {
		t.Fatal("expected not running after release")
	}
}

func TestAcquire_AlreadyRunning(t *testing.T) {
	testPIDFile(t)
	cfg := testConfig(t)

	lock, err := Acquire(cfg)
	if err != nil {
		t.Fatalf("first Acquire() failed: %v", err)
	}
	defer func() {
		_ = lock.Release()
	}()

	_, err = Acquire(cfg)
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("expected ErrAlreadyRunning, got %v", err)
	}
}

func TestStop_NoPidfile(t *testing.T) {
	testPIDFile(t)
	cfg := testConfig(t)

	if err := Stop(cfg); err != nil {
		t.Fatalf("Stop() with no pidfile failed: %v", err)
	}
}

func TestStop_StalePidfile(t *testing.T) {
	path := testPIDFile(t)
	cfg := testConfig(t)

	writePIDFile(t, path, 999999)

	if err := Stop(cfg); err != nil {
		t.Fatalf("Stop() with stale pid failed: %v", err)
	}
}

func TestStatus_StalePidfile(t *testing.T) {
	path := testPIDFile(t)
	cfg := testConfig(t)

	writePIDFile(t, path, 999999)

	running, _, err := Status(cfg)
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}
	if running {
		t.Fatal("expected stale pidfile to report not running")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected stale pidfile to remain: %v", err)
	}
}

func TestStatus_StalePidfileLivePID(t *testing.T) {
	path := testPIDFile(t)
	cfg := testConfig(t)

	writePIDFile(t, path, os.Getpid())

	running, _, err := Status(cfg)
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}
	if running {
		t.Fatal("expected unlocked pidfile with live PID to report not running")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected stale pidfile to remain: %v", err)
	}
}
