package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// DB & FS configuration constants
const (
	DefaultLimit   = 20
	DefaultMaxSize = 500

	// DB configs
	MaxDbConnections = 2
	ConnMaxLifetime  = 12 * time.Hour
	// SQLite pragmas
	DbBusyTimeout = 5000 // 5s in milliseconds
	JournalMode   = "WAL"
	Sync          = "NORMAL"

	// FS configs
	XDGConf     = "XDG_CONFIG_HOME"
	AppConfPath = "$HOME/"
	// db
	DbConfDirPerm = 0755
	DbConfDirName = ".config"
	DbSubdirName  = "homie"
	DbFileName    = "homie.db"
	// app
	ConfFileName = ".homierc"
	ConfFileType = "yaml"
)

// ReadConfig loads configuration from ~/.homierc once.
var ReadConfig = sync.OnceValue(readConfig)

func readConfig() error {
	viper.SetConfigName(ConfFileName)
	viper.SetConfigType(ConfFileType)
	viper.AddConfigPath(AppConfPath)
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return fmt.Errorf("failed to read config file %s from %s: %w", ConfFileName, AppConfPath, err)
		}
	}
	return nil
}

var (
	dbPathOnce sync.Once
	dbPathVal  string
	dbPathErr  error
)

// DBPath returns the absolute path to the SQLite database file.
func DBPath() (string, error) {
	dbPathOnce.Do(func() {
		var subDirsList []string
		if xdgConf := os.Getenv(XDGConf); xdgConf != "" {
			subDirsList = append(subDirsList, xdgConf)
		} else {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				dbPathErr = fmt.Errorf("failed to get user home directory: %w", err)
				return
			}
			subDirsList = append(subDirsList, homeDir, DbConfDirName)
		}
		subDirsList = append(subDirsList, DbSubdirName)
		configDir := filepath.Join(subDirsList...)
		if err := os.MkdirAll(configDir, DbConfDirPerm); err != nil {
			dbPathErr = fmt.Errorf("failed to create config directory %q: %w", configDir, err)
			return
		}
		dbPathVal = filepath.Join(configDir, DbFileName)
	})
	return dbPathVal, dbPathErr
}
