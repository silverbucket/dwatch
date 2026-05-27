package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"dwatch/internal/config"
	"dwatch/internal/report"
)

// Version is set at build time via -ldflags "-X dwatch/cmd.Version=x.y.z".
var Version = "dev"

var (
	dataDir string
	cfg     config.Config
)

var rootCmd = &cobra.Command{
	Use:     "dwatch",
	Short:   "Track disk space growth over time",
	Version: Version,
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
		var code exitCode
		if errors.As(err, &code) {
			os.Exit(int(code))
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	home, _ := os.UserHomeDir()
	var err error
	cfg, err = config.Load(filepath.Join(home, ".dwatch", "dwatch.conf"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "dwatch: warning: could not read config: %v\n", err)
		cfg = config.Defaults()
	}
	rootCmd.PersistentFlags().StringVarP(&dataDir, "data-dir", "d", cfg.DataDir, "directory to store snapshots")
}

// isSkipped reports whether path is covered by the config skip list alone.
func isSkipped(path string) bool {
	return report.PathSkipped(path, cfg.ScanSkip)
}
