package report

import "strings"

// Change is a directory path and signed byte delta between two snapshots.
type Change struct {
	Path  string
	Delta int64
}

// PathSkipped reports whether path matches any skip prefix (exact or child path).
func PathSkipped(path string, skipList []string) bool {
	for _, skip := range skipList {
		if path == skip || strings.HasPrefix(path, skip+"/") {
			return true
		}
	}
	return false
}

// LeafFilter removes entries whose change is fully accounted for by the sum of
// their descendants in the set.
func LeafFilter(entries []Change) []Change {
	deltas := make(map[string]int64, len(entries))
	for _, e := range entries {
		deltas[e.Path] = e.Delta
	}
	out := make([]Change, 0, len(entries))
	for _, e := range entries {
		prefix := e.Path
		if prefix != "/" {
			prefix += "/"
		}
		var childSum int64
		for other, d := range deltas {
			if other != e.Path && strings.HasPrefix(other, prefix) {
				childSum += d
			}
		}
		if e.Delta > 0 && childSum >= e.Delta {
			continue
		}
		if e.Delta < 0 && childSum <= e.Delta {
			continue
		}
		out = append(out, e)
	}
	return out
}
