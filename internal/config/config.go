package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// Viper keys (and YAML keys in .homierc).
const (
	ViperKeyVerbose = "verbose"
	ViperKeyLogFile = "log_file"
	ViperKeyPIDFile = "pid_file"
)

const (
	xdgConf       = "XDG_CONFIG_HOME"
	xdgRuntime    = "XDG_RUNTIME_DIR"
	runDir        = "/run/user"
	appConfPath   = "$HOME/"
	homeDirPrefix = "~/"

	dbConfDirPerm = 0755
	dbConfDirName = ".config"
	dbSubdirName  = "homie"
	dbFileName    = "homie.db"

	pidFileName = "homie.pid"

	confFileName = ".homierc"
	confFileType = "yaml"
)

// ReadConfig loads configuration from ~/.homierc once.
var ReadConfig = sync.OnceValue(readConfig)

func readConfig() error {
	viper.SetDefault(ViperKeyVerbose, false)
	viper.SetDefault(ViperKeyLogFile, "")
	viper.SetDefault(ViperKeyPIDFile, "")

	viper.SetConfigName(confFileName)
	viper.SetConfigType(confFileType)
	viper.AddConfigPath(appConfPath)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); !ok {
			return fmt.Errorf("failed to read config file %s from %s: %w", confFileName, appConfPath, err)
		}
	}
	return nil
}

var (
	once    sync.Once
	dbPath  string
	pathErr error
)

// DBPath returns the absolute path to the SQLite database file.
func DBPath() (string, error) {
	once.Do(func() {
		var subDirsList []string
		if xdgHome := os.Getenv(xdgConf); xdgHome != "" {
			subDirsList = append(subDirsList, xdgHome)
		} else {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				pathErr = fmt.Errorf("failed to get user home directory: %w", err)
				return
			}
			subDirsList = append(subDirsList, homeDir, dbConfDirName)
		}
		subDirsList = append(subDirsList, dbSubdirName)
		configDir := filepath.Join(subDirsList...)
		if err := os.MkdirAll(configDir, dbConfDirPerm); err != nil {
			pathErr = fmt.Errorf("failed to create config directory %q: %w", configDir, err)
			return
		}
		dbPath = filepath.Join(configDir, dbFileName)
	})
	return dbPath, pathErr
}

// PIDFilePath returns the path to the daemon pidfile.
func PIDFilePath() (string, error) {
	if err := ReadConfig(); err != nil {
		return "", err
	}
	if p := ExpandHomePath(strings.TrimSpace(viper.GetString(ViperKeyPIDFile))); p != "" {
		return p, nil
	}

	if xdg := os.Getenv(xdgRuntime); xdg != "" {
		return filepath.Join(xdg, pidFileName), nil
	}
	return filepath.Join(runDir, fmt.Sprintf("%d", os.Getuid()), pidFileName), nil
}

// PreparePIDFile returns the pidfile path and ensures its parent directory exists.
func PreparePIDFile() (string, error) {
	path, err := PIDFilePath()
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(filepath.Dir(path), dbConfDirPerm); err != nil {
		return "", fmt.Errorf("failed to create pidfile directory: %w", err)
	}
	return path, nil
}

func ExpandHomePath(p string) string {
	if p == "" || !strings.HasPrefix(p, homeDirPrefix) {
		return p
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(homeDir, strings.TrimPrefix(p, homeDirPrefix))
}
