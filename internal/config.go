package internal

import (
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/viper"
)

// ReadConfig loads configuration from ~/.homierc once.
var ReadConfig = sync.OnceFunc(readConfig)

// DBPath returns the absolute path to the SQLite database file.
var DBPath = sync.OnceValue(dbPath)

func readConfig() {
	viper.SetConfigName(".homierc")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/")
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			Logger.Fatal(err)
		}
	}
}

func dbPath() string {
	var subDirsList []string
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig != "" {
		subDirsList = append(subDirsList, xdgConfig)
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			Logger.Fatal(err)
		}
		subDirsList = append(subDirsList, homeDir, ".config")
	}
	subDirsList = append(subDirsList, "homie")
	configDir := filepath.Join(subDirsList...)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		Logger.Fatal(err)
	}
	return filepath.Join(configDir, "homie.db")
}
