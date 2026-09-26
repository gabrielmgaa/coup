package tui

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

type Session struct {
	Name   string `json:"name"`
	Server string `json:"server"`
	Room   string `json:"room,omitempty"`
	Token  string `json:"token,omitempty"`
}

func SessionPath() (string, error) {
	folder, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(folder, "coup", "session.json"), nil
}

func LoadSession(path string) (Session, error) {
	encoded, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Session{}, nil
	}
	if err != nil {
		return Session{}, err
	}
	var saved Session
	err = json.Unmarshal(encoded, &saved)
	return saved, err
}

func (s Session) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, encoded, 0o600)
}
