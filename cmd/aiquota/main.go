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
	fmt.Println(bold("+-----------------+"))
	fmt.Println(bold("| AI QUOTA REPORT |"))
	fmt.Println(bold("+-----------------+"))
	fmt.Println()

	if copilotOut != nil {
		printProviderHeader("GitHub Copilot", copilotOut.AccountUser, copilotOut.AccountType)
		fmt.Println(bold("Requests"))
		fmt.Printf(
			"%s %d / %d | Used %s%% | Remaining %s%%\n",
			bold("Usage:"),
			copilotOut.RequestsUsed,
			copilotOut.RequestsTotal,
			formatPercent(copilotOut.RequestsUsedPercent),
			formatPercent(copilotOut.RequestsRemainingPercent),
		)
		fmt.Printf("%s %s | %s\n", bold("Reset:"), copilotOut.ResetIn, formatResetAt(copilotOut.ResetAt))
	}

	if zaiOut != nil {
		printSectionDivider()
		printProviderHeader("Z.ai", zaiOut.AccountID, zaiOut.AccountType)
		fmt.Println(bold("Token Quota"))
		fmt.Printf(
			"%s Used %s%% | Remaining %s%%\n",
			bold("Usage:"),
			formatPercent(zaiOut.TokenQuota.UsedPercent),
			formatPercent(zaiOut.TokenQuota.RemainingPercent),
		)
		fmt.Printf("%s %s | %s\n", bold("Reset:"), zaiOut.TokenQuota.ResetIn, formatResetAt(zaiOut.TokenQuota.ResetAt))
		fmt.Println()
		fmt.Println(bold("MCP Quota"))
		fmt.Printf(
			"%s Used %s%% | Remaining %s%%\n",
			bold("Usage:"),
			formatPercent(zaiOut.MCPQuota.UsedPercent),
			formatPercent(zaiOut.MCPQuota.RemainingPercent),
		)
		fmt.Printf("%s %s | %s\n", bold("Reset:"), zaiOut.MCPQuota.ResetIn, formatResetAt(zaiOut.MCPQuota.ResetAt))
		if len(zaiOut.MCPQuota.Details) > 0 {
			fmt.Println(bold("Details:"))
			for _, detail := range zaiOut.MCPQuota.Details {
				fmt.Printf("- %s: %s\n", detail.ModelCode, formatNumber(detail.Usage))
			}
		}
	}

	if codexOut != nil {
		printSectionDivider()
		printProviderHeader("OpenAI Codex", codexOut.AccountEmail, codexOut.AccountType)
		printRateLimitWindow("Rate Limit Primary Window", codexOut.RateLimitPrimaryWindow)
		fmt.Println()
		printRateLimitWindow("Rate Limit Secondary Window", codexOut.RateLimitSecondaryWindow)
		fmt.Println()
		printRateLimitWindow("Code Review Primary Window", codexOut.CodeReviewPrimaryWindow)
	}

	if len(warnings) > 0 {
		fmt.Println(bold("Warnings"))
		fmt.Println("Some providers could not be queried:")
		for _, warning := range warnings {
			fmt.Printf("- %s\n", warning)
		}
	}

	fmt.Println()
}

func printRateLimitWindow(name string, window codex.RateLimitWindow) {
	fmt.Println(bold(name))
	if window.UsedPercent == nil || window.RemainingPercent == nil || window.ResetAt == nil || window.ResetIn == nil {
		fmt.Printf("%s unavailable\n", bold("Usage:"))
		fmt.Printf("%s unavailable\n", bold("Reset:"))
		return
	}

	fmt.Printf(
		"%s Used %s%% | Remaining %s%%\n",
		bold("Usage:"),
		formatPercent(*window.UsedPercent),
		formatPercent(*window.RemainingPercent),
	)
	fmt.Printf("%s %s | %s\n", bold("Reset:"), *window.ResetIn, formatResetAt(*window.ResetAt))
}

func printProviderHeader(providerName string, account string, accountType string) {
	fmt.Printf("%s %s\n", bold("Provider:"), bold(providerName))
	fmt.Printf("%s %s (%s)\n", bold("Account:"), account, accountType)
	fmt.Println()
}

func printSectionDivider() {
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println()
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

func bold(value string) string {
	return "\033[1m" + value + "\033[0m"
}
