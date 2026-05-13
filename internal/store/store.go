package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Snapshot struct {
	TakenAt time.Time        `json:"taken_at"`
	Root    string           `json:"root"`
	Depth   int              `json:"depth"`
	Dirs    map[string]int64 `json:"dirs"`
}

func Save(dataDir string, snap *Snapshot) (string, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", err
	}
	fname := fmt.Sprintf("snap_%s.json", snap.TakenAt.UTC().Format("20060102_150405"))
	path := filepath.Join(dataDir, fname)
	data, err := json.Marshal(snap)
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, data, 0644)
}

func List(dataDir string) ([]*Snapshot, error) {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var snaps []*Snapshot
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		snap, err := load(filepath.Join(dataDir, e.Name()))
		if err != nil {
			continue
		}
		snaps = append(snaps, snap)
	}

	sort.Slice(snaps, func(i, j int) bool {
		return snaps[i].TakenAt.Before(snaps[j].TakenAt)
	})

	return snaps, nil
}

func Latest(dataDir string) (*Snapshot, error) {
	snaps, err := List(dataDir)
	if err != nil || len(snaps) == 0 {
		return nil, err
	}
	return snaps[len(snaps)-1], nil
}

// LatestBefore returns the newest snapshot taken before t, or nil if none exists.
func LatestBefore(dataDir string, t time.Time) (*Snapshot, error) {
	snaps, err := List(dataDir)
	if err != nil {
		return nil, err
	}
	var found *Snapshot
	for _, s := range snaps {
		if s.TakenAt.Before(t) {
			found = s
		}
	}
	return found, nil
}

// Previous returns the snapshot just before the latest one, or nil.
func Previous(dataDir string) (*Snapshot, error) {
	snaps, err := List(dataDir)
	if err != nil || len(snaps) < 2 {
		return nil, err
	}
	return snaps[len(snaps)-2], nil
}

func load(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snap Snapshot
	return &snap, json.Unmarshal(data, &snap)
}
