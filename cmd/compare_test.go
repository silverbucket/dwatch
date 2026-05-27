package cmd

import (
	"testing"
	"time"

	"dwatch/internal/store"
)

func TestResolveComparePairSince(t *testing.T) {
	dir := t.TempDir()
	dataDir = dir

	mk := func(at time.Time) {
		s := &store.Snapshot{
			TakenAt: at,
			Root:    "/home/user",
			Depth:   5,
			Dirs:    map[string]int64{"/home/user": 100},
		}
		if _, err := store.Save(dir, s); err != nil {
			t.Fatal(err)
		}
	}

	mk(time.Date(2026, 5, 11, 8, 0, 0, 0, time.UTC))
	mk(time.Date(2026, 5, 18, 8, 0, 0, 0, time.UTC))
	mk(time.Date(2026, 5, 19, 8, 0, 0, 0, time.UTC))

	pair, err := resolveComparePair("1w")
	if err != nil {
		t.Fatal(err)
	}
	if pair.latest.TakenAt.Day() != 19 {
		t.Fatalf("latest day %d", pair.latest.TakenAt.Day())
	}
	if pair.baseline == pair.latest {
		t.Fatal("baseline must not equal latest")
	}
	if pair.baseline.TakenAt.After(pair.latest.TakenAt) {
		t.Fatal("baseline must be older than latest")
	}
}

func TestSkipForCompareIncludesSnapshotSkip(t *testing.T) {
	latest := &store.Snapshot{Skip: []string{"/Volumes/Backup"}}
	baseline := &store.Snapshot{}
	skips := skipForCompare(latest, baseline)
	found := false
	for _, p := range skips {
		if p == "/Volumes/Backup" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected snapshot skip in %v", skips)
	}
}

func TestPathSkipped(t *testing.T) {
	latest := &store.Snapshot{Skip: []string{"/custom"}}
	if !pathSkipped("/custom/sub", latest, nil) {
		t.Fatal("expected skip")
	}
	if pathSkipped("/other", latest, nil) {
		t.Fatal("expected not skipped")
	}
}
