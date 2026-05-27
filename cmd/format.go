package cmd

import "fmt"

func formatDuration(days float64) string {
	switch {
	case days < 0.1:
		mins := days * 24 * 60
		return fmt.Sprintf("%.0f minutes", mins)
	case days < 1:
		hours := days * 24
		return fmt.Sprintf("%.1f hours", hours)
	case days < 7:
		return fmt.Sprintf("%.1f days", days)
	case days < 31:
		return fmt.Sprintf("%.1f weeks", days/7)
	default:
		return fmt.Sprintf("%.1f months", days/30)
	}
}
