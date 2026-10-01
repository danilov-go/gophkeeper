package service

import (
	"context"
	"encoding/json"

	"github.com/danilov-go/gophkeeper/internal/client/crypto"
	"github.com/danilov-go/gophkeeper/internal/models"
)

// Storage определяет методы для взаимодействия с хранилищем.
type Storage interface {
	SaveSecret(ctx context.Context, secret models.CipherData) (int, error)
	GetSecret(ctx context.Context, id int) (models.CipherData, error)
	GetAllSecret(ctx context.Context, sType models.SecretType) ([]models.CipherData, error)
	UpdateSecret(ctx context.Context, secret models.CipherData) error
	DeleteSecret(ctx context.Context, id int) error
}

// ClientService управляет бизнес-логикой.
type ClientService struct {
	storage Storage
	key     []byte
}

// NewClientService создает экземпляр ClientService.
func NewClientService(storage Storage, secretKey []byte) *ClientService {
	return &ClientService{
		storage: storage,
		key:     secretKey,
	}
}

// Save сохраняет секрет.
func (s *ClientService) Save(ctx context.Context, data models.SecretData) (int, error) {
	cipher, err := json.Marshal(data)
	if err != nil {
		return 0, err
	}
	encryptedBody, err := crypto.Encrypt(s.key, cipher)
	if err != nil {
		return 0, err
	}
	cipherData := models.CipherData{
		Type:   data.SecretType(),
		Cipher: encryptedBody,
	}
	return s.storage.SaveSecret(ctx, cipherData)
}

// Update обновляет существующий секрет по его ID.
func (s *ClientService) Update(ctx context.Context, id int, data models.SecretData) error {
	cipher, err := json.Marshal(data)
	if err != nil {
		return err
	}
	encryptedBody, err := crypto.Encrypt(s.key, cipher)
	if err != nil {
		return err
	}
	cipherData := models.CipherData{
		ID:     id,
		Type:   data.SecretType(),
		Cipher: encryptedBody,
	}
	return s.storage.UpdateSecret(ctx, cipherData)
}

// Delete удаляет секрет по его ID.
func (s *ClientService) Delete(ctx context.Context, id int) error {
	return s.storage.DeleteSecret(ctx, id)
}

// GetAll возвращает список расшифрованных данных.
func (s *ClientService) GetAll(ctx context.Context, sType models.SecretType) ([]models.SecretData, error) {
	cipherData, err := s.storage.GetAllSecret(ctx, sType)
	if err != nil {
		return nil, err
	}
	data := make([]models.SecretData, 0, len(cipherData))
	for _, cs := range cipherData {
		decryptedData, err := crypto.Decrypt(s.key, cs.Cipher)
		if err != nil {
			return nil, err
		}
		d, err := models.NewSecretData(cs.Type, decryptedData)
		if err != nil {
			return nil, err
		}
		data = append(data, d)
	}
	return data, nil
}
