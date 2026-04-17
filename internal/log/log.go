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

const logFilePerm = 0o600

var (
	mu      sync.RWMutex
	std     *log.Logger
	verbose int
	logFile *os.File
	logPath string // path of the open logFile; empty if none
)

// loggerStyle is stderr logger settings per verbosity tier (index = level, capped at len-1).
var loggerStyle = []struct {
	prefix string
	flags  int
}{
	{prefix: "D'OH: ", flags: 0},
	{prefix: "homie: ", flags: 0},
	{prefix: "homie: ", flags: log.Llongfile},
}

func init() {
	Configure(0, "")
}

// Configure sets verbosity (0 default, 1 Infof, 2+ Debugf and file:line) and optional append-only log (0o600, teed to stderr).
func Configure(level int, filePath string) {
	mu.Lock()
	defer mu.Unlock()
	if level < 0 {
		level = 0
	}
	verbose = level
	filePath = strings.TrimSpace(filePath)
	if filePath != logPath {
		swapLogFile(filePath)
	}

	out := io.Writer(os.Stderr)
	if logFile != nil {
		out = io.MultiWriter(os.Stderr, logFile)
	}
	s := loggerStyle[min(level, len(loggerStyle)-1)]
	std = log.New(out, s.prefix, s.flags)
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
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, logFilePerm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "homie: open log file %q: %v\n", path, err)
		return
	}
	logFile, logPath = f, path
}

// ConfigureFromFlags resolves verbosity/log-file from flags+config and applies logger setup.
// If a flag is explicitly set, it overrides .homierc.
func ConfigureFromFlags(pflags *pflag.FlagSet) {
	var level int
	if v := pflags.Lookup("verbose"); v != nil && v.Changed {
		level, _ = pflags.GetCount("verbose")
	} else {
		level = viper.GetInt(config.ViperKeyVerbose)
	}
	if level < 0 {
		level = 0
	}

	var filePath string
	if v := pflags.Lookup("log-file"); v != nil && v.Changed {
		filePath, _ = pflags.GetString("log-file")
	} else {
		filePath = viper.GetString(config.ViperKeyLogFile)
	}
	Configure(level, expandHomeDir(strings.TrimSpace(filePath)))
}

func expandHomeDir(p string) string {
	if p == "" || !strings.HasPrefix(p, "~/") {
		return p
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(homeDir, strings.TrimPrefix(p, "~/"))
}

// Verbose returns the current verbosity level (repeat -v / --verbose count).
func Verbose() int {
	mu.RLock()
	defer mu.RUnlock()
	return verbose
}

// Logger is the shared logger for errors and unavoidable messages (stderr, and the log file if configured).
func Logger() *log.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return std
}

// Infof logs when verbosity >= 1 (-v).
func Infof(format string, v ...any) {
	mu.RLock()
	if verbose < 1 {
		mu.RUnlock()
		return
	}
	l := std
	mu.RUnlock()
	l.Printf(format, v...)
}

// Debugf logs when verbosity >= 2 (-vv).
func Debugf(format string, v ...any) {
	mu.RLock()
	if verbose < 2 {
		mu.RUnlock()
		return
	}
	l := std
	mu.RUnlock()
	l.Printf(format, v...)
}
