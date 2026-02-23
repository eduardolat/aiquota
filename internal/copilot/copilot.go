package copilot

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/eduardolat/aiquota/internal/credentials"
	"github.com/eduardolat/aiquota/internal/helpers"
	"github.com/tidwall/gjson"
)

const userAgent = "GitHubCopilotChat/0.35.0"

// Quota contains GitHub Copilot usage information.
type Quota struct {
	AccountUser              string  `json:"accountUser"`
	AccountType              string  `json:"accountType"`
	RequestsTotal            int64   `json:"requestsTotal"`
	RequestsUsed             int64   `json:"requestsUsed"`
	RequestsUsedPercent      float64 `json:"requestsUsedPercent"`
	RequestsRemaining        int64   `json:"requestsRemaining"`
	RequestsRemainingPercent float64 `json:"requestsRemainingPercent"`
	ResetAt                  string  `json:"resetAt"`
	ResetIn                  string  `json:"resetIn"`
}

// GetQuota fetches GitHub Copilot quota information.
func GetQuota(ctx context.Context, creds credentials.Credentials) (Quota, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/copilot_internal/user", nil)
	if err != nil {
		return Quota{}, fmt.Errorf("failed to create GitHub Copilot request: %w", err)
	}

	req.Header.Set("Authorization", "token "+stringValue(creds.CopilotAPIKey))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Editor-Version", "vscode/1.107.0")
	req.Header.Set("Editor-Plugin-Version", "copilot-chat/0.35.0")
	req.Header.Set("Copilot-Integration-Id", "vscode-chat")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return Quota{}, fmt.Errorf("failed to fetch GitHub Copilot quota: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return Quota{}, fmt.Errorf("failed to read GitHub Copilot response: %w", err)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Quota{}, fmt.Errorf("failed to fetch GitHub Copilot quota. Status: %d, Response: %s", response.StatusCode, string(body))
	}

	total := gjson.GetBytes(body, "quota_snapshots.premium_interactions.entitlement").Int()
	remaining := gjson.GetBytes(body, "quota_snapshots.premium_interactions.remaining").Int()
	remainingPercent := gjson.GetBytes(body, "quota_snapshots.premium_interactions.percent_remaining").Float()
	used := max(0, total-remaining)
	usedPercent := max(0.0, 100-remainingPercent)
	resetAt := gjson.GetBytes(body, "quota_reset_date_utc").String()

	return Quota{
		AccountUser:              gjson.GetBytes(body, "login").String(),
		AccountType:              gjson.GetBytes(body, "access_type_sku").String(),
		RequestsTotal:            total,
		RequestsUsed:             used,
		RequestsUsedPercent:      usedPercent,
		RequestsRemaining:        remaining,
		RequestsRemainingPercent: remainingPercent,
		ResetAt:                  resetAt,
		ResetIn:                  helpers.FormatTimeUntil(resetAt),
	}, nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
