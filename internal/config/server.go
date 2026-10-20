package config

import (
	"os"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// ConfigServer описывает настройки клиента.
type ConfigServer struct {
	Net         NetAddress `mapstructure:"address"`
	ConfigPath  string     `mapstructure:"config_path"`
	Protocol    string     `mapstructure:"protocol"`
	DatabaseDSN string     `mapstructure:"database_dsn"`
	PathTLSCert string     `mapstructure:"path_tls_cert"`
	PathTLSKey  string     `mapstructure:"path_tls_key"`
	Token       string     `mapstructure:"token"`
}

// Get парсит конфигурацию сервера.
func (c *ConfigServer) Get() error {
	if pflag.CommandLine.Lookup("address") == nil {
		pflag.CommandLine.StringP("address", "a", "", "Address and port to connect to server")
	}
	if pflag.CommandLine.Lookup("configPath") == nil {
		pflag.CommandLine.StringP("configPath", "c", "", "Path to server JSON config file")
	}
	if pflag.CommandLine.Lookup("protocol") == nil {
		pflag.CommandLine.StringP("protocol", "p", "", "Transport protocol to use")
	}
	if pflag.CommandLine.Lookup("databaseDSN") == nil {
		pflag.CommandLine.StringP("database_dsn", "d", "", "DatabaseDSN")
	}
	if pflag.CommandLine.Lookup("pathTLSCert") == nil {
		pflag.CommandLine.StringP("pathTLSCert", "cert", "", "Path to TLS certificate")
	}
	if pflag.CommandLine.Lookup("pathTLSKey") == nil {
		pflag.CommandLine.StringP("pathTLSKey", "key", "", "Path to TLS key")
	}
	if pflag.CommandLine.Lookup("token") == nil {
		pflag.CommandLine.StringP("token", "t", "", "Authentication token")
	}
	pflag.Parse()
	v := viper.New()
	v.SetDefault("address", "localhost:8080")
	v.SetDefault("protocol", "https")
	v.SetDefault("path_tls_cert", "certs/server.crt")
	v.SetDefault("path_tls_key", "certs/server.key")
	v.SetDefault("token", "jwtsecrettoken")
	v.SetEnvPrefix("server")
	v.AutomaticEnv()
	if err := v.BindPFlag("address", pflag.CommandLine.Lookup("address")); err != nil {
		return err
	}
	if err := v.BindPFlag("protocol", pflag.CommandLine.Lookup("protocol")); err != nil {
		return err
	}
	if err := v.BindPFlag("config_path", pflag.CommandLine.Lookup("configPath")); err != nil {
		return err
	}
	if err := v.BindPFlag("database_dsn", pflag.CommandLine.Lookup("database_dsn")); err != nil {
		return err
	}
	if err := v.BindPFlag("path_tls_cert", pflag.CommandLine.Lookup("pathTLSCert")); err != nil {
		return err
	}
	if err := v.BindPFlag("path_tls_key", pflag.CommandLine.Lookup("pathTLSKey")); err != nil {
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
