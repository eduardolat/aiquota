package credentials

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Credentials contains API keys and account information read from auth.json.
type Credentials struct {
	CopilotAPIKey  *string `json:"copilotApiKey,omitempty"`
	ZAIAPIKey      *string `json:"zaiApiKey,omitempty"`
	CodexAPIKey    *string `json:"codexApiKey,omitempty"`
	CodexAccountID *string `json:"codexAccountId,omitempty"`
}

type authFileConfig struct {
	ZAICodingPlan struct {
		Key *string `json:"key"`
	} `json:"zai-coding-plan"`
	GitHubCopilot struct {
		Access *string `json:"access"`
	} `json:"github-copilot"`
	OpenAI struct {
		Access    *string `json:"access"`
		AccountID *string `json:"accountId"`
	} `json:"openai"`
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

	var config authFileConfig
	if err := json.Unmarshal(content, &config); err != nil {
		return Credentials{}, fmt.Errorf("failed to read auth file. please ensure it exists and is properly formatted. error details: %w", err)
	}

	creds := Credentials{
		ZAIAPIKey:      config.ZAICodingPlan.Key,
		CopilotAPIKey:  config.GitHubCopilot.Access,
		CodexAPIKey:    config.OpenAI.Access,
		CodexAccountID: config.OpenAI.AccountID,
	}

	return creds, nil
}
