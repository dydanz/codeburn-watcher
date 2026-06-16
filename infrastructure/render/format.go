package render

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
)

// FmtTokens formats token count with K/M suffix.
func FmtTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// FmtCost formats USD cost with 4 decimal places.
func FmtCost(usd float64) string {
	return fmt.Sprintf("$%.4f", usd)
}

// FmtPct formats a 0–1 ratio as a percentage string.
func FmtPct(ratio float64) string {
	return fmt.Sprintf("%.1f%%", ratio*100)
}

// FmtTrendValue formats a trend ratio as a signed percentage change.
func FmtTrendValue(ratio float64) string {
	pct := (ratio - 1.0) * 100
	if pct > 0 {
		return fmt.Sprintf("+%.1f%%", pct)
	}
	return fmt.Sprintf("%.1f%%", pct)
}

// Bar renders an ASCII progress bar of width w representing ratio r (0–1).
func Bar(r float64, w int) string {
	if r < 0 {
		r = 0
	}
	if r > 1 {
		r = 1
	}
	filled := int(r * float64(w))
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", w-filled) + "]"
}

// padRight pads s to width w using Unicode-aware rune widths.
func padRight(s string, w int) string {
	sw := runewidth.StringWidth(s)
	if sw >= w {
		return s
	}
	return s + strings.Repeat(" ", w-sw)
}

// Table renders a simple table with header + rows.
// cols is the list of column widths.
func Table(header []string, rows [][]string, cols []int) string {
	var sb strings.Builder
	sep := strings.Repeat("─", 80) + "\n"

	// header
	sb.WriteString(sep)
	for i, h := range header {
		w := 12
		if i < len(cols) {
			w = cols[i]
		}
		sb.WriteString(padRight(h, w) + "  ")
	}
	sb.WriteString("\n" + sep)

	for _, row := range rows {
		for i, cell := range row {
			w := 12
			if i < len(cols) {
				w = cols[i]
			}
			sb.WriteString(padRight(cell, w) + "  ")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
