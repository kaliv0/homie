package utils

import (
	"errors"
	"sync"

	"github.com/spf13/viper"
)

var ReadConfig = sync.OnceFunc(readConfig)

func readConfig() {
	viper.SetConfigName(".homeyrc")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/")
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			Logger.Fatal(err)
		}
	}
}
