package util

import (
	"github.com/spf13/viper"
)

// Config stores all configuration of the application.
// The values are used by read by viper from a config file or environment variable.
type Config struct {
	GeminiKey string `mapstructure:"GEMINI_API_KEY"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
