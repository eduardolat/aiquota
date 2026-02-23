package helpers

import (
	"fmt"
	"math"
	"time"
)

// FormatTimeUntil returns a compact human-readable duration until the given ISO datetime.
func FormatTimeUntil(dateISO string) string {
	target, err := time.Parse(time.RFC3339, dateISO)
	if err != nil {
		return "unknown"
	}

	diff := time.Until(target)
	if diff <= 0 {
		return "now"
	}

	totalMinutes := int(math.Floor(diff.Minutes()))
	days := totalMinutes / 1440
	hours := (totalMinutes % 1440) / 60
	minutes := totalMinutes % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}

	return fmt.Sprintf("%dm", minutes)
}

// ClampPercent rounds a percent value to two decimals within [0, 100].
func ClampPercent(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}

	rounded := math.Round(value*100) / 100
	return min(100, max(0, rounded))
}

// UnixSecondsToISO converts unix timestamp seconds to RFC3339 or "unknown".
func UnixSecondsToISO(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "unknown"
	}

	date := time.Unix(int64(value), 0).UTC()
	return date.Format(time.RFC3339)
}

// UnixMillisToISO converts unix timestamp milliseconds to RFC3339 or "unknown".
func UnixMillisToISO(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "unknown"
	}

	date := time.UnixMilli(int64(value)).UTC()
	return date.Format(time.RFC3339)
}

// ToReadableFuture prepends "in " for future duration strings.
func ToReadableFuture(resetAtISO string) string {
	base := FormatTimeUntil(resetAtISO)
	if base == "unknown" || base == "now" {
		return base
	}

	return "in " + base
}
