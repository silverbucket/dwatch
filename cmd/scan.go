package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"dwatch/internal/scanner"
	"dwatch/internal/store"
	"dwatch/internal/ui"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Take a snapshot of current directory sizes",
	Example: `  dwatch scan
  dwatch scan --root /Users --depth 4
  dwatch scan --skip /System/Volumes --skip /Volumes/Backup`,
	RunE: runScan,
}

var (
	scanRoot  string
	scanDepth int
	scanSkip  []string
	scanQuiet bool
	scanShow  int
)

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().StringVarP(&scanRoot, "root", "r", cfg.ScanRoot, "root directory to scan")
	scanCmd.Flags().IntVarP(&scanDepth, "depth", "n", cfg.ScanDepth, "max directory depth to track")
	scanCmd.Flags().StringArrayVarP(&scanSkip, "skip", "s", cfg.ScanSkip, "paths to skip")
	scanCmd.Flags().BoolVarP(&scanQuiet, "quiet", "q", false, "suppress output (for cron use)")
	scanCmd.Flags().IntVar(&scanShow, "show", 10, "number of largest dirs to display after scan (0 = all)")
}

func runScan(_ *cobra.Command, _ []string) error {
	if !scanQuiet {
		fmt.Printf("\n  %s  %s  (depth: %d)\n", ui.Header("Scanning"), ui.Cyan(scanRoot), scanDepth)
		fmt.Printf("  %s %s\n\n", ui.Dim("Skip:"), ui.Dim(strings.Join(scanSkip, "  ")))
	}

	start := time.Now()
	result, err := scanner.Scan(scanRoot, scanDepth, scanSkip)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}
	elapsed := time.Since(start)

	snap := &store.Snapshot{
		TakenAt: time.Now(),
		Root:    scanRoot,
		Depth:   scanDepth,
		Dirs:    result.Dirs,
	}

	path, err := store.Save(dataDir, snap)
	if err != nil {
		return fmt.Errorf("save failed: %w", err)
	}

	if scanQuiet {
		fmt.Printf("dwatch: snapshot saved (%s dirs, %.1fs)\n", ui.Num(len(result.Dirs)), elapsed.Seconds())
		return nil
	}

	fmt.Printf("  Done in %.1fs — %s directories indexed\n", elapsed.Seconds(), ui.Bold(ui.Num(len(result.Dirs))))
	fmt.Printf("  Saved: %s\n\n", ui.Dim(path))

	// Top 10 largest
	type entry struct {
		path string
		size int64
	}
	top := make([]entry, 0, len(result.Dirs))
	for p, s := range result.Dirs {
		top = append(top, entry{p, s})
	}
	sort.Slice(top, func(i, j int) bool { return top[i].size > top[j].size })
	if scanShow > 0 && len(top) > scanShow {
		top = top[:scanShow]
	}

	fmt.Println(ui.Header("  Largest directories:"))
	ui.PrintTable(
		[]ui.Column{{Header: "Path"}, {Header: "Size", RightAlign: true}},
		func() [][]string {
			rows := make([][]string, len(top))
			for i, e := range top {
				rows[i] = []string{e.path, ui.FormatSize(e.size)}
			}
			return rows
		}(),
	)
	fmt.Println()
	return nil
}
