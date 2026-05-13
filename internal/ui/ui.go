package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	styleBold   = lipgloss.NewStyle().Bold(true)
	styleDim    = lipgloss.NewStyle().Faint(true)
	styleRed    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	styleYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	styleCyan   = lipgloss.NewStyle().Foreground(lipgloss.Color("51"))
	styleHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("147"))
)

func Bold(s string) string    { return styleBold.Render(s) }
func Dim(s string) string     { return styleDim.Render(s) }
func Header(s string) string  { return styleHeader.Render(s) }
func Cyan(s string) string    { return styleCyan.Render(s) }

func FormatSize(bytes int64) string {
	if bytes < 0 {
		return "-" + FormatSize(-bytes)
	}
	const (
		KB = int64(1) << 10
		MB = int64(1) << 20
		GB = int64(1) << 30
		TB = int64(1) << 40
	)
	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.0f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func FormatChange(delta int64) string {
	if delta == 0 {
		return styleDim.Render("  —  ")
	}
	s := FormatSize(int64(math.Abs(float64(delta))))
	if delta > 0 {
		return styleRed.Render("+" + s)
	}
	return styleGreen.Render("-" + s)
}

func FormatPct(pct float64) string {
	if math.IsInf(pct, 0) || math.IsNaN(pct) {
		return styleDim.Render("   —  ")
	}
	s := fmt.Sprintf("%+.0f%%", pct)
	switch {
	case pct > 100:
		return styleRed.Render(s)
	case pct > 20:
		return styleYellow.Render(s)
	case pct < 0:
		return styleGreen.Render(s)
	default:
		return s
	}
}

func FormatRate(bytesPerDay float64) string {
	if bytesPerDay <= 0 {
		return styleDim.Render("  —  ")
	}
	return styleRed.Render(FormatSize(int64(bytesPerDay)) + "/d")
}

func Num(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

// visLen returns the visible (non-ANSI) length of a string.
func visLen(s string) int {
	n := 0
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++
		} else {
			n++
			i++
		}
	}
	return n
}

func padRight(s string, totalWidth, visWidth int) string {
	pad := totalWidth - visWidth
	if pad < 0 {
		pad = 0
	}
	return s + strings.Repeat(" ", pad)
}

type Column struct {
	Header     string
	RightAlign bool
}

// PrintTable renders a table with ANSI-aware column widths.
func PrintTable(cols []Column, rows [][]string) {
	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = len(c.Header)
	}
	for _, row := range rows {
		for i := range cols {
			if i < len(row) {
				if w := visLen(row[i]); w > widths[i] {
					widths[i] = w
				}
			}
		}
	}

	// header
	for i, c := range cols {
		cell := styleBold.Render(c.Header)
		fmt.Print(padRight(cell, widths[i]+3, len(c.Header)))
	}
	fmt.Println()

	// separator
	total := 0
	for _, w := range widths {
		total += w + 3
	}
	fmt.Println(styleDim.Render(strings.Repeat("─", total)))

	// rows
	for _, row := range rows {
		for i := range cols {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			vl := visLen(cell)
			if cols[i].RightAlign {
				pad := widths[i] - vl
				if pad < 0 {
					pad = 0
				}
				fmt.Print(strings.Repeat(" ", pad) + cell + "   ")
			} else {
				fmt.Print(padRight(cell, widths[i]+3, vl))
			}
		}
		fmt.Println()
	}
}

// SeparatorLine prints a horizontal separator.
func SeparatorLine(width int) {
	fmt.Println(styleDim.Render(strings.Repeat("─", width)))
}
