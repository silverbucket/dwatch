package cmd

import (
	"fmt"
	"math"
	"sort"

	"github.com/spf13/cobra"

	"dwatch/internal/store"
	"dwatch/internal/ui"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show what changed between two snapshots",
	Example: `  dwatch diff                  # compare latest vs previous snapshot
  dwatch diff --since 1d       # compare latest vs ~1 day ago
  dwatch diff --since 1w       # compare latest vs ~1 week ago
  dwatch diff --since 2026-05-01`,
	RunE: runDiff,
}

var (
	diffSince     string
	diffMinChange string
	diffLimit     int
	diffAll       bool
)

func init() {
	rootCmd.AddCommand(diffCmd)
	diffCmd.Flags().StringVarP(&diffSince, "since", "s", "", "compare to snapshot from this time ago (1h, 2d, 3w, 1m, YYYY-MM-DD)")
	diffCmd.Flags().StringVar(&diffMinChange, "min-change", "1mb", "minimum change to show")
	diffCmd.Flags().IntVarP(&diffLimit, "limit", "l", 30, "max rows to show (0 = all)")
	diffCmd.Flags().BoolVarP(&diffAll, "all", "a", false, "show all dirs including unchanged")
}

func runDiff(_ *cobra.Command, _ []string) error {
	latest, err := store.Latest(dataDir)
	if err != nil || latest == nil {
		return fmt.Errorf("no snapshots found — run 'dwatch scan' first")
	}

	var old *store.Snapshot
	if diffSince != "" {
		cutoff, err := parseSince(diffSince)
		if err != nil {
			return err
		}
		old, err = store.LatestBefore(dataDir, cutoff)
		if err != nil {
			return err
		}
	} else {
		old, err = store.Previous(dataDir)
		if err != nil {
			return err
		}
	}

	if old == nil {
		return fmt.Errorf("no earlier snapshot found — take more scans first")
	}

	minChange, err := parseBytes(diffMinChange)
	if err != nil {
		return err
	}

	duration := latest.TakenAt.Sub(old.TakenAt)
	days := duration.Hours() / 24

	fmt.Printf("\n  %s\n", ui.Header("Disk Change Report"))
	fmt.Printf("  From:  %s\n", ui.Dim(old.TakenAt.Format("2006-01-02 15:04:05")))
	fmt.Printf("  To:    %s\n", ui.Dim(latest.TakenAt.Format("2006-01-02 15:04:05")))
	fmt.Printf("  Span:  %s\n\n", ui.Bold(formatDuration(days)))

	type row struct {
		path   string
		before int64
		after  int64
		delta  int64
		pct    float64
	}

	var rows []row
	seen := make(map[string]bool)

	for path, after := range latest.Dirs {
		seen[path] = true
		if isSkipped(path) {
			continue
		}
		before := old.Dirs[path]
		delta := after - before
		if !diffAll && int64(math.Abs(float64(delta))) < minChange {
			continue
		}
		pct := 0.0
		if before > 0 {
			pct = float64(delta) / float64(before) * 100
		}
		rows = append(rows, row{path, before, after, delta, pct})
	}

	// dirs in old but not latest (deleted or now skipped)
	for path, before := range old.Dirs {
		if seen[path] || isSkipped(path) {
			continue
		}
		delta := -before
		if !diffAll && int64(math.Abs(float64(delta))) < minChange {
			continue
		}
		rows = append(rows, row{path, before, 0, delta, -100})
	}

	sort.Slice(rows, func(i, j int) bool {
		return math.Abs(float64(rows[i].delta)) > math.Abs(float64(rows[j].delta))
	})

	if diffLimit > 0 && len(rows) > diffLimit {
		rows = rows[:diffLimit]
	}

	if len(rows) == 0 {
		fmt.Println("  No significant changes found.")
		fmt.Println()
		return nil
	}

	tableRows := make([][]string, len(rows))
	for i, r := range rows {
		tableRows[i] = []string{
			r.path,
			ui.FormatSize(r.before),
			ui.FormatSize(r.after),
			ui.FormatChange(r.delta),
			ui.FormatPct(r.pct),
		}
	}

	ui.PrintTable(
		[]ui.Column{
			{Header: "Directory"},
			{Header: "Before", RightAlign: true},
			{Header: "After", RightAlign: true},
			{Header: "Change", RightAlign: true},
			{Header: "%", RightAlign: true},
		},
		tableRows,
	)
	fmt.Println()
	return nil
}

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
