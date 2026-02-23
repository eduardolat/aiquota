package codex

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/eduardolat/aiquota/internal/credentials"
	"github.com/eduardolat/aiquota/internal/helpers"
	"github.com/tidwall/gjson"
)

// RateLimitWindow describes a quota window.
type RateLimitWindow struct {
	UsedPercent      *float64 `json:"usedPercent"`
	RemainingPercent *float64 `json:"remainingPercent"`
	ResetAt          *string  `json:"resetAt"`
	ResetIn          *string  `json:"resetIn"`
}

// Quota contains Codex account and rate-limit usage information.
type Quota struct {
	AccountEmail             string          `json:"accountEmail"`
	AccountType              string          `json:"accountType"`
	RateLimitPrimaryWindow   RateLimitWindow `json:"rateLimitPrimaryWindow"`
	RateLimitSecondaryWindow RateLimitWindow `json:"rateLimitSecondaryWindow"`
	CodeReviewPrimaryWindow  RateLimitWindow `json:"codeReviewPrimaryWindow"`
}

// GetQuota fetches Codex usage and rate limit information.
func GetQuota(ctx context.Context, creds credentials.Credentials) (Quota, error) {
	if creds.CodexAPIKey == nil || *creds.CodexAPIKey == "" {
		return Quota{}, fmt.Errorf("missing Codex API key in credentials")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://chatgpt.com/backend-api/wham/usage", nil)
	if err != nil {
		return Quota{}, fmt.Errorf("failed to create Codex request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+*creds.CodexAPIKey)
	req.Header.Set("User-Agent", "OpenCode-Quota-Plugin/1.0")
	if creds.CodexAccountID != nil && *creds.CodexAccountID != "" {
		req.Header.Set("ChatGPT-Account-Id", *creds.CodexAccountID)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return Quota{}, fmt.Errorf("failed to fetch Codex quota: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return Quota{}, fmt.Errorf("failed to read Codex response: %w", err)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Quota{}, fmt.Errorf("failed to fetch OpenAI quota. Status: %d, Response: %s", response.StatusCode, string(body))
	}

	result := Quota{
		AccountEmail:             gjson.GetBytes(body, "email").String(),
		AccountType:              gjson.GetBytes(body, "plan_type").String(),
		RateLimitPrimaryWindow:   RateLimitWindow{},
		RateLimitSecondaryWindow: RateLimitWindow{},
		CodeReviewPrimaryWindow:  RateLimitWindow{},
	}

	primary := gjson.GetBytes(body, "rate_limit.primary_window")
	if primary.Exists() && primary.Type != gjson.Null {
		result.RateLimitPrimaryWindow = parseWindow(primary)
	}

	secondary := gjson.GetBytes(body, "rate_limit.secondary_window")
	if secondary.Exists() && secondary.Type != gjson.Null {
		result.RateLimitSecondaryWindow = parseWindow(secondary)
	}

	codeReview := gjson.GetBytes(body, "code_review_rate_limit.primary_window")
	if codeReview.Exists() && codeReview.Type != gjson.Null {
		result.CodeReviewPrimaryWindow = parseWindow(codeReview)
	}

	return result, nil
}

func parseWindow(window gjson.Result) RateLimitWindow {
	usedPercent := helpers.ClampPercent(window.Get("used_percent").Float())
	remainingPercent := helpers.ClampPercent(100 - usedPercent)
	resetAt := unixSecondsResultToISO(window.Get("reset_at"))
	resetIn := helpers.FormatTimeUntil(resetAt)

	return RateLimitWindow{
		UsedPercent:      &usedPercent,
		RemainingPercent: &remainingPercent,
		ResetAt:          &resetAt,
		ResetIn:          &resetIn,
	}
}

func unixSecondsResultToISO(value gjson.Result) string {
	if !value.Exists() || value.Type == gjson.Null {
		return "unknown"
	}

	return helpers.UnixSecondsToISO(value.Float())
}
