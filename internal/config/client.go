package config

import (
	"errors"
	"os"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// ConfigClient описывает настройки клиента.
type ConfigClient struct {
	// Net содержит сетевой адрес и порт сервера.
	Net NetAddress `mapstructure:"address"`
	// ConfigPath содержит путь к конфигурационному файлу JSON.
	ConfigPath string `mapstructure:"config_path"`
	// Protocol содержит используемый транспортный протокол.
	Protocol string `mapstructure:"protocol"`
	// PathDB содержит путь к файлу базы данных.
	PathDB string `mapstructure:"path_db"`
	// SendInterval содержит интервал времени между отправками данных на сервер.
	SendInterval time.Duration `mapstructure:"send_interval"`
	// Cooldown содержит время ожидания перед повторной попыткой при ошибке соединения.
	Cooldown time.Duration `mapstructure:"cooldown"`
}

// Get парсит конфигурацию клиента.
func (c *ConfigClient) Get() error {
	if pflag.CommandLine.Lookup("address") == nil {
		pflag.CommandLine.StringP("address", "a", "", "Address and port to connect to client")
	}
	if pflag.CommandLine.Lookup("configPath") == nil {
		pflag.CommandLine.StringP("configPath", "c", "", "Path to server JSON config file")
	}
	if pflag.CommandLine.Lookup("protocol") == nil {
		pflag.CommandLine.StringP("protocol", "p", "", "Transport protocol to use")
	}
	if pflag.CommandLine.Lookup("pathDB") == nil {
		pflag.CommandLine.StringP("pathDB", "d", "", "Path to DB file")
	}
	if pflag.CommandLine.Lookup("sendInterval") == nil {
		pflag.CommandLine.DurationP("sendInterval", "i", 0, "Interval for full data synchronization")
	}
	if pflag.CommandLine.Lookup("cooldown") == nil {
		pflag.CommandLine.DurationP("cooldown", "s", 0, "Minimum interval between synchronizations")
	}
	pflag.Parse()
	v := viper.New()
	v.SetDefault("address", "localhost:8080")
	v.SetDefault("protocol", "https")
	v.SetDefault("send_interval", 5*time.Minute)
	v.SetDefault("cooldown", 5*time.Second)
	v.SetDefault("PathDB", "client.db")
	v.SetEnvPrefix("client")
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
	if err := v.BindPFlag("path_db", pflag.CommandLine.Lookup("pathDB")); err != nil {
		return err
	}
	if err := v.BindPFlag("send_interval", pflag.CommandLine.Lookup("sendInterval")); err != nil {
		return err
	}
	if err := v.BindPFlag("cooldown", pflag.CommandLine.Lookup("cooldown")); err != nil {
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
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.TextUnmarshallerHookFunc(),
		),
	))
	if err != nil {
		return err
	}
	return nil
}

// Validate проверяет конфигурацию клиента.
func (c *ConfigClient) Validate() error {
	if c.Net.Host == "" {
		return errors.New("хост не может быть пустым")
	}
	if c.Net.Port <= 0 || c.Net.Port > 65535 {
		return errors.New("передан неверный порт")
	}
	if c.Protocol != "grpc" && c.Protocol != "https" {
		return errors.New("неверный протокол обмена")
	}
	if c.SendInterval <= 0 || c.Cooldown <= 0 {
		return errors.New("интервалы работы воркера должны быть больше нуля")
	}
	if c.Cooldown > c.SendInterval {
		return errors.New("минимальный интервал не может быть больше основного интервала синхронизации")
	}
	if c.PathDB == "" {
		return errors.New("путь к базе данных не может быть пустым")
	}
	return nil
}
