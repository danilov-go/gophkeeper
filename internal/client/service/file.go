package service

import "os"

// GetFile считывает содержимое файла по указанному пути.
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

// ExportFile записывает бинарные данные в файл.
func (s *ClientService) ExportFile(path string, data []byte) error {
	if path == "" {
		path = "."
	}
	return os.WriteFile(path, data, 0600)
}

// Logout сбрасывает ключ шифрования в памяти и завершает сессию на сервере.
func (s *ClientService) Logout() {
	s.key = nil
	s.sender.Logout()
}
