package cmd

import (
	"fmt"
	"math"
	"sort"

	"github.com/spf13/cobra"

	"dwatch/internal/store"
	"dwatch/internal/ui"
)

var topCmd = &cobra.Command{
	Use:   "top",
	Short: "Rank directories by growth",
	Example: `  dwatch top                        # top movers since previous scan
  dwatch top --since 1w             # top movers over the last week
  dwatch top --since 1d --by pct   # fastest-growing by percentage (catches small dirs exploding)
  dwatch top --since 1m --limit 10`,
	RunE: runTop,
}

var (
	topSince string
	topLimit int
	topBy    string
)

func init() {
	rootCmd.AddCommand(topCmd)
	topCmd.Flags().StringVarP(&topSince, "since", "s", "", "time window (1h, 2d, 3w, 1m, YYYY-MM-DD)")
	topCmd.Flags().IntVarP(&topLimit, "limit", "l", 20, "number of results to show (0 = all)")
	topCmd.Flags().StringVar(&topBy, "by", "growth", "sort by: growth (absolute bytes), pct (percentage), rate (bytes/day)")
}

func runTop(_ *cobra.Command, _ []string) error {
	latest, err := store.Latest(dataDir)
	if err != nil || latest == nil {
		return fmt.Errorf("no snapshots found — run 'dwatch scan' first")
	}

	var old *store.Snapshot
	if topSince != "" {
		cutoff, err := parseSince(topSince)
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

	duration := latest.TakenAt.Sub(old.TakenAt)
	days := duration.Hours() / 24

	type entry struct {
		path   string
		before int64
		growth int64
		after  int64
		pct    float64
	}

	var entries []entry
	for path, after := range latest.Dirs {
		if isSkipped(path) {
			continue
		}
		before := old.Dirs[path]
		growth := after - before
		if growth <= 0 {
			continue
		}
		pct := 0.0
		if before > 0 {
			pct = float64(growth) / float64(before) * 100
		}
		entries = append(entries, entry{path, before, growth, after, pct})
	}

	switch topBy {
	case "pct":
		sort.Slice(entries, func(i, j int) bool { return entries[i].pct > entries[j].pct })
	case "rate":
		sort.Slice(entries, func(i, j int) bool {
			ri := float64(entries[i].growth) / math.Max(days, 0.001)
			rj := float64(entries[j].growth) / math.Max(days, 0.001)
			return ri > rj
		})
	default: // "growth"
		sort.Slice(entries, func(i, j int) bool { return entries[i].growth > entries[j].growth })
	}

	if topLimit > 0 && len(entries) > topLimit {
		entries = entries[:topLimit]
	}

	sortLabel := map[string]string{
		"growth": "absolute growth",
		"pct":    "% growth",
		"rate":   "growth rate/day",
	}[topBy]

	fmt.Printf("\n  %s\n", ui.Header("Top Growing Directories"))
	fmt.Printf("  From:  %s\n", ui.Dim(old.TakenAt.Format("2006-01-02 15:04:05")))
	fmt.Printf("  To:    %s\n", ui.Dim(latest.TakenAt.Format("2006-01-02 15:04:05")))
	fmt.Printf("  Span:  %s   Sorted by: %s\n\n", ui.Bold(formatDuration(days)), ui.Dim(sortLabel))

	if len(entries) == 0 {
		fmt.Println("  No growth detected in this window.")
		fmt.Println()
		return nil
	}

	tableRows := make([][]string, len(entries))
	for i, e := range entries {
		ratePerDay := 0.0
		if days > 0 {
			ratePerDay = float64(e.growth) / days
		}
		tableRows[i] = []string{
			fmt.Sprintf("%d", i+1),
			e.path,
			ui.FormatChange(e.growth),
			ui.FormatPct(e.pct),
			ui.FormatSize(e.after),
			ui.FormatRate(ratePerDay),
		}
	}

	ui.PrintTable(
		[]ui.Column{
			{Header: "#", RightAlign: true},
			{Header: "Directory"},
			{Header: "Growth", RightAlign: true},
			{Header: "%", RightAlign: true},
			{Header: "Current", RightAlign: true},
			{Header: "Rate/day", RightAlign: true},
		},
		tableRows,
	)
	fmt.Println()
	return nil
}
