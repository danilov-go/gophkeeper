package service

import (
	"context"
	"errors"

	"github.com/danilov-go/gophkeeper/internal/client/crypto"
)

func (s *ClientService) Register(ctx context.Context, login, password string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if login == "" || password == "" {
		return errors.New("пустой логин или пароль")
	}
	err := s.sender.Register(ctx, login, password)
	if err != nil {
		return err
	}
	s.key = crypto.GenerateKey(login, password)
	s.syncer.RunSync()
	return nil
}

func (s *ClientService) Login(ctx context.Context, login, password string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if login == "" || password == "" {
		return errors.New("пустой логин или пароль")
	}
	err := s.sender.Auth(ctx, login, password)
	if err != nil {
		return err
	}
	s.key = crypto.GenerateKey(login, password)
	s.syncer.RunSync()
	return nil
}
