package cmd

import (
	"fmt"
	"math"
	"sort"

	"github.com/spf13/cobra"

	"dwatch/internal/report"
	"dwatch/internal/ui"
)

var alertCmd = &cobra.Command{
	Use:   "alert",
	Short: "Exit non-zero if any directory grew past a threshold",
	Long: `Designed for cron jobs. Prints offending directories and exits 1 if
any directory grew more than --threshold since the comparison snapshot.
Combined with cron MAILTO, this sends you an email when disk is growing fast.

Example crontab:
  0 * * * *   /usr/local/bin/dwatch scan --quiet
  0 8 * * *   /usr/local/bin/dwatch alert --since 1d --threshold 500mb`,
	RunE: runAlert,
}

var (
	alertSince     string
	alertThreshold string
)

func init() {
	rootCmd.AddCommand(alertCmd)
	alertCmd.Flags().StringVarP(&alertSince, "since", "s", "", "comparison window (30min, 1h, 2d, 3w, 1m, YYYY-MM-DD); default: previous scan")
	alertCmd.Flags().StringVarP(&alertThreshold, "threshold", "t", "500mb", "growth threshold to alert on (e.g. 100mb, 1gb)")
}

func runAlert(_ *cobra.Command, _ []string) error {
	threshold, err := parseBytes(alertThreshold)
	if err != nil {
		return err
	}

	pair, err := resolveComparePair(alertSince)
	if err != nil {
		return err
	}
	latest, old := pair.latest, pair.baseline

	type hit struct {
		path  string
		delta int64
		pct   float64
	}

	var changes []report.Change
	for path, after := range latest.Dirs {
		if pathSkipped(path, latest, old) {
			continue
		}
		before := old.Dirs[path]
		delta := after - before
		if delta >= threshold {
			changes = append(changes, report.Change{Path: path, Delta: delta})
		}
	}
	changes = report.LeafFilter(changes)

	var hits []hit
	for _, c := range changes {
		before := old.Dirs[c.Path]
		pct := 0.0
		if before > 0 {
			pct = float64(c.Delta) / float64(before) * 100
		}
		hits = append(hits, hit{c.Path, c.Delta, pct})
	}

	sort.Slice(hits, func(i, j int) bool {
		return math.Abs(float64(hits[i].delta)) > math.Abs(float64(hits[j].delta))
	})

	if len(hits) == 0 {
		return nil
	}

	duration := latest.TakenAt.Sub(old.TakenAt)
	days := duration.Hours() / 24

	fmt.Printf("\ndwatch ALERT: %d director(ies) grew >%s in %s\n\n",
		len(hits), ui.FormatSize(threshold), formatDuration(days))

	tableRows := make([][]string, len(hits))
	for i, h := range hits {
		tableRows[i] = []string{
			h.path,
			ui.FormatChange(h.delta),
			ui.FormatPct(h.pct),
		}
	}

	ui.PrintTable(
		[]ui.Column{
			{Header: "Directory"},
			{Header: "Growth", RightAlign: true},
			{Header: "%", RightAlign: true},
		},
		tableRows,
	)
	fmt.Println()

	return exitCode(1)
}
