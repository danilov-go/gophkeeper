package config

import (
	"os"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// ConfigServer описывает настройки клиента.
type ConfigClient struct {
	Net        NetAddress `mapstructure:"ADDRESS"`
	ConfigPath string     `mapstructure:"CONFIG_PATH"`
}

// Get парсит конфигурацию клиента.
func (c *ConfigClient) Get() error {
	if pflag.CommandLine.Lookup("address") == nil {
		pflag.CommandLine.StringP("address", "a", "", "Address and port to run client")
	}
	if pflag.CommandLine.Lookup("configPath") == nil {
		pflag.CommandLine.StringP("configPath", "c", "", "Path to JSON file config")
	}
	pflag.Parse()
	if err := viper.BindPFlag("ADDRESS", pflag.CommandLine.Lookup("address")); err != nil {
		return err
	}
	if err := viper.BindPFlag("CONFIG_PATH", pflag.CommandLine.Lookup("configPath")); err != nil {
		return err
	}
	viper.AutomaticEnv()
	configJson := viper.GetString("CONFIG_PATH")
	if configJson != "" {
		if _, err := os.Stat(configJson); err == nil {
			viper.SetConfigFile(configJson)
			viper.SetConfigType("json")
			if err := viper.ReadInConfig(); err != nil {
				return err
			}
		}
	}
	err := viper.Unmarshal(c, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.TextUnmarshallerHookFunc(),
		),
	))
	if err != nil {
		return err
	}
	return nil
}
