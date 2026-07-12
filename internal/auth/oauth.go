package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/acidghost/gh-auth-cli/internal/store"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
)

const flowTimeout = 5 * time.Minute

type callbackResult struct {
	code string
	err  error
}

type githubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

func Login(ctx context.Context, cfg *store.Config, clientSecret string) (*store.StoredToken, error) {
	listenAddr, callbackPath, err := validateCallbackURL(cfg.CallbackURL)
	if err != nil {
		return nil, err
	}
	logger := zerolog.Ctx(ctx)
	logger.Debug().Str("listen_address", listenAddr).Str("callback_path", callbackPath).Msg("starting OAuth callback server")
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("listen for OAuth callback on %s: %w", listenAddr, err)
	}
	defer listener.Close()

	state := oauth2.GenerateVerifier()
	verifier := oauth2.GenerateVerifier()
	oauthCfg := oauthConfig(cfg, clientSecret)
	authURL := oauthCfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))

	result := make(chan callbackResult, 1)
	var once sync.Once
	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get("state") != state {
			http.Error(w, "invalid OAuth state", http.StatusBadRequest)
			return
		}
		var callbackErr error
		code := req.URL.Query().Get("code")
		if oauthErr := req.URL.Query().Get("error"); oauthErr != "" {
			callbackErr = fmt.Errorf("GitHub denied authorization: %s", oauthErr)
		} else if code == "" {
			callbackErr = errors.New("OAuth callback did not contain an authorization code")
		}
		once.Do(func() { result <- callbackResult{code: code, err: callbackErr} })
		if callbackErr != nil {
			http.Error(w, callbackErr.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("Authentication complete. You may close this window.\n"))
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	serverErrors := make(chan error, 1)
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			serverErrors <- serveErr
		}
	}()

	fmt.Fprintf(os.Stderr, "Open this URL to authorize the GitHub App:\n%s\n", authURL)
	logger.Debug().Msg("launching browser for GitHub authorization")
	if err := openBrowser(authURL); err != nil {
		logger.Debug().Err(err).Msg("automatic browser launch failed")
		fmt.Fprintf(os.Stderr, "Could not open a browser automatically: %v\n", err)
	}

	flowCtx, cancel := context.WithTimeout(ctx, flowTimeout)
	defer cancel()
	var received callbackResult
	select {
	case received = <-result:
	case serveErr := <-serverErrors:
		return nil, fmt.Errorf("OAuth callback server: %w", serveErr)
	case <-flowCtx.Done():
		return nil, errors.New("timed out waiting for GitHub authorization")
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
	if received.err != nil {
		return nil, received.err
	}

	logger.Debug().Msg("received valid OAuth callback; exchanging authorization code")
	receivedAt := time.Now()
	token, err := oauthCfg.Exchange(flowCtx, received.code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("exchange authorization code: %w", err)
	}
	logger.Debug().Time("access_expires_at", token.Expiry).Bool("has_refresh_token", token.RefreshToken != "").Msg("authorization code exchanged")
	user, err := fetchUser(flowCtx, token)
	if err != nil {
		return nil, err
	}
	return storedToken(token, receivedAt, cfg.GitHubHost, user, time.Time{}), nil
}

func Refresh(ctx context.Context, cfg *store.Config, clientSecret string, current *store.StoredToken) (*store.StoredToken, error) {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Time("access_expires_at", current.AccessExpiresAt).Time("refresh_expires_at", current.RefreshExpiresAt).Msg("preparing OAuth token refresh")
	if current.RefreshToken == "" {
		return nil, errors.New("access token expired and no refresh token is available; run login again")
	}
	if !current.RefreshExpiresAt.IsZero() && !current.RefreshExpiresAt.After(time.Now()) {
		return nil, errors.New("refresh token expired; run login again")
	}
	old := &oauth2.Token{
		AccessToken:  current.AccessToken,
		TokenType:    current.TokenType,
		RefreshToken: current.RefreshToken,
		Expiry:       time.Now().Add(-time.Minute),
	}
	receivedAt := time.Now()
	fresh, err := oauthConfig(cfg, clientSecret).TokenSource(ctx, old).Token()
	if err != nil {
		return nil, fmt.Errorf("refresh GitHub token: %w", err)
	}
	// OAuth servers may omit refresh_token when it has not changed.
	if fresh.RefreshToken == "" {
		fresh.RefreshToken = current.RefreshToken
	}
	logger.Debug().Time("access_expires_at", fresh.Expiry).Msg("OAuth token refresh completed")
	user := &githubUser{ID: current.UserID, Login: current.Login}
	return storedToken(fresh, receivedAt, current.GitHubHost, user, current.RefreshExpiresAt), nil
}

func Usable(token *store.StoredToken, minValidity time.Duration) bool {
	if token.AccessToken == "" {
		return false
	}
	if token.AccessExpiresAt.IsZero() {
		return true
	}
	return token.AccessExpiresAt.After(time.Now().Add(minValidity + 30*time.Second))
}

func oauthConfig(cfg *store.Config, clientSecret string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: clientSecret,
		RedirectURL:  cfg.CallbackURL,
		Endpoint: oauth2.Endpoint{
			AuthURL: "https://github.com/login/oauth/authorize",
			// #nosec G101 -- this is a public OAuth endpoint, not a credential.
			TokenURL:  "https://github.com/login/oauth/access_token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
}

func fetchUser(ctx context.Context, token *oauth2.Token) (*githubUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("create GitHub user request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "gh-auth-cli")
	response, err := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token)).Do(req)
	if err != nil {
		return nil, fmt.Errorf("get authenticated GitHub user: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get authenticated GitHub user: GitHub returned %s", response.Status)
	}
	var user githubUser
	if err := json.NewDecoder(response.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode authenticated GitHub user: %w", err)
	}
	if user.ID == 0 || user.Login == "" {
		return nil, errors.New("GitHub returned an invalid user identity")
	}
	return &user, nil
}

func storedToken(token *oauth2.Token, receivedAt time.Time, host string, user *githubUser, previousRefreshExpiry time.Time) *store.StoredToken {
	refreshExpiry := durationExpiry(token.Extra("refresh_token_expires_in"), receivedAt)
	if refreshExpiry.IsZero() && token.RefreshToken != "" {
		refreshExpiry = previousRefreshExpiry
	}
	return &store.StoredToken{
		Version:          1,
		AccessToken:      token.AccessToken,
		TokenType:        token.TokenType,
		AccessExpiresAt:  token.Expiry,
		RefreshToken:     token.RefreshToken,
		RefreshExpiresAt: refreshExpiry,
		GitHubHost:       host,
		UserID:           user.ID,
		Login:            user.Login,
	}
}

func durationExpiry(value any, receivedAt time.Time) time.Time {
	var seconds int64
	switch typed := value.(type) {
	case float64:
		seconds = int64(typed)
	case json.Number:
		seconds, _ = typed.Int64()
	case string:
		seconds, _ = strconv.ParseInt(typed, 10, 64)
	case int64:
		seconds = typed
	case int:
		seconds = int64(typed)
	}
	if seconds <= 0 {
		return time.Time{}
	}
	return receivedAt.Add(time.Duration(seconds) * time.Second)
}

func ValidateCallbackURL(raw string) error {
	_, _, err := validateCallbackURL(raw)
	return err
}

func validateCallbackURL(raw string) (string, string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", "", fmt.Errorf("parse callback URL: %w", err)
	}
	if parsed.Scheme != "http" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", "", errors.New("callback URL must be a plain HTTP loopback URL without query or fragment")
	}
	hostname := parsed.Hostname()
	if hostname != "127.0.0.1" && hostname != "localhost" && hostname != "::1" {
		return "", "", errors.New("callback URL must use a loopback host")
	}
	if parsed.Port() == "" {
		return "", "", errors.New("callback URL must include a fixed port")
	}
	if parsed.Path == "" || parsed.Path == "/" {
		return "", "", errors.New("callback URL must include a callback path")
	}
	return parsed.Host, parsed.Path, nil
}

func openBrowser(target string) error {
	if browser := os.Getenv("BROWSER"); browser != "" {
		return startCommand(browser, target)
	}
	switch runtime.GOOS {
	case "darwin":
		return startCommand("open", target)
	case "linux":
		return startCommand("xdg-open", target)
	default:
		return errors.New("automatic browser opening is unsupported on this platform")
	}
}

func startCommand(name, target string) error {
	if strings.TrimSpace(name) != name || name == "" {
		return errors.New("BROWSER must contain one executable path without arguments")
	}
	// #nosec G204,G702 -- the executable is either a fixed platform opener or the
	// explicitly configured BROWSER executable; target is passed without a shell.
	cmd := exec.Command(name, target)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
