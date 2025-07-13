package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	ServiceName string `mapstructure:"service_name"`
	Exporter string `mapstructure:"exporter"`
	LogLevel string `mapstructure:"log_level"`
}

func Load(path string) (config Config, err error) {
	viper.SetDefault("service_name", "unknown-service")
	viper.SetDefault("exporter", "logging")
	viper.SetDefault("log_level", "info")

	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err = viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return
		}
		err = nil
	}

	err = viper.Unmarshal(&config)
	return
} 
