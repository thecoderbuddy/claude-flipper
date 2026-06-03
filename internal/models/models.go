package models

import "time"

// OAuthAccount mirrors the oauthAccount object inside ~/.claude.json.
type OAuthAccount struct {
	EmailAddress     string `json:"emailAddress"`
	OrganizationUUID string `json:"organizationUuid"`
	OrganizationName string `json:"organizationName"`
	UUID             string `json:"uuid"`
}

// AccountConfig is what flipper backs up per slot: the oauthAccount section
// plus the userID field, both of which are account-specific in ~/.claude.json.
type AccountConfig struct {
	OAuthAccount OAuthAccount `json:"oauthAccount"`
	UserID       string       `json:"userID,omitempty"`
}

// ClaudeCredentials mirrors the content of ~/.claude/.credentials.json (Linux/Windows)
// and the Keychain entry (macOS).
type ClaudeCredentials struct {
	ClaudeAiOauth OAuthToken `json:"claudeAiOauth"`
}

// OAuthToken holds the raw token fields stored by Claude Code.
type OAuthToken struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    int64  `json:"expiresAt"`
}

// AccountRecord is a single entry in sequence.json's accounts map.
type AccountRecord struct {
	Email            string    `json:"email"`
	UUID             string    `json:"uuid"`
	OrganizationUUID string    `json:"organizationUuid"`
	OrganizationName string    `json:"organizationName"`
	AddedAt          time.Time `json:"addedAt"`
}

// Sequence is the top-level structure of ~/.claude-flipper/sequence.json.
type Sequence struct {
	ActiveSlot int                      `json:"activeSlot"`
	Accounts   map[string]AccountRecord `json:"accounts"` // key = slot number as string
	Sequence   []int                    `json:"sequence"`
}
