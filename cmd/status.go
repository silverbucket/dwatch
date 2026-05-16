package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"dwatch/internal/store"
	"dwatch/internal/ui"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Quick overview: latest snapshot + recent changes",
	RunE:  runStatus,
}

var statusSince string

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringVar(&statusSince, "since", "", "compare growth over this window (e.g. 1h, 2d, 1w, 1m)")
}

func runStatus(_ *cobra.Command, _ []string) error {
	snaps, err := store.List(dataDir)
	if err != nil {
		return fmt.Errorf("could not read snapshots: %w", err)
	}
	if len(snaps) == 0 {
		return fmt.Errorf("no snapshots found — run 'dwatch scan' first")
	}

	latest := snaps[len(snaps)-1]
	age := time.Since(latest.TakenAt)

	fmt.Printf("\n  %s\n", ui.Header("dwatch status"))
	fmt.Printf("  Snapshot:  %s  (%s ago)\n",
		latest.TakenAt.Format("2006-01-02 15:04:05"),
		formatDuration(age.Hours()/24))
	fmt.Printf("  Root:      %s   Depth: %d\n", latest.Root, latest.Depth)
	fmt.Printf("  Tracked:   %s directories\n", ui.Bold(ui.Num(len(latest.Dirs))))
	fmt.Printf("  Snapshots: %s total\n\n", ui.Num(len(snaps)))

	// Top 8 largest current dirs
	type entry struct {
		path string
		size int64
	}
	largest := make([]entry, 0, len(latest.Dirs))
	for p, s := range latest.Dirs {
		if isSkipped(p) {
			continue
		}
		largest = append(largest, entry{p, s})
	}
	sort.Slice(largest, func(i, j int) bool { return largest[i].size > largest[j].size })
	if len(largest) > 8 {
		largest = largest[:8]
	}

	fmt.Println(ui.Header("  Current largest:"))
	ui.PrintTable(
		[]ui.Column{{Header: "Directory"}, {Header: "Size", RightAlign: true}},
		func() [][]string {
			rows := make([][]string, len(largest))
			for i, e := range largest {
				rows[i] = []string{e.path, ui.FormatSize(e.size)}
			}
			return rows
		}(),
	)
	fmt.Println()

	// Find baseline snapshot for growth comparison.
	if len(snaps) < 2 {
		fmt.Println(ui.Dim("  (Take more snapshots to see growth trends)"))
		fmt.Println()
		return nil
	}

	candidates := snaps[:len(snaps)-1] // all except latest
	var prev *store.Snapshot
	var sectionLabel string

	if statusSince != "" {
		cutoff, err := parseSince(statusSince)
		if err != nil {
			return err
		}
		// Latest candidate at or before the cutoff.
		for _, s := range candidates {
			if !s.TakenAt.After(cutoff) {
				prev = s
			}
		}
		if prev == nil {
			// No snapshot old enough; use oldest and note it.
			prev = candidates[0]
			sectionLabel = fmt.Sprintf("  Growth since %s %s:", statusSince,
				ui.Dim("(oldest snapshot: "+formatDuration(time.Since(prev.TakenAt).Hours()/24)+")"))
		} else {
			sectionLabel = fmt.Sprintf("  Growth since %s:", statusSince)
		}
	} else {
		prev = candidates[len(candidates)-1]
		span := latest.TakenAt.Sub(prev.TakenAt)
		sectionLabel = fmt.Sprintf("  Growth since last scan %s:", ui.Dim("("+formatDuration(span.Hours()/24)+")"))
	}

	type chgEntry struct {
		path  string
		delta int64
	}
	var movers []chgEntry
	for path, after := range latest.Dirs {
		if isSkipped(path) {
			continue
		}
		before := prev.Dirs[path]
		delta := after - before
		if delta > 1<<20 { // only show dirs that grew >1MB
			movers = append(movers, chgEntry{path, delta})
		}
	}
	sort.Slice(movers, func(i, j int) bool { return movers[i].delta > movers[j].delta })
	if len(movers) > 8 {
		movers = movers[:8]
	}

	fmt.Println(ui.Header(sectionLabel))
	if len(movers) == 0 {
		fmt.Println(ui.Dim("  No significant changes."))
	} else {
		ui.PrintTable(
			[]ui.Column{{Header: "Directory"}, {Header: "Growth", RightAlign: true}},
			func() [][]string {
				rows := make([][]string, len(movers))
				for i, e := range movers {
					rows[i] = []string{e.path, ui.FormatChange(e.delta)}
				}
				return rows
			}(),
		)
	}
	fmt.Println()
	return nil
}
