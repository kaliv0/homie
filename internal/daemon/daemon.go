package daemon

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/kaliv0/homie/internal/config"
	"github.com/kaliv0/homie/internal/log"
)

const (
	pidFileFlags  = os.O_CREATE | os.O_RDWR
	pidFilePerm   = 0o600
	pollInterval  = 100 * time.Millisecond
	daemonTimeout = 3 * time.Second
)

// ErrAlreadyRunning is returned when another daemon holds the pidfile lock.
var ErrAlreadyRunning = errors.New("daemon already running")

// Lock holds the pidfile open with an exclusive flock for the daemon lifetime.
type Lock struct {
	file *os.File
	path string
}

// Acquire opens the pidfile, takes an exclusive lock, and writes the current PID.
func Acquire(cfg *config.Config) (*Lock, error) {
	if err := cfg.PreparePIDFile(); err != nil {
		return nil, err
	}
	path := cfg.PIDFile

	f, err := os.OpenFile(path, pidFileFlags, pidFilePerm)
	if err != nil {
		return nil, err
	}

	// acquire lock
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		closeErr := f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrAlreadyRunning
		}
		return nil, errors.Join(err, closeErr)
	}
	// clear file (if stale there will old invalid pid)
	if err = f.Truncate(0); err != nil {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		closeErr := f.Close()
		return nil, errors.Join(err, closeErr)
	}
	// write current pid
	if _, err = fmt.Fprintf(f, "%d\n", os.Getpid()); err != nil {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		closeErr := f.Close()
		return nil, errors.Join(err, closeErr)
	}

	return &Lock{file: f, path: path}, nil
}

// Release unlocks the pidfile, closes it, and removes it.
func (l *Lock) Release() error {
	_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	return errors.Join(l.file.Close(), os.Remove(l.path))
}

// Status reports whether a daemon holds the pidfile lock.
func Status(cfg *config.Config) (bool, int, error) {
	path := cfg.PIDFile
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, 0, nil
		}
		return false, 0, err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			log.Logger().Println(closeErr)
		}
	}()

	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) {
			pid, err := readPID(path)
			if err != nil {
				return false, 0, err
			}
			return true, pid, nil
		}
		return false, 0, err
	}

	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return false, 0, nil
}

// Stop sends SIGTERM to the running daemon and waits until it releases the pidfile lock.
func Stop(cfg *config.Config) error {
	running, pid, err := Status(cfg)
	if err != nil {
		return err
	}
	if running {
		proc, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			return err
		}
	}
	return waitUntilStopped(cfg)
}

// Start spawns binary args in a new session and waits until the daemon holds the pidfile lock.
func Start(cfg *config.Config, binary string, args ...string) error {
	cmd := exec.Command(binary, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start daemon process (command=%q %v): %w", binary, args, err)
	}
	defer func() {
		if err := cmd.Process.Release(); err != nil {
			log.Logger().Printf("failed to release daemon process: %v\n", err)
		}
	}()
	return waitUntilRunning(cfg)
}

func waitUntilStopped(cfg *config.Config) error {
	return waitForStatus(cfg, false, "stop")
}

func waitUntilRunning(cfg *config.Config) error {
	return waitForStatus(cfg, true, "start")
}

func waitForStatus(cfg *config.Config, wantRunning bool, action string) error {
	deadline := time.Now().Add(daemonTimeout)
	for time.Now().Before(deadline) {
		running, _, err := Status(cfg)
		if err != nil {
			return err
		}
		if running == wantRunning {
			return nil
		}
		time.Sleep(pollInterval)
	}
	return fmt.Errorf("timed out waiting for daemon to %s after %s", action, daemonTimeout)
}

func readPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}
