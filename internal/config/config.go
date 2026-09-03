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

// ~/.homierc keys
const (
	Verbose   = "verbose"
	LogFile   = "log_file"
	PIDFile   = "pid_file"
	Limit     = "limit"
	TTL       = "ttl"
	Keep      = "keep"
	Threshold = "threshold"
	Tool      = "tool"
)

const (
	xdgConf       = "XDG_CONFIG_HOME"
	xdgRuntime    = "XDG_RUNTIME_DIR"
	runDir        = "/run/user"
	appConfPath   = "$HOME/"
	homeDirPrefix = "~/"

	dbConfDirPerm = 0o700
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
	viper.SetDefault(Verbose, false)
	viper.SetDefault(LogFile, "")
	viper.SetDefault(PIDFile, "")
	viper.SetDefault(Tool, "")
	viper.SetDefault(TTL, 0)
	viper.SetDefault(Limit, DefaultLimit)
	viper.SetDefault(Keep, DefaultKeep)
	viper.SetDefault(Threshold, DefaultThreshold)

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

// PreparePIDFile ensures the pidfile parent directory exists.
func (c *Config) PreparePIDFile() error {
	if err := os.MkdirAll(filepath.Dir(c.PIDFile), dbConfDirPerm); err != nil {
		return fmt.Errorf("failed to create pidfile directory: %w", err)
	}
	return nil
}

func (c *Config) resolvePaths() {
	c.LogFile = expandHomePath(c.LogFile)
	c.PIDFile = expandHomePath(c.PIDFile)
	if c.PIDFile == "" {
		c.resolvePIDFileDefault()
	}
}

func (c *Config) resolvePIDFileDefault() {
	if xdg := os.Getenv(xdgRuntime); xdg != "" {
		c.PIDFile = filepath.Join(xdg, pidFileName)
		return
	}
	c.PIDFile = filepath.Join(runDir, fmt.Sprintf("%d", os.Getuid()), pidFileName)
}

func expandHomePath(p string) string {
	if p == "" || !strings.HasPrefix(p, homeDirPrefix) {
		return p
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(homeDir, strings.TrimPrefix(p, homeDirPrefix))
}
