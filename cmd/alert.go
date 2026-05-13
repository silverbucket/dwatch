package cmd

import (
	"fmt"
	"math"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"dwatch/internal/store"
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
	alertCmd.Flags().StringVarP(&alertSince, "since", "s", "", "comparison window (1h, 2d, 3w, 1m, YYYY-MM-DD); default: previous scan")
	alertCmd.Flags().StringVarP(&alertThreshold, "threshold", "t", "500mb", "growth threshold to alert on (e.g. 100mb, 1gb)")
	_ = alertCmd.MarkFlagRequired("threshold")
}

func runAlert(_ *cobra.Command, _ []string) error {
	threshold, err := parseBytes(alertThreshold)
	if err != nil {
		return err
	}

	latest, err := store.Latest(dataDir)
	if err != nil || latest == nil {
		return fmt.Errorf("no snapshots found — run 'dwatch scan' first")
	}

	var old *store.Snapshot
	if alertSince != "" {
		cutoff, err := parseSince(alertSince)
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

	type hit struct {
		path  string
		delta int64
		pct   float64
	}

	var hits []hit
	for path, after := range latest.Dirs {
		before := old.Dirs[path]
		delta := after - before
		if delta >= threshold {
			pct := 0.0
			if before > 0 {
				pct = float64(delta) / float64(before) * 100
			}
			hits = append(hits, hit{path, delta, pct})
		}
	}

	sort.Slice(hits, func(i, j int) bool {
		return math.Abs(float64(hits[i].delta)) > math.Abs(float64(hits[j].delta))
	})

	if len(hits) == 0 {
		return nil // exit 0, no alert
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

	os.Exit(1)
	return nil
}
