package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/danilov-go/gophkeeper/internal/crypto"
	"github.com/danilov-go/gophkeeper/internal/models"
)

type payload struct {
	Meta json.RawMessage `json:"meta"`
	Data json.RawMessage `json:"data"`
}

// Save сохраняет секрет.
func (s *ClientService) Save(ctx context.Context, meta models.MetaData, data models.SecretData) (string, error) {
	s.mu.RLock()
	if s.key == nil {
		s.mu.RUnlock()
		return "", errors.New("отсутствует ключ шифрования")
	}
	key := s.key.Bytes()
	s.mu.RUnlock()
	encryptedBody, err := s.encrypt(key, meta, data)
	if err != nil {
		return "", err
	}
	cipherData := models.CipherData{
		Type:   data.SecretType(),
		Cipher: encryptedBody,
	}
	id, err := s.storage.SaveSecret(ctx, cipherData)
	if err == nil {
		s.trigger()
	}
	return id, err
}

// Update обновляет существующий секрет по его ID.
func (s *ClientService) Update(ctx context.Context, id string, meta models.MetaData, data models.SecretData) error {
	secret, err := s.storage.GetSecret(ctx, id)
	if err != nil {
		return err
	}
	s.mu.RLock()
	if s.key == nil {
		s.mu.RUnlock()
		return errors.New("отсутствует ключ шифрования")
	}
	key := s.key.Bytes()
	s.mu.RUnlock()
	encryptedBody, err := s.encrypt(key, meta, data)
	if err != nil {
		return err
	}
	cipherData := models.CipherData{
		ID:      id,
		Type:    data.SecretType(),
		Cipher:  encryptedBody,
		Version: secret.Version,
	}
	err = s.storage.UpdateSecret(ctx, cipherData)
	if err == nil {
		s.trigger()
	}
	return err
}

// Delete удаляет секрет по его ID.
func (s *ClientService) Delete(ctx context.Context, id string) error {
	err := s.storage.DeleteSecret(ctx, id)
	if err == nil {
		s.trigger()
	}
	return err

}

// GetAll возвращает список расшифрованных данных.
func (s *ClientService) GetAll(ctx context.Context, sType models.SecretType) ([]models.Secret, error) {
	cipherData, err := s.storage.GetAllSecret(ctx, sType)
	if err != nil {
		return nil, err
	}
	data := make([]models.Secret, 0, len(cipherData))
	s.mu.RLock()
	if s.key == nil {
		s.mu.RUnlock()
		return nil, errors.New("отсутствует ключ шифрования")
	}
	key := s.key.Bytes()
	s.mu.RUnlock()
	for _, cs := range cipherData {
		decryptedData, err := s.decrypt(key, cs)
		if err != nil {
			return nil, err
		}
		data = append(data, decryptedData)
	}
	return data, nil
}

func (s *ClientService) Get(ctx context.Context, id string) (models.Secret, error) {
	var data models.Secret
	cipherData, err := s.storage.GetSecret(ctx, id)
	if err != nil {
		return data, err
	}
	s.mu.RLock()
	if s.key == nil {
		s.mu.RUnlock()
		return data, errors.New("отсутствует ключ шифрования")
	}
	key := s.key.Bytes()
	s.mu.RUnlock()
	return s.decrypt(key, cipherData)
}

func (s *ClientService) decrypt(key []byte, cs models.CipherData) (models.Secret, error) {
	decryptedData, err := crypto.Decrypt(key, cs.Cipher)
	if err != nil {
		return models.Secret{}, err
	}
	var p payload
	if err := json.Unmarshal(decryptedData, &p); err != nil {
		return models.Secret{}, err
	}
	d, err := models.NewSecretData(cs.Type, p.Data)
	if err != nil {
		return models.Secret{}, err
	}
	m, err := models.NewMetaData(cs.Type, p.Meta)
	if err != nil {
		return models.Secret{}, err
	}
	return models.Secret{
		ID:   cs.ID,
		Type: cs.Type,
		Data: d,
		Meta: m,
	}, nil
}

func (s *ClientService) encrypt(key []byte, meta models.MetaData, data models.SecretData) ([]byte, error) {
	dataBody, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	metaBody, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	p := payload{
		Meta: metaBody,
		Data: dataBody,
	}
	body, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return crypto.Encrypt(key, body)
}
