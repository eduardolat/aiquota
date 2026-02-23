package credentials

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tidwall/gjson"
)

// Credentials contains API keys and account information read from auth.json.
type Credentials struct {
	CopilotAPIKey  *string `json:"copilotApiKey,omitempty"`
	ZAIAPIKey      *string `json:"zaiApiKey,omitempty"`
	CodexAPIKey    *string `json:"codexApiKey,omitempty"`
	CodexAccountID *string `json:"codexAccountId,omitempty"`
}

// GetCredentials reads API keys and account information from OpenCode auth.json.
func GetCredentials() (Credentials, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Credentials{}, fmt.Errorf("failed to resolve user home directory: %w", err)
	}

	authFilePath := filepath.Join(home, ".local", "share", "opencode", "auth.json")
	content, err := os.ReadFile(authFilePath)
	if err != nil {
		return Credentials{}, fmt.Errorf("failed to read auth file. please ensure it exists and is properly formatted. error details: %w", err)
	}

	if !gjson.ValidBytes(content) {
		return Credentials{}, fmt.Errorf("failed to read auth file. please ensure it exists and is properly formatted. error details: invalid JSON")
	}

	creds := Credentials{
		ZAIAPIKey:      optionalString(gjson.GetBytes(content, "zai-coding-plan.key")),
		CopilotAPIKey:  optionalString(gjson.GetBytes(content, "github-copilot.access")),
		CodexAPIKey:    optionalString(gjson.GetBytes(content, "openai.access")),
		CodexAccountID: optionalString(gjson.GetBytes(content, "openai.accountId")),
	}

	return creds, nil
}

func optionalString(result gjson.Result) *string {
	if !result.Exists() || result.Type == gjson.Null {
		return nil
	}

	value := result.String()
	return &value
}
