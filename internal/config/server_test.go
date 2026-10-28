package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigServer_Get(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	tests := []struct {
		name     string
		args     []string
		envSetup map[string]string
		want     ConfigServer
	}{
		{
			name: "приоритет флагов командной строки",
			args: []string{
				"cmd",
				"-a", "127.0.0.1:9090",
				"-p", "https",
				"-d", "flag_dsn",
				"-t", "flag_token",
				"-r", "flag.crt",
				"-k", "flag.key",
			},
			envSetup: map[string]string{
				"SERVER_ADDRESS":      "10.0.0.1:8080",
				"SERVER_PROTOCOL":     "grpc",
				"SERVER_DATABASE_DSN": "env_dsn",
				"SERVER_TOKEN":        "env_token",
			},
			want: ConfigServer{
				Net: NetAddress{
					Host: "127.0.0.1",
					Port: 9090,
				},
				Protocol:    "https",
				DatabaseDSN: "flag_dsn",
				Token:       "flag_token",
				PathTLSCert: "flag.crt",
				PathTLSKey:  "flag.key",
			},
		},
		{
			name: "приоритет переменных окружения",
			args: []string{"cmd"},
			envSetup: map[string]string{
				"SERVER_ADDRESS":       "10.0.0.1:8080",
				"SERVER_PROTOCOL":      "grpc",
				"SERVER_DATABASE_DSN":  "env_dsn",
				"SERVER_TOKEN":         "env_token",
				"SERVER_PATH_TLS_CERT": "env.crt",
				"SERVER_PATH_TLS_KEY":  "env.key",
			},
			want: ConfigServer{
				Net: NetAddress{
					Host: "10.0.0.1",
					Port: 8080,
				},
				Protocol:    "grpc",
				DatabaseDSN: "env_dsn",
				Token:       "env_token",
				PathTLSCert: "env.crt",
				PathTLSKey:  "env.key",
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
			var cfg ConfigServer
			err := cfg.Get()
			assert.NoError(t, err)
			assert.Equal(t, tt.want, cfg)
		})
	}
}

func TestConfigServer_Validate(t *testing.T) {
	testDir := t.TempDir()
	certPath := filepath.Join(testDir, "server.crt")
	keyPath := filepath.Join(testDir, "server.key")
	err := os.WriteFile(certPath, []byte("cert-data"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(keyPath, []byte("key-data"), 0644)
	require.NoError(t, err)
	config := ConfigServer{
		Net: NetAddress{
			Host: "localhost",
			Port: 8080,
		},
		Protocol:    "https",
		DatabaseDSN: "postgres/db",
		Token:       "jwt_secret_token",
		PathTLSCert: certPath,
		PathTLSKey:  keyPath,
	}
	tests := []struct {
		name   string
		set    func(cfg *ConfigServer)
		expErr string
	}{
		{
			name:   "положительный тест (https)",
			set:    func(cfg *ConfigServer) {},
			expErr: "",
		},
		{
			name: "положительнный тест (grpc)",
			set: func(cfg *ConfigServer) {
				cfg.Protocol = "grpc"
			},
			expErr: "",
		},
		{
			name: "пустой хост",
			set: func(cfg *ConfigServer) {
				cfg.Net.Host = ""
			},
			expErr: "хост не может быть пустым",
		},
		{
			name: "порт равен нулю",
			set: func(cfg *ConfigServer) {
				cfg.Net.Port = 0
			},
			expErr: "передан неверный порт",
		},
		{
			name: "порт выше 65535",
			set: func(cfg *ConfigServer) {
				cfg.Net.Port = 65536
			},
			expErr: "передан неверный порт",
		},
		{
			name: "неверный протокол",
			set: func(cfg *ConfigServer) {
				cfg.Protocol = "http"
			},
			expErr: "неверный протокол обмена",
		},
		{
			name: "пустая строка подключения к БД",
			set: func(cfg *ConfigServer) {
				cfg.DatabaseDSN = ""
			},
			expErr: "строка подключения к базе данных не может быть пустой",
		},
		{
			name: "пустой токен",
			set: func(cfg *ConfigServer) {
				cfg.Token = ""
			},
			expErr: "токен не может быть пустым",
		},
		{
			name: "слишком короткий токен",
			set: func(cfg *ConfigServer) {
				cfg.Token = "token"
			},
			expErr: "передан короткий токен, для безопасности требуется не менее 16 символов",
		},
		{
			name: "пустой путь к TLS-сертификату",
			set: func(cfg *ConfigServer) {
				cfg.PathTLSCert = ""
			},
			expErr: "путь к TLS-сертификату не может быть пустым",
		},
		{
			name: "пустой путь к TLS-ключу",
			set: func(cfg *ConfigServer) {
				cfg.PathTLSKey = ""
			},
			expErr: "путь к TLS-ключу не может быть пустым",
		},
		{
			name: "файла с TLS-сертификата не существует",
			set: func(cfg *ConfigServer) {
				cfg.PathTLSCert = filepath.Join(testDir, "unknow.crt")
			},
			expErr: "файл TLS-сертификата не найден",
		},
		{
			name: "файла с TLS-ключом не существует",
			set: func(cfg *ConfigServer) {
				cfg.PathTLSKey = filepath.Join(testDir, "unknow.key")
			},
			expErr: "файл TLS-ключа не найден",
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
