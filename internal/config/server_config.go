package config

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// NetAddress определяет адрес сервера.
type NetAddress struct {
	Host string
	Port int
}

// String возвращает строковое представление сетевого адреса в формате host:port.
func (n NetAddress) String() string {
	return n.Host + ":" + strconv.Itoa(n.Port)
}

// UnmarshalText десериализует сетевой адрес из текстового формата для библиотеки env.
func (n *NetAddress) UnmarshalText(adr []byte) error {
	return n.Set(string(adr))
}

// Set парсит строку в формате host:port и валидирует ее для пакета flag.
func (n *NetAddress) Set(s string) error {
	hp := strings.Split(s, ":")
	if len(hp) != 2 {
		return errors.New("need address in a form host:port")
	}
	port, err := strconv.Atoi(hp[1])
	if err != nil {
		return err
	}
	n.Host = hp[0]
	n.Port = port
	return nil
}

// UnmarshalJSON десериализует сетевой адрес из JSON формата.
func (n *NetAddress) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	return n.Set(str)
}

// ConfigServer описывает настройки сервера.
type ConfigServer struct {
	Net        NetAddress `mapstructure:"ADDRESS"`
	ConfigPath string     `mapstructure:"CONFIG_PATH"`
}

// Get парсит конфигурацию сервера.
func (c *ConfigServer) Get() error {
	if pflag.CommandLine.Lookup("address") == nil {
		pflag.CommandLine.StringP("address", "a", "", "Address and port to run server")
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
