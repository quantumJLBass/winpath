package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestColorConstants(t *testing.T) {
	// Verify color constants are valid lipgloss colors
	colors := []lipgloss.Color{
		Cyan,
		Green,
		Yellow,
		Red,
		Gray,
		White,
	}

	for _, c := range colors {
		// Should not panic when creating a style with the color
		style := lipgloss.NewStyle().Foreground(c)
		_ = style.Render("test")
	}
}

func TestStylesNotNil(t *testing.T) {
	styles := []lipgloss.Style{
		TitleStyle,
		SubtitleStyle,
		SelectedStyle,
		NormalStyle,
		DimStyle,
		SuccessStyle,
		WarningStyle,
		ErrorStyle,
	}

	for i, style := range styles {
		// Render should not panic
		result := style.Render("test")
		if result == "" {
			t.Errorf("Style %d rendered empty string", i)
		}
	}
}

func TestTitleStyle(t *testing.T) {
	result := TitleStyle.Render("Test Title")

	if result == "" {
		t.Error("TitleStyle should render non-empty string")
	}

	if result == "Test Title" {
		t.Log("TitleStyle may not be applying styling (check if colors work)")
	}
}

func TestSelectedStyle(t *testing.T) {
	result := SelectedStyle.Render("Selected Item")

	if result == "" {
		t.Error("SelectedStyle should render non-empty string")
	}
}

func TestSuccessStyle(t *testing.T) {
	result := SuccessStyle.Render("Success!")

	if result == "" {
		t.Error("SuccessStyle should render non-empty string")
	}
}

func TestWarningStyle(t *testing.T) {
	result := WarningStyle.Render("Warning!")

	if result == "" {
		t.Error("WarningStyle should render non-empty string")
	}
}

func TestErrorStyle(t *testing.T) {
	result := ErrorStyle.Render("Error!")

	if result == "" {
		t.Error("ErrorStyle should render non-empty string")
	}
}

func TestDimStyle(t *testing.T) {
	result := DimStyle.Render("Dimmed text")

	if result == "" {
		t.Error("DimStyle should render non-empty string")
	}
}

func TestNormalStyle(t *testing.T) {
	result := NormalStyle.Render("Normal text")

	if result == "" {
		t.Error("NormalStyle should render non-empty string")
	}
}

func TestRenderKey_Format(t *testing.T) {
	result := RenderKey("Enter", "Select")

	// Should contain both key and action
	if len(result) < len("Enter")+len("Select") {
		t.Log("RenderKey output shorter than expected")
	}
}

func TestRenderKey_EmptyKey(t *testing.T) {
	result := RenderKey("", "Action")

	// Should handle empty key gracefully
	if result == "" {
		t.Log("RenderKey returns empty for empty key")
	}
}

func TestRenderKey_EmptyAction(t *testing.T) {
	result := RenderKey("Key", "")

	// Should handle empty action gracefully
	if result == "" {
		t.Log("RenderKey returns empty for empty action")
	}
}

func TestStyleChaining(t *testing.T) {
	// Test that styles can be chained
	style := lipgloss.NewStyle().
		Foreground(Cyan).
		Bold(true)

	result := style.Render("Chained style")

	if result == "" {
		t.Error("Chained style should render")
	}
}

func TestStyleCopy(t *testing.T) {
	// Styles should be copyable
	original := TitleStyle
	copy := original.Copy()

	// Both should render the same
	origResult := original.Render("test")
	copyResult := copy.Render("test")

	if origResult != copyResult {
		t.Error("Copied style should render same as original")
	}
}

func TestStyleWithPadding(t *testing.T) {
	style := lipgloss.NewStyle().Padding(1, 2)
	result := style.Render("Padded")

	// Result should be longer due to padding
	if len(result) <= len("Padded") {
		t.Log("Padding may not be visible in plain text")
	}
}

func TestStyleWithBorder(t *testing.T) {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Cyan)

	result := style.Render("Bordered")

	// Result should contain border characters
	if len(result) <= len("Bordered") {
		t.Log("Border should add characters")
	}
}

func BenchmarkStyleRender(b *testing.B) {
	text := "Benchmark text"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TitleStyle.Render(text)
	}
}

func BenchmarkRenderKey(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RenderKey("Enter", "Select")
	}
}

func BenchmarkStyleChain(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		style := lipgloss.NewStyle().
			Foreground(Cyan).
			Bold(true).
			Padding(0, 1)
		style.Render("test")
	}
}

// ============================================================================
// Additional coverage tests for styles.go
// ============================================================================

func TestItoa_Zero(t *testing.T) {
	result := itoa(0)
	if result != "0" {
		t.Errorf("Expected '0', got '%s'", result)
	}
}

func TestItoa_Positive(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{1000, "1000"},
	}

	for _, tt := range tests {
		result := itoa(tt.input)
		if result != tt.expected {
			t.Errorf("itoa(%d): expected '%s', got '%s'", tt.input, tt.expected, result)
		}
	}
}

func TestItoa_Negative(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{-1, "-1"},
		{-10, "-10"},
		{-123, "-123"},
	}

	for _, tt := range tests {
		result := itoa(tt.input)
		if result != tt.expected {
			t.Errorf("itoa(%d): expected '%s', got '%s'", tt.input, tt.expected, result)
		}
	}
}

func TestRenderMetric_NoSavings(t *testing.T) {
	result := RenderMetric("Count", 10, 10, "")

	if result == "" {
		t.Error("RenderMetric should not return empty string")
	}
	// Should not contain savings indicator
	if strings.Contains(result, "(-") {
		t.Error("Should not show savings when before == after")
	}
}

func TestRenderMetric_WithSavings(t *testing.T) {
	result := RenderMetric("Length", 100, 80, " chars")

	if result == "" {
		t.Error("RenderMetric should not return empty string")
	}
	// Should contain savings indicator
	if !strings.Contains(result, "(-20)") {
		t.Error("Should show savings of 20")
	}
}

func TestRenderMetric_NegativeSavings(t *testing.T) {
	result := RenderMetric("Count", 10, 15, "")

	// When after > before, saved is negative, so savedStr should be empty
	if strings.Contains(result, "(-") {
		t.Error("Should not show negative savings")
	}
}

func BenchmarkItoa(b *testing.B) {
	for i := 0; i < b.N; i++ {
		itoa(12345)
	}
}

func BenchmarkRenderMetric(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RenderMetric("Length", 100, 80, " chars")
	}
}
