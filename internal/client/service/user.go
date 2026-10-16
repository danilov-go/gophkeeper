package service

import (
	"context"
	"errors"

	"github.com/awnumar/memguard"

	"github.com/danilov-go/gophkeeper/internal/crypto"
	"github.com/danilov-go/gophkeeper/internal/models"
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
	key, hash, err := crypto.GenerateKey(salt, password)
	if err != nil {
		return err
	}
	if err = s.sender.Register(ctx, login, hash, salt); err != nil {
		return err
	}
	if err = s.storage.SaveUser(ctx, login, hash, salt); err != nil {
		return err
	}
	s.mu.Lock()
	s.key = memguard.NewBufferFromBytes(key)
	s.mu.Unlock()
	s.trigger()
	return nil
}

func (s *ClientService) Login(ctx context.Context, login, password string) error {
	var offline bool
	var salt string
	var localUser models.LocalUser
	if err := ctx.Err(); err != nil {
		return err
	}
	if login == "" || password == "" {
		return errors.New("пустой логин или пароль")
	}
	serverSalt, err := s.sender.GetSalt(ctx, login)
	if err != nil {
		var storeErr error
		localUser, storeErr = s.storage.GetUser(ctx, login)
		if storeErr != nil {
			return errors.New("пользователь не найден")
		}
		salt = localUser.Salt
		offline = true
	} else {
		salt = serverSalt
	}
	key, hash, err := crypto.GenerateKey(salt, password)
	if err != nil {
		return err
	}
	if !offline {
		if err = s.sender.Auth(ctx, login, hash); err != nil {
			return err
		}
		if err = s.storage.SaveUser(ctx, login, hash, salt); err != nil {
		}
	} else {
		if localUser.Hash != hash {
			return errors.New("неверный пароль")
		}
	}
	s.mu.Lock()
	s.key = memguard.NewBufferFromBytes(key)
	s.mu.Unlock()
	s.key.TryLock()
	return nil
}
