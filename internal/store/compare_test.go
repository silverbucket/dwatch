package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveUniqueFilenames(t *testing.T) {
	dir := t.TempDir()
	ts := time.Date(2026, 5, 19, 8, 0, 0, 0, time.UTC)
	snap := &Snapshot{
		TakenAt: ts,
		Root:    "/",
		Depth:   1,
		Dirs:    map[string]int64{"/": 100},
	}
	p1, err := Save(dir, snap)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := Save(dir, snap)
	if err != nil {
		t.Fatal(err)
	}
	if p1 == p2 {
		t.Fatalf("expected unique paths, both %s", p1)
	}
	info, err := os.Stat(p2)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("file mode %o, want 0600", info.Mode().Perm())
	}
}

func TestListWarnsOnCorruptJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "snap_bad.json"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	snaps, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 0 {
		t.Fatalf("expected 0 snaps, got %d", len(snaps))
	}
}
