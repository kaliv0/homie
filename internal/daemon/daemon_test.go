package daemon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func testPIDFile(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)
	return filepath.Join(tmpDir, "homie.pid")
}

func TestAcquire_Release(t *testing.T) {
	testPIDFile(t)

	lock, err := Acquire()
	if err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}

	running, pid, err := Status()
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}
	if !running || pid != os.Getpid() {
		t.Fatalf("expected running with pid %d, got running=%v pid=%d", os.Getpid(), running, pid)
	}

	if err := lock.Release(); err != nil {
		t.Fatalf("Release() failed: %v", err)
	}

	running, _, err = Status()
	if err != nil {
		t.Fatalf("Status() after release failed: %v", err)
	}
	if running {
		t.Fatal("expected not running after release")
	}
}

func TestAcquire_AlreadyRunning(t *testing.T) {
	testPIDFile(t)

	lock, err := Acquire()
	if err != nil {
		t.Fatalf("first Acquire() failed: %v", err)
	}
	defer func() {
		_ = lock.Release()
	}()

	_, err = Acquire()
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("expected ErrAlreadyRunning, got %v", err)
	}
}

func TestStop_NoPidfile(t *testing.T) {
	testPIDFile(t)

	if err := Stop(); err != nil {
		t.Fatalf("Stop() with no pidfile failed: %v", err)
	}
}

func TestStop_StalePidfile(t *testing.T) {
	path := testPIDFile(t)

	if err := os.WriteFile(path, []byte("999999\n"), 0600); err != nil {
		t.Fatalf("failed to write stale pidfile: %v", err)
	}

	if err := Stop(); err != nil {
		t.Fatalf("Stop() with stale pid failed: %v", err)
	}
}

func TestStatus_StalePidfile(t *testing.T) {
	path := testPIDFile(t)

	if err := os.WriteFile(path, []byte("999999\n"), 0600); err != nil {
		t.Fatalf("failed to write stale pidfile: %v", err)
	}

	running, _, err := Status()
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

	if err := os.WriteFile(path, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0600); err != nil {
		t.Fatalf("failed to write stale pidfile: %v", err)
	}

	running, _, err := Status()
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
