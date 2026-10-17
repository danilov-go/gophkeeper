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
func (s *ClientService) Save(ctx context.Context, meta models.MetaData, data models.SecretData) (int, error) {
	dataBody, err := json.Marshal(data)
	if err != nil {
		return 0, err
	}
	metaBody, err := json.Marshal(meta)
	if err != nil {
		return 0, err
	}
	p := payload{
		Meta: metaBody,
		Data: dataBody,
	}
	body, err := json.Marshal(p)
	if err != nil {
		return 0, err
	}
	s.mu.RLock()
	if s.key == nil {
		s.mu.RUnlock()
		return 0, errors.New("отсутствует ключ шифрования")
	}
	key := s.key.Bytes()
	s.mu.RUnlock()
	encryptedBody, err := crypto.Encrypt(key, body)
	if err != nil {
		return 0, err
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
func (s *ClientService) Update(ctx context.Context, id int, meta models.MetaData, data models.SecretData) error {
	secret, err := s.storage.GetSecret(ctx, id)
	if err != nil {
		return err
	}
	dataBody, err := json.Marshal(data)
	if err != nil {
		return err
	}
	metaBody, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	p := payload{Meta: metaBody, Data: dataBody}
	body, err := json.Marshal(p)
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

	encryptedBody, err := crypto.Encrypt(key, body)
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
func (s *ClientService) Delete(ctx context.Context, id int) error {
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
		decryptedData, err := crypto.Decrypt(key, cs.Cipher)
		if err != nil {
			return nil, err
		}
		var p payload
		if err := json.Unmarshal(decryptedData, &p); err != nil {
			return nil, err
		}
		d, err := models.NewSecretData(cs.Type, p.Data)
		if err != nil {
			return nil, err
		}
		m, err := models.NewMetaData(cs.Type, p.Meta)
		if err != nil {
			return nil, err
		}
		data = append(data, models.Secret{
			ID:   cs.ID,
			Type: cs.Type,
			Data: d,
			Meta: m,
		})
	}
	return data, nil
}

func (s *ClientService) Get(ctx context.Context, id int) (models.Secret, error) {
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
	decryptedData, err := crypto.Decrypt(key, cipherData.Cipher)
	if err != nil {
		return data, err
	}
	var p payload
	if err := json.Unmarshal(decryptedData, &p); err != nil {
		return data, err
	}
	m, err := models.NewMetaData(cipherData.Type, p.Meta)
	if err != nil {
		return data, err
	}
	d, err := models.NewSecretData(cipherData.Type, p.Data)
	if err != nil {
		return data, err
	}
	data = models.Secret{
		ID:   cipherData.ID,
		Type: cipherData.Type,
		Data: d,
		Meta: m,
	}
	return data, nil
}
