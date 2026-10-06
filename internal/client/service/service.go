package service

import (
	"context"
	"encoding/json"
	"os"

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
	encryptedBody, err := crypto.Encrypt(s.key, body)
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
func (s *ClientService) Update(ctx context.Context, id int, meta models.MetaData, data models.SecretData) error {
	dataBody, err := json.Marshal(data)
	if err != nil {
		return err
	}
	metaBody, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	p := payload{
		Meta: metaBody,
		Data: dataBody,
	}
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}

	encryptedBody, err := crypto.Encrypt(s.key, body)
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
func (s *ClientService) GetAll(ctx context.Context, sType models.SecretType) ([]models.Secret, error) {
	cipherData, err := s.storage.GetAllSecret(ctx, sType)
	if err != nil {
		return nil, err
	}
	data := make([]models.Secret, 0, len(cipherData))
	for _, cs := range cipherData {
		decryptedData, err := crypto.Decrypt(s.key, cs.Cipher)
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
	decryptedData, err := crypto.Decrypt(s.key, cipherData.Cipher)
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

func (s *ClientService) GetFile(path string) ([]byte, error) {
	if path == "" {
		path = "."
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *ClientService) ExportFile(path string, data []byte) error {
	if path == "" {
		path = "."
	}
	return os.WriteFile(path, data, 0600)
}
