package cmd

import (
	"testing"
	"time"
)

func TestParseBytes(t *testing.T) {
	tests := []struct {
		in   string
		want int64
	}{
		{"500mb", 500 << 20},
		{"500MB", 500 << 20},
		{"1gb", 1 << 30},
		{"1GB", 1 << 30},
		{"100kb", 100 << 10},
		{"2.5mb", int64(2.5 * float64(1<<20))},
	}
	for _, tc := range tests {
		got, err := parseBytes(tc.in)
		if err != nil {
			t.Fatalf("parseBytes(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("parseBytes(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestParseBytesMixedCase(t *testing.T) {
	got, err := parseBytes("500Mb")
	if err != nil {
		t.Fatal(err)
	}
	if got != 500<<20 {
		t.Fatalf("got %d", got)
	}
}

func TestParseBytesInvalid(t *testing.T) {
	if _, err := parseBytes("x"); err == nil {
		t.Fatal("expected error for invalid size")
	}
}

func TestParseSinceMinutes(t *testing.T) {
	before := time.Now()
	cutoff, err := parseSince("30min")
	if err != nil {
		t.Fatal(err)
	}
	elapsed := before.Sub(cutoff)
	if elapsed < 29*time.Minute || elapsed > 31*time.Minute {
		t.Fatalf("30min cutoff off by %v", elapsed)
	}
}

func TestParseSinceMonths(t *testing.T) {
	cutoff, err := parseSince("1m")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Now().AddDate(0, -1, 0)
	if cutoff.Sub(want) > time.Minute || want.Sub(cutoff) > time.Minute {
		t.Fatalf("1m cutoff %v, want ~%v", cutoff, want)
	}

	cutoffMo, err := parseSince("2mo")
	if err != nil {
		t.Fatal(err)
	}
	wantMo := time.Now().AddDate(0, -2, 0)
	if cutoffMo.Sub(wantMo) > time.Minute || wantMo.Sub(cutoffMo) > time.Minute {
		t.Fatalf("2mo cutoff %v, want ~%v", cutoffMo, wantMo)
	}
}

func TestParseSinceDate(t *testing.T) {
	cutoff, err := parseSince("2026-05-01")
	if err != nil {
		t.Fatal(err)
	}
	if cutoff.Year() != 2026 || cutoff.Month() != 5 || cutoff.Day() != 1 {
		t.Fatalf("unexpected date: %v", cutoff)
	}
}
