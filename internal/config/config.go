package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/viper"
)

const (
	xdgConf       = "XDG_CONFIG_HOME"
	appConfPath   = "$HOME/"
	dbConfDirPerm = 0755
	dbConfDirName = ".config"
	dbSubdirName  = "homie"
	dbFileName    = "homie.db"

	confFileName = ".homierc"
	confFileType = "yaml"
)

// ReadConfig loads configuration from ~/.homierc once.
var ReadConfig = sync.OnceValue(readConfig)

func readConfig() error {
	viper.SetConfigName(confFileName)
	viper.SetConfigType(confFileType)
	viper.AddConfigPath(appConfPath)
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return fmt.Errorf("failed to read config file %s from %s: %w", confFileName, appConfPath, err)
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
		if xdgConf := os.Getenv(xdgConf); xdgConf != "" {
			subDirsList = append(subDirsList, xdgConf)
		} else {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				dbPathErr = fmt.Errorf("failed to get user home directory: %w", err)
				return
			}
			subDirsList = append(subDirsList, homeDir, dbConfDirName)
		}
		subDirsList = append(subDirsList, dbSubdirName)
		configDir := filepath.Join(subDirsList...)
		if err := os.MkdirAll(configDir, dbConfDirPerm); err != nil {
			dbPathErr = fmt.Errorf("failed to create config directory %q: %w", configDir, err)
			return
		}
		dbPathVal = filepath.Join(configDir, dbFileName)
	})
	return dbPathVal, dbPathErr
}
