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

// deadPID is assumed to belong to no running process.
const deadPID = 999999

// setup points XDG_RUNTIME_DIR at a temp dir -> then parses config so PIDFile resolves there.
func setup(t *testing.T) (*config.Config, string) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)
	return config.Parse(viper.New()), filepath.Join(tmpDir, "homie.pid")
}

func writePIDFile(t *testing.T, path string, pid int) {
	t.Helper()
	if err := os.WriteFile(path, fmt.Appendf(nil, "%d\n", pid), 0o600); err != nil {
		t.Fatalf("failed to write pidfile: %v", err)
	}
}

func TestAcquire_Release(t *testing.T) {
	cfg, _ := setup(t)

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
	cfg, _ := setup(t)

	lock, err := Acquire(cfg)
	if err != nil {
		t.Fatalf("first Acquire() failed: %v", err)
	}
	t.Cleanup(func() { _ = lock.Release() })

	_, err = Acquire(cfg)
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("expected ErrAlreadyRunning, got %v", err)
	}
}

func TestStop_NoPidfile(t *testing.T) {
	cfg, _ := setup(t)

	if err := Stop(cfg); err != nil {
		t.Fatalf("Stop() with no pidfile failed: %v", err)
	}
}

func TestStop_StalePidfile(t *testing.T) {
	cfg, path := setup(t)
	writePIDFile(t, path, deadPID)

	if err := Stop(cfg); err != nil {
		t.Fatalf("Stop() with stale pid failed: %v", err)
	}
}

func TestStatus_UnlockedPidfile(t *testing.T) {
	tests := []struct {
		name string
		pid  int
	}{
		{"dead pid", deadPID},
		{"live pid", os.Getpid()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, path := setup(t)
			writePIDFile(t, path, tt.pid)

			running, _, err := Status(cfg)
			if err != nil {
				t.Fatalf("Status() failed: %v", err)
			}
			if running {
				t.Fatal("expected unlocked pidfile to report not running")
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("expected stale pidfile to remain: %v", err)
			}
		})
	}
}
