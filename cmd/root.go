package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var dataDir string

var rootCmd = &cobra.Command{
	Use:   "dwatch",
	Short: "Track disk space growth over time",
	Long: `dwatch snapshots directory sizes and shows you what's growing.

Quick start:
  dwatch scan                  # take a snapshot (run this from cron too)
  dwatch status                # quick summary of latest snapshot + recent changes
  dwatch diff --since 1w       # what grew in the last week
  dwatch top --since 1m        # top growing directories this month
  dwatch alert --threshold 1gb # exit 1 if anything grew >1 GB since last scan`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	home, _ := os.UserHomeDir()
	rootCmd.PersistentFlags().StringVarP(&dataDir, "data-dir", "d", home+"/.dwatch", "directory to store snapshots")
}

// parseSince converts a human duration string to an absolute time.
// Accepts: 1h, 2d, 3w, 1m, or YYYY-MM-DD.
func parseSince(s string) (time.Time, error) {
	if len(s) < 2 {
		return time.Time{}, fmt.Errorf("invalid format %q — use 1h/2d/3w/1m or YYYY-MM-DD", s)
	}

	n, err := strconv.Atoi(s[:len(s)-1])
	if err == nil {
		switch s[len(s)-1] {
		case 'h':
			return time.Now().Add(-time.Duration(n) * time.Hour), nil
		case 'd':
			return time.Now().AddDate(0, 0, -n), nil
		case 'w':
			return time.Now().AddDate(0, 0, -n*7), nil
		case 'm':
			return time.Now().AddDate(0, -n, 0), nil
		}
	}

	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid format %q — use 1h/2d/3w/1m or YYYY-MM-DD", s)
	}
	return t, nil
}

func parseBytes(s string) (int64, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid size %q — use 100mb, 1gb, 500kb", s)
	}

	lower := s
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			lower = s[:i] + string(s[i]+32) + s[i+1:]
		}
	}

	suffixes := map[string]int64{
		"kb": 1 << 10,
		"mb": 1 << 20,
		"gb": 1 << 30,
		"tb": 1 << 40,
		"k":  1 << 10,
		"m":  1 << 20,
		"g":  1 << 30,
		"t":  1 << 40,
	}

	for suffix, mult := range suffixes {
		if strings.HasSuffix(lower, suffix) {
			numStr := lower[:len(lower)-len(suffix)]
			n, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid size %q", s)
			}
			return int64(n * float64(mult)), nil
		}
	}

	n, err := strconv.ParseInt(lower, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q — use 100mb, 1gb, 500kb", s)
	}
	return n, nil
}
