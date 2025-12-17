package config

import (
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/viper"
)

// ReadConfig loads configuration from ~/.homierc once.
var ReadConfig = sync.OnceValue(readConfig)

func readConfig() error {
	viper.SetConfigName(".homierc")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/")
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return err
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
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig != "" {
			subDirsList = append(subDirsList, xdgConfig)
		} else {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				dbPathErr = err
				return
			}
			subDirsList = append(subDirsList, homeDir, ".config")
		}
		subDirsList = append(subDirsList, "homie")
		configDir := filepath.Join(subDirsList...)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			dbPathErr = err
			return
		}
		dbPathVal = filepath.Join(configDir, "homie.db")
	})
	return dbPathVal, dbPathErr
}
