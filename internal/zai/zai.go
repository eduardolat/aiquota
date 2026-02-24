package zai

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/eduardolat/aiquota/internal/credentials"
	"github.com/eduardolat/aiquota/internal/helpers"
	"github.com/tidwall/gjson"
)

// QuotaWindow represents a usage window.
type QuotaWindow struct {
	UsedPercent      float64 `json:"usedPercent"`
	RemainingPercent float64 `json:"remainingPercent"`
	ResetAt          string  `json:"resetAt"`
	ResetIn          string  `json:"resetIn"`
}

// MCPQuota represents MCP quota information.
type MCPQuota struct {
	QuotaWindow
	Details []MCPDetail `json:"details"`
}

// MCPDetail contains usage per model code.
type MCPDetail struct {
	ModelCode string  `json:"modelCode"`
	Usage     float64 `json:"usage"`
}

// Quota contains Z.ai quota information.
type Quota struct {
	AccountID   string      `json:"accountId"`
	AccountType string      `json:"accountType"`
	TokenQuota  QuotaWindow `json:"tokenQuota"`
	MCPQuota    MCPQuota    `json:"mcpQuota"`
}

// GetQuota fetches Z.ai quota information.
func GetQuota(ctx context.Context, creds credentials.Credentials) (Quota, error) {
	if creds.ZAIAPIKey == nil || *creds.ZAIAPIKey == "" {
		return Quota{}, fmt.Errorf("missing Z.ai API key in credentials")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.z.ai/api/monitor/usage/quota/limit", nil)
	if err != nil {
		return Quota{}, fmt.Errorf("failed to create Z.ai request: %w", err)
	}

	req.Header.Set("Authorization", *creds.ZAIAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "OpenCode-Quota-Plugin/1.0")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return Quota{}, fmt.Errorf("failed to fetch Z.ai quota: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return Quota{}, fmt.Errorf("failed to read Z.ai response: %w", err)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Quota{}, fmt.Errorf("failed to fetch Z.ai quota. Status: %d, Response: %s", response.StatusCode, string(body))
	}

	if !gjson.GetBytes(body, "success").Bool() || gjson.GetBytes(body, "code").Int() != 200 {
		return Quota{}, fmt.Errorf(
			"failed to fetch Z.ai quota. Code: %d, Message: %s",
			gjson.GetBytes(body, "code").Int(),
			gjson.GetBytes(body, "msg").String(),
		)
	}

	limits := gjson.GetBytes(body, "data.limits").Array()
	tokenLimit := findLimitByType(limits, "TOKENS_LIMIT")
	timeLimit := findLimitByType(limits, "TIME_LIMIT")

	tokenResetAt := unixMillisResultToISO(tokenLimit.Get("nextResetTime"))
	tokenUsedPercent := helpers.ClampPercent(tokenLimit.Get("percentage").Float())

	mcpResetAt := unixMillisResultToISO(timeLimit.Get("nextResetTime"))
	mcpUsedPercent := helpers.ClampPercent(timeLimit.Get("percentage").Float())

	return Quota{
		AccountID:   maskToken(*creds.ZAIAPIKey),
		AccountType: gjson.GetBytes(body, "data.level").String(),
		TokenQuota: QuotaWindow{
			UsedPercent:      tokenUsedPercent,
			RemainingPercent: helpers.ClampPercent(100 - tokenUsedPercent),
			ResetAt:          tokenResetAt,
			ResetIn:          helpers.FormatTimeUntil(tokenResetAt),
		},
		MCPQuota: MCPQuota{
			QuotaWindow: QuotaWindow{
				UsedPercent:      mcpUsedPercent,
				RemainingPercent: helpers.ClampPercent(100 - mcpUsedPercent),
				ResetAt:          mcpResetAt,
				ResetIn:          helpers.FormatTimeUntil(mcpResetAt),
			},
			Details: parseUsageDetails(timeLimit.Get("usageDetails")),
		},
	}, nil
}

func findLimitByType(limits []gjson.Result, limitType string) gjson.Result {
	for _, limit := range limits {
		if limit.Get("type").String() == limitType {
			return limit
		}
	}

	return gjson.Result{}
}

func parseUsageDetails(details gjson.Result) []MCPDetail {
	items := details.Array()
	result := make([]MCPDetail, 0, len(items))
	for _, item := range items {
		result = append(result, MCPDetail{
			ModelCode: item.Get("modelCode").String(),
			Usage:     item.Get("usage").Float(),
		})
	}

	return result
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "********"
	}

	start := token[:6]
	end := token[len(token)-4:]
	return start + "..." + end
}

func unixMillisResultToISO(value gjson.Result) string {
	if !value.Exists() || value.Type == gjson.Null {
		return "unknown"
	}

	return helpers.UnixMillisToISO(value.Float())
}
