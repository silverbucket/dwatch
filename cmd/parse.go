package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const parseSinceHelp = "use 30min, 1h, 2d, 3w, 1m (months), 1mo, or YYYY-MM-DD"

// parseSince converts a human duration string to an absolute cutoff time.
// Durations: min (minutes), h, d, w, m/mo (months). Dates: YYYY-MM-DD.
func parseSince(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return time.Time{}, fmt.Errorf("invalid format %q — %s", s, parseSinceHelp)
	}

	if strings.HasSuffix(s, "min") {
		n, err := strconv.Atoi(s[:len(s)-3])
		if err != nil || n < 0 {
			return time.Time{}, fmt.Errorf("invalid format %q — %s", s, parseSinceHelp)
		}
		return time.Now().Add(-time.Duration(n) * time.Minute), nil
	}

	if strings.HasSuffix(s, "mo") {
		n, err := strconv.Atoi(s[:len(s)-2])
		if err != nil || n < 0 {
			return time.Time{}, fmt.Errorf("invalid format %q — %s", s, parseSinceHelp)
		}
		return time.Now().AddDate(0, -n, 0), nil
	}

	n, err := strconv.Atoi(s[:len(s)-1])
	if err != nil || n < 0 {
		return tryParseSinceDate(s)
	}

	switch s[len(s)-1] {
	case 'h':
		return time.Now().Add(-time.Duration(n) * time.Hour), nil
	case 'd':
		return time.Now().AddDate(0, 0, -n), nil
	case 'w':
		return time.Now().AddDate(0, 0, -n*7), nil
	case 'm':
		return time.Now().AddDate(0, -n, 0), nil
	default:
		return tryParseSinceDate(s)
	}
}

func tryParseSinceDate(s string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid format %q — %s", s, parseSinceHelp)
	}
	return t, nil
}

func parseBytes(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid size %q — use 100mb, 1gb, 500kb", s)
	}

	lower := strings.ToLower(s)
	suffixes := []struct {
		suffix string
		mult   int64
	}{
		{"tb", 1 << 40},
		{"gb", 1 << 30},
		{"mb", 1 << 20},
		{"kb", 1 << 10},
		{"t", 1 << 40},
		{"g", 1 << 30},
		{"m", 1 << 20},
		{"k", 1 << 10},
	}

	for _, su := range suffixes {
		if !strings.HasSuffix(lower, su.suffix) {
			continue
		}
		numStr := lower[:len(lower)-len(su.suffix)]
		n, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid size %q", s)
		}
		return int64(n * float64(su.mult)), nil
	}

	n, err := strconv.ParseInt(lower, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q — use 100mb, 1gb, 500kb", s)
	}
	return n, nil
}
