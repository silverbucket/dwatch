package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/spf13/cobra"

	"dwatch/internal/report"
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
	statusCmd.Flags().StringVar(&statusSince, "since", "", "compare growth over this window (e.g. 30min, 1h, 2d, 1w, 1m)")
	statusCmd.Flags().IntVarP(&statusLimit, "limit", "l", 5, "max entries per section")
}

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

	type entry struct {
		path string
		size int64
	}
	allDirs := make([]entry, 0, len(latest.Dirs))
	sortedPaths := make([]string, 0, len(latest.Dirs))
	for p, s := range latest.Dirs {
		if pathSkipped(p, latest, nil) {
			continue
		}
		allDirs = append(allDirs, entry{p, s})
		sortedPaths = append(sortedPaths, p)
	}
	sort.Strings(sortedPaths)

	largest := make([]entry, 0, len(allDirs))
	for _, e := range allDirs {
		prefix := e.path
		if prefix != "/" {
			prefix += "/"
		}
		pos := sort.SearchStrings(sortedPaths, prefix)
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

	if len(snaps) < 2 {
		fmt.Println(ui.Dim("  (Take more snapshots to see growth trends)"))
		fmt.Println()
		return nil
	}

	pair, err := resolveComparePair(statusSince)
	if err != nil {
		return err
	}
	prev := pair.baseline

	var growthLabel, freedLabel string
	if statusSince != "" {
		if pair.note != "" {
			note := ui.Dim("(oldest snapshot: " + formatDuration(time.Since(prev.TakenAt).Hours()/24) + ")")
			growthLabel = fmt.Sprintf("  Growth since %s %s:", statusSince, note)
			freedLabel = fmt.Sprintf("  Freed since %s %s:", statusSince, note)
		} else {
			growthLabel = fmt.Sprintf("  Growth since %s:", statusSince)
			freedLabel = fmt.Sprintf("  Freed since %s:", statusSince)
		}
	} else {
		span := latest.TakenAt.Sub(prev.TakenAt)
		note := ui.Dim("(" + formatDuration(span.Hours()/24) + ")")
		growthLabel = fmt.Sprintf("  Growth since last scan %s:", note)
		freedLabel = fmt.Sprintf("  Freed since last scan %s:", note)
	}

	var growChanges, shrinkChanges []report.Change
	for path, after := range latest.Dirs {
		if pathSkipped(path, latest, prev) {
			continue
		}
		before := prev.Dirs[path]
		delta := after - before
		if delta > 1<<20 {
			growChanges = append(growChanges, report.Change{Path: path, Delta: delta})
		} else if delta < -(1 << 20) {
			shrinkChanges = append(shrinkChanges, report.Change{Path: path, Delta: delta})
		}
	}

	growChanges = report.LeafFilter(growChanges)
	shrinkChanges = report.LeafFilter(shrinkChanges)

	sort.Slice(growChanges, func(i, j int) bool { return growChanges[i].Delta > growChanges[j].Delta })
	if len(growChanges) > statusLimit {
		growChanges = growChanges[:statusLimit]
	}

	sort.Slice(shrinkChanges, func(i, j int) bool { return shrinkChanges[i].Delta < shrinkChanges[j].Delta })
	if len(shrinkChanges) > statusLimit {
		shrinkChanges = shrinkChanges[:statusLimit]
	}

	fmt.Println(ui.Header(growthLabel))
	if len(growChanges) == 0 {
		fmt.Println(ui.Dim("  No significant growth."))
	} else {
		ui.PrintTable(
			[]ui.Column{{Header: "Directory"}, {Header: "Growth", RightAlign: true}},
			func() [][]string {
				rows := make([][]string, len(growChanges))
				for i, e := range growChanges {
					rows[i] = []string{e.Path, ui.FormatChange(e.Delta)}
				}
				return rows
			}(),
		)
	}
	fmt.Println()

	if len(shrinkChanges) > 0 {
		fmt.Println(ui.Header(freedLabel))
		ui.PrintTable(
			[]ui.Column{{Header: "Directory"}, {Header: "Freed", RightAlign: true}},
			func() [][]string {
				rows := make([][]string, len(shrinkChanges))
				for i, e := range shrinkChanges {
					rows[i] = []string{e.Path, ui.FormatChange(e.Delta)}
				}
				return rows
			}(),
		)
		fmt.Println()
	}

	return nil
}
