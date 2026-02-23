package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eduardolat/aiquota/internal/codex"
	"github.com/eduardolat/aiquota/internal/copilot"
	"github.com/eduardolat/aiquota/internal/credentials"
	"github.com/eduardolat/aiquota/internal/zai"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	creds, err := credentials.GetCredentials()
	if err != nil {
		return err
	}

	hasCopilot := hasCredential(creds.CopilotAPIKey)
	hasZAI := hasCredential(creds.ZAIAPIKey)
	hasCodex := hasCredential(creds.CodexAPIKey)
	if !hasCopilot && !hasZAI && !hasCodex {
		return fmt.Errorf("no provider credentials found in auth.json")
	}

	ctx := context.Background()

	var (
		wg         sync.WaitGroup
		mu         sync.Mutex
		warnings   []string
		copilotOut *copilot.Quota
		zaiOut     *zai.Quota
		codexOut   *codex.Quota
	)

	if hasCopilot {
		wg.Go(func() {
			quota, err := copilot.GetQuota(ctx, creds)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				warnings = append(warnings, "GitHub Copilot: "+err.Error())
				return
			}
			copilotOut = &quota
		})
	}

	if hasZAI {
		wg.Go(func() {
			quota, err := zai.GetQuota(ctx, creds)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				warnings = append(warnings, "Z.ai: "+err.Error())
				return
			}
			zaiOut = &quota
		})
	}

	if hasCodex {
		wg.Go(func() {
			quota, err := codex.GetQuota(ctx, creds)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				warnings = append(warnings, "OpenAI Codex: "+err.Error())
				return
			}
			codexOut = &quota
		})
	}

	wg.Wait()

	if copilotOut == nil && zaiOut == nil && codexOut == nil {
		return fmt.Errorf("could not fetch quota data from any provider")
	}

	printReport(copilotOut, zaiOut, codexOut, warnings)

	return nil
}

func hasCredential(value *string) bool {
	return value != nil && strings.TrimSpace(*value) != ""
}

func printReport(copilotOut *copilot.Quota, zaiOut *zai.Quota, codexOut *codex.Quota, warnings []string) {
	fmt.Println("AI QUOTA REPORT")
	fmt.Println()

	if copilotOut != nil {
		fmt.Println("Provider: GitHub Copilot")
		fmt.Printf("Account: %s (%s)\n", copilotOut.AccountUser, copilotOut.AccountType)
		fmt.Println("- Requests")
		fmt.Printf(
			"  Usage: %d / %d | Used %s%% | Remaining %s%%\n",
			copilotOut.RequestsUsed,
			copilotOut.RequestsTotal,
			formatPercent(copilotOut.RequestsUsedPercent),
			formatPercent(copilotOut.RequestsRemainingPercent),
		)
		fmt.Printf("  Reset: %s | %s\n", copilotOut.ResetIn, formatResetAt(copilotOut.ResetAt))
		fmt.Println()
	}

	if zaiOut != nil {
		fmt.Println("Provider: Z.ai")
		fmt.Printf("Account: %s (%s)\n", zaiOut.AccountID, zaiOut.AccountType)
		fmt.Println("- Token Quota")
		fmt.Printf(
			"  Usage: Used %s%% | Remaining %s%%\n",
			formatPercent(zaiOut.TokenQuota.UsedPercent),
			formatPercent(zaiOut.TokenQuota.RemainingPercent),
		)
		fmt.Printf("  Reset: %s | %s\n", zaiOut.TokenQuota.ResetIn, formatResetAt(zaiOut.TokenQuota.ResetAt))
		fmt.Println("- MCP Quota")
		fmt.Printf(
			"  Usage: Used %s%% | Remaining %s%%\n",
			formatPercent(zaiOut.MCPQuota.UsedPercent),
			formatPercent(zaiOut.MCPQuota.RemainingPercent),
		)
		fmt.Printf("  Reset: %s | %s\n", zaiOut.MCPQuota.ResetIn, formatResetAt(zaiOut.MCPQuota.ResetAt))
		if len(zaiOut.MCPQuota.Details) > 0 {
			fmt.Println("  Details:")
			for _, detail := range zaiOut.MCPQuota.Details {
				fmt.Printf("  - %s: %s\n", detail.ModelCode, formatNumber(detail.Usage))
			}
		}
		fmt.Println()
	}

	if codexOut != nil {
		fmt.Println("Provider: OpenAI Codex")
		fmt.Printf("Account: %s (%s)\n", codexOut.AccountEmail, codexOut.AccountType)
		printRateLimitWindow("Rate Limit Primary Window", codexOut.RateLimitPrimaryWindow)
		printRateLimitWindow("Rate Limit Secondary Window", codexOut.RateLimitSecondaryWindow)
		printRateLimitWindow("Code Review Primary Window", codexOut.CodeReviewPrimaryWindow)
		fmt.Println()
	}

	if len(warnings) > 0 {
		fmt.Println("Warnings:")
		for _, warning := range warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printRateLimitWindow(name string, window codex.RateLimitWindow) {
	fmt.Printf("- %s\n", name)
	if window.UsedPercent == nil || window.RemainingPercent == nil || window.ResetAt == nil || window.ResetIn == nil {
		fmt.Println("  Usage: unavailable")
		fmt.Println("  Reset: unavailable")
		return
	}

	fmt.Printf(
		"  Usage: Used %s%% | Remaining %s%%\n",
		formatPercent(*window.UsedPercent),
		formatPercent(*window.RemainingPercent),
	)
	fmt.Printf("  Reset: %s | %s\n", *window.ResetIn, formatResetAt(*window.ResetAt))
}

func formatResetAt(value string) string {
	if value == "" || value == "unknown" {
		return "unknown"
	}

	timeValue, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}

	return timeValue.UTC().Format("2006-01-02 15:04:05")
}

func formatPercent(value float64) string {
	formatted := strconv.FormatFloat(value, 'f', 2, 64)
	formatted = strings.TrimRight(formatted, "0")
	formatted = strings.TrimRight(formatted, ".")
	if formatted == "" {
		return "0"
	}

	return formatted
}

func formatNumber(value float64) string {
	if value == float64(int64(value)) {
		return strconv.FormatInt(int64(value), 10)
	}

	return formatPercent(value)
}
