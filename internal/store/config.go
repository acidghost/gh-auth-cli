package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const configVersion = 1

var ErrNotConfigured = errors.New("gh-auth-cli is not configured")

type Config struct {
	Version      int    `json:"version"`
	ProfileID    string `json:"profile_id"`
	GitHubHost   string `json:"github_host"`
	ClientID     string `json:"client_id"`
	CallbackURL  string `json:"callback_url"`
	ActiveUserID int64  `json:"active_user_id,omitempty"`
	ActiveLogin  string `json:"active_login,omitempty"`
}

func DefaultConfig(clientID, callbackURL string) (*Config, error) {
	id, err := randomID()
	if err != nil {
		return nil, err
	}
	return &Config{
		Version:     configVersion,
		ProfileID:   id,
		GitHubHost:  "github.com",
		ClientID:    clientID,
		CallbackURL: callbackURL,
	}, nil
}

func ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}
	return filepath.Join(dir, "gh-auth-cli", "config.json"), nil
}

func LoadConfig() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	// #nosec G304 -- path is fixed beneath the OS user config directory.
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotConfigured
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	if cfg.Version != configVersion || cfg.ProfileID == "" || cfg.ClientID == "" || cfg.CallbackURL == "" {
		return nil, errors.New("config is invalid or uses an unsupported version")
	}
	return &cfg, nil
}

func DeleteConfig() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete config: %w", err)
	}
	return nil
}

func SaveConfig(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	configDir := filepath.Dir(path)
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	// #nosec G302 -- this is a directory and needs its execute bit.
	if err := os.Chmod(configDir, 0o700); err != nil {
		return fmt.Errorf("secure config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		if closeErr := tmp.Close(); closeErr != nil {
			return fmt.Errorf("secure temporary config: %w (also failed to close it: %v)", err, closeErr)
		}
		return fmt.Errorf("secure temporary config: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		if closeErr := tmp.Close(); closeErr != nil {
			return fmt.Errorf("write temporary config: %w (also failed to close it: %v)", err, closeErr)
		}
		return fmt.Errorf("write temporary config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		if closeErr := tmp.Close(); closeErr != nil {
			return fmt.Errorf("sync temporary config: %w (also failed to close it: %v)", err, closeErr)
		}
		return fmt.Errorf("sync temporary config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary config: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

func randomID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate profile id: %w", err)
	}
	return hex.EncodeToString(value[:]), nil
}
