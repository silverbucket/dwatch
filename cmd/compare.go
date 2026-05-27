package cmd

import (
	"fmt"

	"dwatch/internal/report"
	"dwatch/internal/store"
)

// comparePair holds the latest snapshot and the baseline used for deltas.
type comparePair struct {
	latest   *store.Snapshot
	baseline *store.Snapshot
	note     string // set when falling back to oldest snapshot
}

// resolveComparePair selects latest and baseline snapshots for reporting commands.
func resolveComparePair(since string) (*comparePair, error) {
	snaps, err := store.List(dataDir)
	if err != nil {
		return nil, fmt.Errorf("could not read snapshots: %w", err)
	}
	if len(snaps) == 0 {
		return nil, fmt.Errorf("no snapshots found — run 'dwatch scan' first")
	}
	if len(snaps) < 2 {
		return nil, fmt.Errorf("no earlier snapshot found — take more scans first")
	}

	latest := snaps[len(snaps)-1]
	candidates := snaps[:len(snaps)-1]

	if since == "" {
		return &comparePair{
			latest:   latest,
			baseline: candidates[len(candidates)-1],
		}, nil
	}

	cutoff, err := parseSince(since)
	if err != nil {
		return nil, err
	}

	var baseline *store.Snapshot
	for _, s := range candidates {
		if !s.TakenAt.After(cutoff) {
			baseline = s
		}
	}

	pair := &comparePair{latest: latest, baseline: baseline}
	if baseline == nil {
		pair.baseline = candidates[0]
		pair.note = "oldest snapshot"
	}
	return pair, nil
}

// skipForCompare merges config and per-snapshot skip paths.
func skipForCompare(latest, baseline *store.Snapshot) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(paths []string) {
		for _, p := range paths {
			if p == "" {
				continue
			}
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			out = append(out, p)
		}
	}
	add(cfg.ScanSkip)
	if baseline != nil {
		add(baseline.Skip)
	}
	if latest != nil {
		add(latest.Skip)
	}
	return out
}

func pathSkipped(path string, latest, baseline *store.Snapshot) bool {
	return report.PathSkipped(path, skipForCompare(latest, baseline))
}
