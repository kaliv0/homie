package daemon

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/kaliv0/homie/internal/config"
	"github.com/kaliv0/homie/internal/log"
)

const (
	pidFileFlags = os.O_CREATE | os.O_RDWR
	pidFilePerm  = 0o600
)

// ErrAlreadyRunning is returned when another daemon holds the pidfile lock.
var ErrAlreadyRunning = errors.New("daemon already running")

// Lock holds the pidfile open with an exclusive flock for the daemon lifetime.
type Lock struct {
	file *os.File
	path string
}

// Acquire opens the pidfile, takes an exclusive lock, and writes the current PID.
func Acquire() (*Lock, error) {
	// find path and open file
	path, err := config.PreparePIDFile()
	if err != nil {
		return nil, err
	}

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
func Status() (bool, int, error) {
	// find path to pidfile
	path, err := config.PIDFilePath()
	if err != nil {
		return false, 0, err
	}
	// open file
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

	// try to acquire lock
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) {
			// file already locked -> another instance is running
			pid, err := readPID(path)
			if err != nil {
				return false, 0, err
			}
			// return pid of running instance
			return true, pid, nil
		}
		return false, 0, err
	}

	// if we were able to acquire lock to existing file -> it is stale,
	// leftover from a previous instance no longer holding the lock
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return false, 0, nil
}

// Stop sends SIGTERM to the daemon PID from the pidfile.
func Stop() error {
	// check if running
	running, pid, err := Status()
	if err != nil || !running {
		return err
	}
	// find running process & terminate it
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGTERM)
}

func readPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}
