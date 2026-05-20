package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"golang.org/x/sys/unix"

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
var statusLimit int

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringVar(&statusSince, "since", "", "compare growth over this window (e.g. 1h, 2d, 1w, 1m)")
	statusCmd.Flags().IntVarP(&statusLimit, "limit", "l", 5, "max entries per section")
}

// diskUsage reports the total, used, and available bytes for the filesystem containing path.
// It returns the total size, used size, and available size (in bytes). If retrieving filesystem
// statistics fails, the returned error describes the failure.
func diskUsage(path string) (total, used, avail uint64, err error) {
	var stat unix.Statfs_t
	if err = unix.Statfs(path, &stat); err != nil {
		return
	}
	bsize := uint64(stat.Bsize)
	total = stat.Blocks * bsize
	avail = stat.Bavail * bsize
	used = (stat.Blocks - stat.Bfree) * bsize
	return
}

type chgEntry struct {
	path  string
	delta int64
}

// leafFilter removes entries whose change is fully accounted for by the sum of
// their descendants in the set. A parent is kept when descendants explain less
// than its total delta (meaning growth/shrink at this level is not attributable
// to any tracked child alone).
func leafFilter(entries []chgEntry) []chgEntry {
	deltas := make(map[string]int64, len(entries))
	for _, e := range entries {
		deltas[e.path] = e.delta
	}
	out := make([]chgEntry, 0, len(entries))
	for _, e := range entries {
		prefix := e.path
		if prefix != "/" {
			prefix += "/"
		}
		var childSum int64
		for other, d := range deltas {
			if other != e.path && strings.HasPrefix(other, prefix) {
				childSum += d
			}
		}
		// Drop only when descendants fully account for the change.
		if e.delta > 0 && childSum >= e.delta {
			continue
		}
		if e.delta < 0 && childSum <= e.delta {
			continue
		}
		out = append(out, e)
	}
	return out
}

// runStatus prints a concise status report for the latest snapshot and growth since a baseline.
//
// It loads stored snapshots, displays the latest snapshot's timestamp, root, depth, tracked
// directory count and total snapshots, and — when available — disk usage for the snapshot root.
// It then shows the top current largest directories and, if a prior snapshot exists, the biggest
// growths since either the previous snapshot or the snapshot selected by the `--since` flag.
// It returns an error if snapshot listing fails, if no snapshots exist, or if the `--since`
// flag cannot be parsed.
func runStatus(_ *cobra.Command, _ []string) error {
	if statusLimit < 0 {
		return fmt.Errorf("--limit must be >= 0")
	}
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
	fmt.Printf("  Snapshots: %s total\n", ui.Num(len(snaps)))
	if total, used, avail, err := diskUsage(latest.Root); err == nil {
		fmt.Printf("  Disk:      %s used · %s free  (%s total)\n",
			ui.FormatSize(int64(used)),
			ui.FormatSize(int64(avail)),
			ui.FormatSize(int64(total)))
	}
	fmt.Println()

	// Top 8 largest leaf directories (ancestors filtered out).
	type entry struct {
		path string
		size int64
	}
	allDirs := make([]entry, 0, len(latest.Dirs))
	sortedPaths := make([]string, 0, len(latest.Dirs))
	for p, s := range latest.Dirs {
		if isSkipped(p) {
			continue
		}
		allDirs = append(allDirs, entry{p, s})
		sortedPaths = append(sortedPaths, p)
	}
	sort.Strings(sortedPaths)

	// Binary-search for a child: O(N log N) over the full directory set.
	largest := make([]entry, 0, len(allDirs))
	for _, e := range allDirs {
		prefix := e.path
		if prefix != "/" {
			prefix += "/"
		}
		pos := sort.SearchStrings(sortedPaths, prefix)
		// SearchStrings can land on the path itself when prefix == path (root "/").
		// Advance past it so we're only checking for actual children.
		if pos < len(sortedPaths) && sortedPaths[pos] == e.path {
			pos++
		}
		if pos >= len(sortedPaths) || !strings.HasPrefix(sortedPaths[pos], prefix) {
			largest = append(largest, e)
		}
	}

	sort.Slice(largest, func(i, j int) bool { return largest[i].size > largest[j].size })
	if len(largest) > statusLimit {
		largest = largest[:statusLimit]
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
	var growthLabel, freedLabel string

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
			note := ui.Dim("(oldest snapshot: " + formatDuration(time.Since(prev.TakenAt).Hours()/24) + ")")
			growthLabel = fmt.Sprintf("  Growth since %s %s:", statusSince, note)
			freedLabel = fmt.Sprintf("  Freed since %s %s:", statusSince, note)
		} else {
			growthLabel = fmt.Sprintf("  Growth since %s:", statusSince)
			freedLabel = fmt.Sprintf("  Freed since %s:", statusSince)
		}
	} else {
		prev = candidates[len(candidates)-1]
		span := latest.TakenAt.Sub(prev.TakenAt)
		note := ui.Dim("(" + formatDuration(span.Hours()/24) + ")")
		growthLabel = fmt.Sprintf("  Growth since last scan %s:", note)
		freedLabel = fmt.Sprintf("  Freed since last scan %s:", note)
	}

	var growers, shrinkers []chgEntry
	for path, after := range latest.Dirs {
		if isSkipped(path) {
			continue
		}
		before := prev.Dirs[path]
		delta := after - before
		if delta > 1<<20 {
			growers = append(growers, chgEntry{path, delta})
		} else if delta < -(1 << 20) {
			shrinkers = append(shrinkers, chgEntry{path, delta})
		}
	}

	growers = leafFilter(growers)
	shrinkers = leafFilter(shrinkers)

	sort.Slice(growers, func(i, j int) bool { return growers[i].delta > growers[j].delta })
	if len(growers) > statusLimit {
		growers = growers[:statusLimit]
	}

	sort.Slice(shrinkers, func(i, j int) bool { return shrinkers[i].delta < shrinkers[j].delta })
	if len(shrinkers) > statusLimit {
		shrinkers = shrinkers[:statusLimit]
	}

	fmt.Println(ui.Header(growthLabel))
	if len(growers) == 0 {
		fmt.Println(ui.Dim("  No significant growth."))
	} else {
		ui.PrintTable(
			[]ui.Column{{Header: "Directory"}, {Header: "Growth", RightAlign: true}},
			func() [][]string {
				rows := make([][]string, len(growers))
				for i, e := range growers {
					rows[i] = []string{e.path, ui.FormatChange(e.delta)}
				}
				return rows
			}(),
		)
	}
	fmt.Println()

	if len(shrinkers) > 0 {
		fmt.Println(ui.Header(freedLabel))
		ui.PrintTable(
			[]ui.Column{{Header: "Directory"}, {Header: "Freed", RightAlign: true}},
			func() [][]string {
				rows := make([][]string, len(shrinkers))
				for i, e := range shrinkers {
					rows[i] = []string{e.path, ui.FormatChange(e.delta)}
				}
				return rows
			}(),
		)
		fmt.Println()
	}

	return nil
}
