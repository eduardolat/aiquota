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
	"github.com/varavelio/tinta"
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
	title := tinta.Box().BorderDouble().BrightCyan().PaddingX(3).Center()
	titleText := tinta.Text().BrightCyan().Bold().String("AI QUOTA REPORT")
	fmt.Println(title.String(titleText))
	fmt.Println()

	if copilotOut != nil {
		printCopilotReport(copilotOut)
	}

	if zaiOut != nil {
		printZAIReport(zaiOut)
	}

	if codexOut != nil {
		printCodexReport(codexOut)
	}

	if len(warnings) > 0 {
		printWarnings(warnings)
	}

	fmt.Println()
}

func printCopilotReport(out *copilot.Quota) {
	key := tinta.Text().Bold()
	heading := tinta.Text().BrightBlue().Bold().String("GitHub Copilot")
	box := tinta.Box().BorderRounded().Blue().PaddingX(2).PaddingY(1).MarginBottom(1).CenterFirstLine()

	content := strings.Join([]string{
		heading,
		"",
		fmt.Sprintf("%s %s (%s)", key.String("Account:"), out.AccountUser, out.AccountType),
		"",
		fmt.Sprintf("%s %d / %d", key.String("Requests:"), out.RequestsUsed, out.RequestsTotal),
		fmt.Sprintf("%s %s", key.String("Used:"), colorPercent(out.RequestsUsedPercent)),
		fmt.Sprintf("%s %s", key.String("Reset in:"), formatReset(out.ResetIn, out.ResetAt)),
	}, "\n")

	fmt.Println(box.String(content))
}

func printZAIReport(out *zai.Quota) {
	key := tinta.Text().Bold()
	heading := tinta.Text().BrightYellow().Bold().String("Z.ai")
	box := tinta.Box().BorderRounded().Yellow().PaddingX(2).PaddingY(1).MarginBottom(1).CenterFirstLine()

	sections := []string{
		heading,
		"",
		fmt.Sprintf("%s %s (%s)", key.String("Account:"), out.AccountID, out.AccountType),
		"",
		key.String("Token Quota"),
		fmt.Sprintf("%s %s", key.String("Used:"), colorPercent(out.TokenQuota.UsedPercent)),
		fmt.Sprintf("%s %s", key.String("Reset in:"), formatReset(out.TokenQuota.ResetIn, out.TokenQuota.ResetAt)),
		"",
		key.String("MCP Quota"),
		fmt.Sprintf("%s %s", key.String("Used:"), colorPercent(out.MCPQuota.UsedPercent)),
		fmt.Sprintf("%s %s", key.String("Reset in:"), formatReset(out.MCPQuota.ResetIn, out.MCPQuota.ResetAt)),
	}

	if len(out.MCPQuota.Details) > 0 {
		sections = append(sections, "", key.String("MCP Details"))
		for _, detail := range out.MCPQuota.Details {
			sections = append(sections, fmt.Sprintf("- %s: %s", detail.ModelCode, formatNumber(detail.Usage)))
		}
	}

	fmt.Println(box.String(strings.Join(sections, "\n")))
}

func printCodexReport(out *codex.Quota) {
	key := tinta.Text().Bold()
	section := tinta.Text().Bold()
	heading := tinta.Text().BrightMagenta().Bold().String("OpenAI Codex")
	box := tinta.Box().BorderRounded().Magenta().PaddingX(2).PaddingY(1).MarginBottom(1).CenterFirstLine()

	sections := []string{
		heading,
		"",
		fmt.Sprintf("%s %s (%s)", key.String("Account:"), out.AccountEmail, out.AccountType),
		"",
		formatRateLimitWindow("Rate Limit Primary Window", out.RateLimitPrimaryWindow, key, section),
		"",
		formatRateLimitWindow("Rate Limit Secondary Window", out.RateLimitSecondaryWindow, key, section),
		"",
		formatRateLimitWindow("Code Review Primary Window", out.CodeReviewPrimaryWindow, key, section),
	}

	fmt.Println(box.String(strings.Join(sections, "\n")))
}

func printWarnings(warnings []string) {
	title := tinta.Text().BrightRed().Bold().String("Warnings")
	body := []string{title, tinta.Text().Red().String("Some providers could not be queried:")}
	for _, warning := range warnings {
		body = append(body, tinta.Text().Yellow().Sprintf("- %s", warning))
	}

	box := tinta.Box().BorderRounded().Red().PaddingX(2).PaddingY(1)
	fmt.Println(box.String(strings.Join(body, "\n")))
}

func formatRateLimitWindow(name string, window codex.RateLimitWindow, key *tinta.TextStyle, section *tinta.TextStyle) string {
	lines := []string{section.String(name)}

	if window.UsedPercent == nil || window.ResetAt == nil || window.ResetIn == nil {
		lines = append(lines,
			fmt.Sprintf("%s %s", key.String("Usage:"), "unavailable"),
			fmt.Sprintf("%s %s", key.String("Reset in:"), "unavailable"),
		)

		return strings.Join(lines, "\n")
	}

	lines = append(lines,
		fmt.Sprintf("%s %s", key.String("Used:"), colorPercent(*window.UsedPercent)),
		fmt.Sprintf("%s %s", key.String("Reset in:"), formatReset(*window.ResetIn, *window.ResetAt)),
	)

	return strings.Join(lines, "\n")
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

func formatReset(resetIn string, resetAt string) string {
	return fmt.Sprintf("%s - %s", resetIn, formatResetAt(resetAt))
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

func colorPercent(value float64) string {
	percent := formatPercent(value) + "%"

	switch {
	case value >= 75:
		return tinta.Text().BrightRed().Bold().String(percent)
	case value >= 50:
		return tinta.Text().BrightYellow().Bold().String(percent)
	default:
		return tinta.Text().BrightGreen().Bold().String(percent)
	}
}
