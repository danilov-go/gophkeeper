package config

import (
	"os"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// ConfigClient описывает настройки клиента.
type ConfigClient struct {
	Net        NetAddress `mapstructure:"address"`
	ConfigPath string     `mapstructure:"config_path"`
}

// Get парсит конфигурацию клиента.
func (c *ConfigClient) Get() error {
	if pflag.CommandLine.Lookup("address") == nil {
		pflag.CommandLine.StringP("address", "a", "", "Address and port to connect to client")
	}
	if pflag.CommandLine.Lookup("configPath") == nil {
		pflag.CommandLine.StringP("configPath", "c", "", "Path to client JSON config file")
	}
	pflag.Parse()
	v := viper.New()
	v.SetDefault("address", "localhost:8080")
	v.SetEnvPrefix("client")
	v.AutomaticEnv()
	if err := v.BindPFlag("address", pflag.CommandLine.Lookup("address")); err != nil {
		return err
	}
	if err := v.BindPFlag("config_path", pflag.CommandLine.Lookup("configPath")); err != nil {
		return err
	}
	configFilePath := v.GetString("config_path")
	if configFilePath != "" {
		if _, err := os.Stat(configFilePath); err == nil {
			v.SetConfigFile(configFilePath)
			v.SetConfigType("json")
			if err := v.ReadInConfig(); err != nil {
				return err
			}
		}
	}
	err := v.Unmarshal(c, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.TextUnmarshallerHookFunc(),
		),
	))
	if err != nil {
		return err
	}
	return nil
}
