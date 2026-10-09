package log

import (
	"fmt"
	"io"
	stdlog "log"
	"os"
	"sync"
)

const (
	logFileFlags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	logFilePerm  = 0o600
)

const logPrefix = "D'OH: "
const calldepth = 2

// Logger wraps stdlib logger with concurrency-safe config and optional append-only log file.
type Logger struct {
	mu      sync.RWMutex
	std     *stdlog.Logger
	verbose bool
	logFile *os.File
	logPath string // path of the open logFile, empty if none
}

// New initializes Logger with given verbose flag and optional log file path.
func New(isVerbose bool, filePath string) *Logger {
	l := &Logger{}
	l.Configure(isVerbose, filePath)
	return l
}

// Configure updates verbose diagnostics, optional append-only log file (0o600) and output (stderr, file, or tee both).
func (l *Logger) Configure(isVerbose bool, filePath string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.verbose = isVerbose
	if filePath != l.logPath {
		oldPath := l.logPath
		if err := l.closeFile(); err != nil {
			fmt.Fprintf(os.Stderr, "homie: close log file %q: %v\n", oldPath, err)
		}
		l.openFile(filePath)
	}
	l.rebuild()
}

// Verbose reports whether verbose diagnostics are enabled.
func (l *Logger) Verbose() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.verbose
}

// Std returns the underlying standard library logger.
func (l *Logger) Std() *stdlog.Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.std
}

// Close closes the log file if open and points the logger back at stderr.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	err := l.closeFile()
	l.rebuild()
	return err
}

// rebuild creates std from current state -> stderr, file, or tee when verbose.
func (l *Logger) rebuild() {
	var out io.Writer = os.Stderr
	if l.logFile != nil {
		out = l.logFile
		if l.verbose {
			out = io.MultiWriter(os.Stderr, l.logFile)
		}
	}
	flag := 0
	if l.verbose {
		flag = stdlog.Llongfile
	}
	l.std = stdlog.New(out, logPrefix, flag)
}

// openFile opens path for appending -> reports failure on stderr and keeps logging there.
func (l *Logger) openFile(path string) {
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, logFileFlags, logFilePerm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "homie: open log file %q: %v\n", path, err)
		return
	}
	l.logFile, l.logPath = f, path
}

// closeFile closes log file if open and clears its path.
func (l *Logger) closeFile() error {
	if l.logFile == nil {
		return nil
	}
	err := l.logFile.Close()
	l.logFile, l.logPath = nil, ""
	return err
}

var defaultLogger = New(false, "")

// Configure sets verbose diagnostics and optional append-only log file on default logger.
// Called only at process start.
func Configure(isVerbose bool, filePath string) {
	defaultLogger.Configure(isVerbose, filePath)
}

// Verbose reports whether verbose diagnostics are enabled on default logger.
func Verbose() bool {
	return defaultLogger.Verbose()
}

/* NB: Calldepth 2 skips the following wrappers -> Llongfile reports caller's file and line. */
// Printf calls Printf on default logger.
func Printf(format string, v ...any) {
	_ = defaultLogger.Std().Output(calldepth, fmt.Sprintf(format, v...))
}

// Println calls Println on default logger.
func Println(v ...any) {
	_ = defaultLogger.Std().Output(calldepth, fmt.Sprintln(v...))
}

// Fatal calls Fatal on default logger.
func Fatal(v ...any) {
	_ = defaultLogger.Std().Output(calldepth, fmt.Sprint(v...))
	os.Exit(1)
}
