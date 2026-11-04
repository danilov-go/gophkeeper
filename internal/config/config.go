// Package config управляет конфигурацией приложения.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
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

// UnmarshalText десериализует сетевой адрес из текстового формата.
func (n *NetAddress) UnmarshalText(adr []byte) error {
	return n.Set(string(adr))
}

// Set парсит строку в формате host:port и валидирует ее.
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

// PrintBuild выводит в консоль информацию о текущей сборке приложения.
func PrintBuild(w io.Writer, version, date, commit string) {
	fmt.Fprintf(w, "Build version: %s\n", version)
	fmt.Fprintf(w, "Build date: %s\n", date)
	fmt.Fprintf(w, "Build commit: %s\n", commit)
}
