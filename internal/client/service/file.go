package service

import "os"

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

func (s *ClientService) Logout() {
	s.key = nil
	s.sender.Logout()
}
