package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	Cyan    = lipgloss.Color("86")
	Green   = lipgloss.Color("82")
	Yellow  = lipgloss.Color("226")
	Red     = lipgloss.Color("196")
	Magenta = lipgloss.Color("213")
	Gray    = lipgloss.Color("245")
	DimGray = lipgloss.Color("239")
	White   = lipgloss.Color("255")

	// Text styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Cyan)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(Gray)

	SelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Cyan)

	NormalStyle = lipgloss.NewStyle().
			Foreground(White)

	DimStyle = lipgloss.NewStyle().
			Foreground(DimGray)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(Green)

	WarningStyle = lipgloss.NewStyle().
			Foreground(Yellow)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Red)

	InfoStyle = lipgloss.NewStyle().
			Foreground(Magenta)

	// Box styles
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Gray).
			Padding(0, 1)

	SelectedBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Cyan).
				Padding(0, 1)

	WarningBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Yellow).
			Padding(0, 1)

	// Footer style
	FooterStyle = lipgloss.NewStyle().
			Foreground(Gray).
			MarginTop(1)

	// Key style
	KeyStyle = lipgloss.NewStyle().
			Foreground(Cyan).
			Bold(true)

	// Metric styles
	MetricLabelStyle = lipgloss.NewStyle().
				Foreground(Gray).
				Width(12)

	MetricValueStyle = lipgloss.NewStyle().
				Foreground(White)

	MetricSavedStyle = lipgloss.NewStyle().
				Foreground(Green)
)

// RenderKey renders a keyboard shortcut
func RenderKey(key, desc string) string {
	return KeyStyle.Render("["+key+"]") + " " + NormalStyle.Render(desc)
}

// RenderMetric renders a before/after metric
func RenderMetric(label string, before, after int, unit string) string {
	saved := before - after
	savedStr := ""
	if saved > 0 {
		savedStr = MetricSavedStyle.Render(" (-" + itoa(saved) + ")")
	}
	return MetricLabelStyle.Render(label+":") +
		DimStyle.Render(itoa(before)) +
		NormalStyle.Render(" -> ") +
		MetricValueStyle.Render(itoa(after)+unit) +
		savedStr
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	negative := i < 0
	if negative {
		i = -i
	}
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	if negative {
		s = "-" + s
	}
	return s
}
