package config

import (
	"errors"
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
		pflag.CommandLine.StringP("pathTLSCert", "r", "", "Path to TLS certificate")
	}
	if pflag.CommandLine.Lookup("pathTLSKey") == nil {
		pflag.CommandLine.StringP("pathTLSKey", "k", "", "Path to TLS key")
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
	v.SetDefault("token", "jwt_secret_token")
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
	if err := v.BindPFlag("token", pflag.CommandLine.Lookup("token")); err != nil {
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

// Validate проверяет конфигурацию сервера.
func (c *ConfigServer) Validate() error {
	if c.Net.Host == "" {
		return errors.New("хост не может быть пустым")
	}
	if c.Net.Port <= 0 || c.Net.Port > 65535 {
		return errors.New("передан неверный порт")
	}
	if c.Protocol != "grpc" && c.Protocol != "https" {
		return errors.New("неверный протокол обмена")
	}
	if c.DatabaseDSN == "" {
		return errors.New("строка подключения к базе данных не может быть пустой")
	}
	if c.Token == "" {
		return errors.New("токен не может быть пустым")
	}
	if len(c.Token) < 16 {
		return errors.New("передан короткий токен, для безопасности требуется не менее 16 символов")
	}
	if c.PathTLSCert == "" {
		return errors.New("путь к TLS-сертификату не может быть пустым")
	}
	if c.PathTLSKey == "" {
		return errors.New("путь к TLS-ключу не может быть пустым")
	}
	_, err := os.Stat(c.PathTLSCert)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("файл TLS-сертификата не найден")
		}
		return err
	}
	_, err = os.Stat(c.PathTLSKey)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("файл TLS-ключа не найден")
		}
		return err
	}
	return nil
}
