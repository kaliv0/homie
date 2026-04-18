package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/kaliv0/homie/internal/config"
)

const (
	logFileFlags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	logFilePerm  = 0o600
)

const (
	verboseConfig = "verbose"
	fileConfig    = "log-file"
	homeDirPrefix = "~/"
	logPrefix = "D'OH: "
)

var (
	mu      sync.RWMutex
	std     *log.Logger
	verbose bool
	logFile *os.File
	logPath string // path of the open logFile; empty if none
)

func init() {
	Configure(false, "")
}

// ConfigureFromFlags applies verbose and log-file from flags and .homierc (explicit flags override config).
func ConfigureFromFlags(pflags *pflag.FlagSet) {
	var verboseEnabled bool
	if pflags.Changed(verboseConfig) {
		verboseEnabled, _ = pflags.GetBool(verboseConfig)
	} else {
		verboseEnabled = viper.GetBool(config.ViperKeyVerbose)
	}

	var filePath string
	if pflags.Changed(fileConfig) {
		filePath, _ = pflags.GetString(fileConfig)
	} else {
		filePath = viper.GetString(config.ViperKeyLogFile)
	}
	expandedPath := expandPath(strings.TrimSpace(filePath))
	Configure(verboseEnabled, expandedPath)
}

func expandPath(p string) string {
	if p == "" || !strings.HasPrefix(p, homeDirPrefix) {
		return p
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(homeDir, strings.TrimPrefix(p, homeDirPrefix))
}

// Configure sets verbose diagnostics, an optional append-only log file (0o600), and tee (stderr + file).
func Configure(verboseEnabled bool, filePath string) {
	mu.Lock()
	defer mu.Unlock()
	verbose = verboseEnabled
	filePath = strings.TrimSpace(filePath)
	if filePath != logPath {
		swapLogFile(filePath)
	}

	toFile := logFile != nil
	tee := toFile && verbose && filePath != ""

	var out io.Writer
	switch {
	case tee:
		out = io.MultiWriter(os.Stderr, logFile)
	case toFile:
		out = logFile
	default:
		out = os.Stderr
	}
	std = log.New(out, logPrefix, log.Llongfile)
}

// swapLogFile closes the current log file (if any) and opens path when non-empty.
func swapLogFile(path string) {
	if logFile != nil {
		if err := logFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "homie: close log file %q: %v\n", logPath, err)
		}
		logFile = nil
		logPath = ""
	}
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, logFileFlags, logFilePerm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "homie: open log file %q: %v\n", path, err)
		return
	}
	logFile, logPath = f, path
}

// Verbose reports whether verbose diagnostics are enabled (from -v / .homierc).
func Verbose() bool {
	mu.RLock()
	defer mu.RUnlock()
	return verbose
}

// Logger is the shared logger (stderr, log file, or both when teeing).
func Logger() *log.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return std
}
