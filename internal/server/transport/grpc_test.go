package transport

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
)

func TestGRPCServer(t *testing.T) {
	tests := []struct {
		name     string
		port     string
		pathCert string
		pathKey  string
		wantErr  bool
	}{
		{
			name:     "положительный тест",
			port:     "localhost:8080",
			pathCert: certFile,
			pathKey:  keyFile,
			wantErr:  false,
		},
		{
			name:     "неверный адрес",
			port:     "invalid port",
			pathCert: certFile,
			pathKey:  keyFile,
			wantErr:  true,
		},
		{
			name:     "ошибка отсутствия сертификатов",
			port:     "localhost:8080",
			pathCert: "test.crt",
			pathKey:  "test.key",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			baseGRPCServer := grpc.NewServer()
			serv := NewGRPC(tt.port, logger.Sugar(), baseGRPCServer, tt.pathCert, tt.pathKey)
			if tt.wantErr {
				err := serv.Run()
				assert.Error(t, err)
			} else {
				go func() {
					_ = serv.Run()
				}()
				time.Sleep(5 * time.Millisecond)
				err := serv.Stop()
				assert.NoError(t, err)
			}
		})
	}
}
