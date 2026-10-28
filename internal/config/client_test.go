package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigClient_Get(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	tests := []struct {
		name     string
		args     []string
		envSetup map[string]string
		want     ConfigClient
	}{
		{
			name: "приоритет флагов командной строки",
			args: []string{
				"cmd",
				"-a", "127.0.0.1:9090",
				"-p", "https",
				"-d", "flag.db",
				"-t", "flag.crt",
				"-i", "1m",
				"-s", "3s",
			},
			envSetup: map[string]string{
				"CLIENT_ADDRESS":       "10.0.0.1:8080",
				"CLIENT_PROTOCOL":      "grpc",
				"CLIENT_PATH_DB":       "env.db",
				"CLIENT_PATH_TLS":      "env.crt",
				"CLIENT_SEND_INTERVAL": "10m",
				"CLIENT_COOLDOWN":      "30s",
			},
			want: ConfigClient{
				Net: NetAddress{
					Host: "127.0.0.1",
					Port: 9090,
				},
				Protocol:     "https",
				PathDB:       "flag.db",
				PathTLS:      "flag.crt",
				SendInterval: 1 * time.Minute,
				Cooldown:     3 * time.Second,
			},
		},
		{
			name: "приоритет переменных окружения",
			args: []string{
				"cmd",
			},
			envSetup: map[string]string{
				"CLIENT_ADDRESS":       "10.0.0.1:8080",
				"CLIENT_PROTOCOL":      "grpc",
				"CLIENT_PATH_DB":       "env.db",
				"CLIENT_PATH_TLS":      "env.crt",
				"CLIENT_SEND_INTERVAL": "10m",
				"CLIENT_COOLDOWN":      "30s",
			},
			want: ConfigClient{
				Net: NetAddress{
					Host: "10.0.0.1",
					Port: 8080,
				},
				Protocol:     "grpc",
				PathDB:       "env.db",
				PathTLS:      "env.crt",
				SendInterval: 10 * time.Minute,
				Cooldown:     30 * time.Second,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
			for k, v := range tt.envSetup {
				t.Setenv(k, v)
			}
			os.Args = tt.args
			var cfg ConfigClient
			err := cfg.Get()
			assert.NoError(t, err)
			assert.Equal(t, tt.want, cfg)
		})
	}
}

func TestConfigClient_Validate(t *testing.T) {
	testDir := t.TempDir()
	certPath := filepath.Join(testDir, "ca.crt")
	err := os.WriteFile(certPath, []byte("cert-data"), 0644)
	require.NoError(t, err)
	config := ConfigClient{
		Net: NetAddress{
			Host: "127.0.0.1",
			Port: 8080,
		},
		Protocol:     "https",
		SendInterval: 5 * time.Minute,
		Cooldown:     5 * time.Second,
		PathTLS:      certPath,
		PathDB:       "client.db",
	}
	tests := []struct {
		name   string
		set    func(cfg *ConfigClient)
		expErr string
	}{
		{
			name:   "положительный тест (https)",
			set:    func(cfg *ConfigClient) {},
			expErr: "",
		},
		{
			name: "положительный тест (grpc)",
			set: func(cfg *ConfigClient) {
				cfg.Protocol = "grpc"
			},
			expErr: "",
		},
		{
			name: "пустой хост",
			set: func(cfg *ConfigClient) {
				cfg.Net.Host = ""
			},
			expErr: "хост не может быть пустым",
		},
		{
			name: "порт равен нулю",
			set: func(cfg *ConfigClient) {
				cfg.Net.Port = 0
			},
			expErr: "передан неверный порт",
		},
		{
			name: "порт выше 65535",
			set: func(cfg *ConfigClient) {
				cfg.Net.Port = 65536
			},
			expErr: "передан неверный порт",
		},
		{
			name: "неверный протокол",
			set: func(cfg *ConfigClient) {
				cfg.Protocol = "http"
			},
			expErr: "неверный протокол обмена",
		},
		{
			name: "SendInterval равен нулю",
			set: func(cfg *ConfigClient) {
				cfg.SendInterval = 0
			},
			expErr: "интервалы работы воркера должны быть больше нуля",
		},
		{
			name: "Cooldown равен нулю",
			set: func(cfg *ConfigClient) {
				cfg.Cooldown = 0
			},
			expErr: "интервалы работы воркера должны быть больше нуля",
		},
		{
			name: "Cooldown больше чем SendInterval",
			set: func(cfg *ConfigClient) {
				cfg.SendInterval = 5 * time.Second
				cfg.Cooldown = 10 * time.Second
			},
			expErr: "минимальный интервал не может быть больше основного интервала синхронизации",
		},
		{
			name: "пустой путь к TLS-сертификату",
			set: func(cfg *ConfigClient) {
				cfg.PathTLS = ""
			},
			expErr: "путь к TLS-сертификату не может быть пустым",
		},
		{
			name: "пустой путь к базе данных",
			set: func(cfg *ConfigClient) {
				cfg.PathDB = ""
			},
			expErr: "путь к базе данных не может быть пустым",
		},
		{
			name: "файл TLS-сертификата не существует",
			set: func(cfg *ConfigClient) {
				cfg.PathTLS = filepath.Join(testDir, "non_existent.crt")
			},
			expErr: "файл TLS-сертификата не найден",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config
			tt.set(&cfg)
			err := cfg.Validate()
			if tt.expErr != "" {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
