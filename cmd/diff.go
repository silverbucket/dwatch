package cmd

import (
	"fmt"
	"math"
	"sort"

	"github.com/spf13/cobra"

	"dwatch/internal/report"
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
	diffCmd.Flags().StringVarP(&diffSince, "since", "s", "", "compare to snapshot from this time ago (30min, 1h, 2d, 3w, 1m, YYYY-MM-DD)")
	diffCmd.Flags().StringVar(&diffMinChange, "min-change", "1mb", "minimum change to show")
	diffCmd.Flags().IntVarP(&diffLimit, "limit", "l", 30, "max rows to show (0 = all)")
	diffCmd.Flags().BoolVarP(&diffAll, "all", "a", false, "show all dirs including unchanged")
}

func runDiff(_ *cobra.Command, _ []string) error {
	pair, err := resolveComparePair(diffSince)
	if err != nil {
		return err
	}
	latest, old := pair.latest, pair.baseline

	minChange, err := parseBytes(diffMinChange)
	if err != nil {
		return err
	}

	duration := latest.TakenAt.Sub(old.TakenAt)
	days := duration.Hours() / 24

	fmt.Printf("\n  %s\n", ui.Header("Disk Change Report"))
	fmt.Printf("  From:  %s\n", ui.Dim(old.TakenAt.Format("2006-01-02 15:04:05")))
	fmt.Printf("  To:    %s\n", ui.Dim(latest.TakenAt.Format("2006-01-02 15:04:05")))
	fmt.Printf("  Span:  %s\n", ui.Bold(formatDuration(days)))
	if pair.note != "" {
		fmt.Printf("  %s\n", ui.Dim("(baseline: "+pair.note+")"))
	}
	fmt.Println()

	var changes []report.Change
	seen := make(map[string]bool)

	for path, after := range latest.Dirs {
		seen[path] = true
		if pathSkipped(path, latest, old) {
			continue
		}
		before := old.Dirs[path]
		delta := after - before
		if !diffAll && int64(math.Abs(float64(delta))) < minChange {
			continue
		}
		changes = append(changes, report.Change{Path: path, Delta: delta})
	}

	for path, before := range old.Dirs {
		if seen[path] || pathSkipped(path, latest, old) {
			continue
		}
		delta := -before
		if !diffAll && int64(math.Abs(float64(delta))) < minChange {
			continue
		}
		changes = append(changes, report.Change{Path: path, Delta: delta})
	}

	changes = report.LeafFilter(changes)

	type row struct {
		path   string
		before int64
		after  int64
		delta  int64
		pct    float64
	}
	var rows []row
	for _, c := range changes {
		before := old.Dirs[c.Path]
		after := latest.Dirs[c.Path]
		pct := 0.0
		if before > 0 {
			pct = float64(c.Delta) / float64(before) * 100
		}
		rows = append(rows, row{c.Path, before, after, c.Delta, pct})
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
