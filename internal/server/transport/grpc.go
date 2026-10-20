package transport

import (
	"crypto/tls"
	"net"

	"google.golang.org/grpc"
)

// GRPCServer определяет конфигурацию gRPC-сервера.
type GRPCServer struct {
	Server      *grpc.Server
	Listener    net.Listener
	Logger      log
	Address     string
	PathTLSCert string
	PathTLSKey  string
}

// NewGRPC создает новый экземпляр gRPC-сервера.
func NewGRPC(address string, l log, s *grpc.Server, pathCert, pathKey string) *GRPCServer {
	return &GRPCServer{
		Server:      s,
		Logger:      l,
		Address:     address,
		PathTLSCert: pathCert,
		PathTLSKey:  pathKey,
	}
}

// Run запускает gRPC-сервер.
func (serv *GRPCServer) Run() error {
	listener, err := net.Listen("tcp", serv.Address)
	if err != nil {
		return err
	}
	cert, err := tls.LoadX509KeyPair(serv.PathTLSCert, serv.PathTLSKey)
	if err != nil {
		listener.Close()
		return err
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
	serv.Listener = tls.NewListener(listener, tlsConfig)
	if err := serv.Server.Serve(serv.Listener); err != nil && err != grpc.ErrServerStopped {
		return err
	}
	return nil
}

// Stop останавливает gRPC-сервер.
func (serv *GRPCServer) Stop() error {
	serv.Server.Stop()
	return nil
}
