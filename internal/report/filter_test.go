package report

import "testing"

func TestLeafFilter(t *testing.T) {
	entries := []Change{
		{Path: "/home", Delta: 100},
		{Path: "/home/user", Delta: 100},
	}
	out := LeafFilter(entries)
	if len(out) != 1 || out[0].Path != "/home/user" {
		t.Fatalf("got %+v", out)
	}
}

func TestPathSkipped(t *testing.T) {
	if !PathSkipped("/proc/1", []string{"/proc"}) {
		t.Fatal("expected skip")
	}
}
