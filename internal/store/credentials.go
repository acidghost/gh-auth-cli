package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/zalando/go-keyring"
)

const keyringService = "gh-auth-cli"

var ErrSecretNotFound = errors.New("secret not found")

type StoredToken struct {
	Version          int       `json:"version"`
	AccessToken      string    `json:"access_token"`
	TokenType        string    `json:"token_type"`
	AccessExpiresAt  time.Time `json:"access_expires_at,omitempty"`
	RefreshToken     string    `json:"refresh_token,omitempty"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at,omitempty"`
	GitHubHost       string    `json:"github_host"`
	UserID           int64     `json:"user_id"`
	Login            string    `json:"login"`
}

func ClientSecretAccount(profileID string) string {
	return "client-secret:" + profileID
}

func TokenAccount(profileID string, userID int64) string {
	return fmt.Sprintf("oauth-token:%s:%d", profileID, userID)
}

func SetClientSecret(profileID, secret string) error {
	if err := keyring.Set(keyringService, ClientSecretAccount(profileID), secret); err != nil {
		return fmt.Errorf("store client secret in keyring: %w", err)
	}
	return nil
}

func GetClientSecret(profileID string) (string, error) {
	secret, err := keyring.Get(keyringService, ClientSecretAccount(profileID))
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrSecretNotFound
	}
	if err != nil {
		return "", fmt.Errorf("read client secret from keyring: %w", err)
	}
	return secret, nil
}

func DeleteClientSecret(profileID string) error {
	err := keyring.Delete(keyringService, ClientSecretAccount(profileID))
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete client secret from keyring: %w", err)
	}
	return nil
}

func SetToken(profileID string, token *StoredToken) error {
	// #nosec G117 -- the token envelope is intentionally serialized directly
	// into the OS keyring, never into the plaintext config file.
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("encode token: %w", err)
	}
	if err := keyring.Set(keyringService, TokenAccount(profileID, token.UserID), string(data)); err != nil {
		return fmt.Errorf("store token in keyring: %w", err)
	}
	return nil
}

func GetToken(profileID string, userID int64) (*StoredToken, error) {
	value, err := keyring.Get(keyringService, TokenAccount(profileID, userID))
	if errors.Is(err, keyring.ErrNotFound) {
		return nil, ErrSecretNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read token from keyring: %w", err)
	}
	var token StoredToken
	if err := json.Unmarshal([]byte(value), &token); err != nil {
		return nil, fmt.Errorf("decode stored token: %w", err)
	}
	if token.Version != 1 || token.AccessToken == "" || token.UserID != userID {
		return nil, errors.New("stored token is invalid or uses an unsupported version")
	}
	return &token, nil
}

func DeleteToken(profileID string, userID int64) error {
	err := keyring.Delete(keyringService, TokenAccount(profileID, userID))
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete token from keyring: %w", err)
	}
	return nil
}
