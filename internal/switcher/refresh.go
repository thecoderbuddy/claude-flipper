package switcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sharvari/claude-flipper/internal/models"
)

const (
	oauthTokenURL = "https://platform.claude.com/v1/oauth/token"
	// oauthClientID is the client ID Claude Code uses for its own OAuth flow.
	oauthClientID = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
)

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// tryRefreshCredentials refreshes the access token in creds if it has expired
// (or will expire within the next 60 seconds). Returns updated credentials on
// success, or the original credentials unchanged if the refresh token is
// revoked or the network is unavailable.
func tryRefreshCredentials(creds *models.ClaudeCredentials) (*models.ClaudeCredentials, error) {
	token := creds.ClaudeAiOauth

	// Skip refresh if the access token is still valid.
	expiresAt := time.UnixMilli(token.ExpiresAt)
	if time.Now().Before(expiresAt.Add(-60 * time.Second)) {
		return creds, nil
	}

	if token.RefreshToken == "" {
		return creds, fmt.Errorf("access token expired and no refresh token available")
	}

	body, _ := json.Marshal(map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": token.RefreshToken,
		"client_id":     oauthClientID,
	})

	req, err := http.NewRequest(http.MethodPost, oauthTokenURL, bytes.NewReader(body))
	if err != nil {
		return creds, fmt.Errorf("build refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return creds, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return creds, fmt.Errorf("refresh token rejected (status %d) — re-login required", resp.StatusCode)
	}

	var r refreshResponse
	if err := json.Unmarshal(data, &r); err != nil || r.AccessToken == "" {
		return creds, fmt.Errorf("invalid refresh response")
	}

	updated := *creds
	updated.ClaudeAiOauth.AccessToken = r.AccessToken
	if r.RefreshToken != "" {
		updated.ClaudeAiOauth.RefreshToken = r.RefreshToken
	}
	if r.ExpiresIn > 0 {
		updated.ClaudeAiOauth.ExpiresAt = time.Now().Add(time.Duration(r.ExpiresIn) * time.Second).UnixMilli()
	}
	return &updated, nil
}
