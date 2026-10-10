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
	salt, err := crypto.GenerateSalt()
	if err != nil {
		return err
	}
	err = s.sender.Register(ctx, login, password, salt)
	if err != nil {
		return err
	}
	key, err := crypto.GenerateKey(salt, password)
	if err != nil {
		return err
	}
	s.key = key
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
	salt, err := s.sender.Auth(ctx, login, password)
	if err != nil {
		return err
	}
	key, err := crypto.GenerateKey(salt, password)
	if err != nil {
		return err
	}
	s.key = key
	s.syncer.RunSync()
	return nil
}
