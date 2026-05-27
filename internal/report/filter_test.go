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
	cases := []struct {
		path  string
		skips []string
		want  bool
	}{
		{path: "/proc/1", skips: []string{"/proc"}, want: true},
		{path: "/proc/1", skips: []string{"/proc/"}, want: true},
		{path: "/home/user", skips: []string{"/"}, want: true},
		{path: "/", skips: []string{"/"}, want: true},
		{path: "/procx", skips: []string{"/proc"}, want: false},
	}
	for _, tc := range cases {
		if got := PathSkipped(tc.path, tc.skips); got != tc.want {
			t.Fatalf("PathSkipped(%q, %v) = %v, want %v", tc.path, tc.skips, got, tc.want)
		}
	}
}
