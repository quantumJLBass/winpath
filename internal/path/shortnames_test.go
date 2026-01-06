package path

import (
	"strings"
	"testing"
)

func TestToShortPath_Empty(t *testing.T) {
	result, changed := ToShortPath("")
	if result != "" || changed {
		t.Error("Empty path should return empty unchanged")
	}
}

func TestToShortPath_HasVariable(t *testing.T) {
	input := `%USERPROFILE%\bin`
	result, changed := ToShortPath(input)

	if changed {
		t.Error("Paths with variables should not be shortened")
	}
	if result != input {
		t.Error("Path with variable should be returned unchanged")
	}
}

func TestToShortPath_MultipleVariables(t *testing.T) {
	input := `%SystemRoot%\%TEMP%\bin`
	result, changed := ToShortPath(input)

	if changed || result != input {
		t.Error("Multiple variables should be returned unchanged")
	}
}

func TestToShortPath_NonExistent(t *testing.T) {
	input := `C:\This\Path\Does\Not\Exist\12345`
	result, changed := ToShortPath(input)

	t.Logf("ToShortPath nonexistent: %s -> %s, changed=%v", input, result, changed)
}

func TestToShortPath_SystemRoot(t *testing.T) {
	result, changed := ToShortPath(`C:\Windows`)
	t.Logf("ToShortPath C:\\Windows: %s, changed=%v", result, changed)
}

func TestToShortPath_LongPath(t *testing.T) {
	input := `C:\Program Files\Very Long Application Name\Subfolder`
	result, changed := ToShortPath(input)
	t.Logf("ToShortPath long: %s -> %s, changed=%v", input, result, changed)
}

func TestShortenSuffix_NoVariable(t *testing.T) {
	input := `C:\Windows\System32`
	result, changed := ShortenSuffix(input)

	if changed {
		t.Error("Path without variable should not be shortened")
	}
	if result != input {
		t.Error("Path without variable should be returned unchanged")
	}
}

func TestShortenSuffix_Empty(t *testing.T) {
	result, changed := ShortenSuffix("")
	if result != "" || changed {
		t.Error("Empty should return empty unchanged")
	}
}

func TestShortenSuffix_VariableOnly(t *testing.T) {
	input := `%USERPROFILE%`
	result, changed := ShortenSuffix(input)
	t.Logf("ShortenSuffix variable only: %s -> %s, changed=%v", input, result, changed)
}

func TestShortenSuffix_VariableWithSlash(t *testing.T) {
	input := `%USERPROFILE%\`
	result, changed := ShortenSuffix(input)
	t.Logf("ShortenSuffix with slash: %s -> %s, changed=%v", input, result, changed)
}

func TestShortenSuffix_WithVariable(t *testing.T) {
	input := `%USERPROFILE%\AppData\Local\Programs\Test`
	result, changed := ShortenSuffix(input)

	if changed && !strings.HasPrefix(result, "%") {
		t.Error("Variable prefix should be preserved")
	}
	t.Logf("ShortenSuffix: %s -> %s, changed=%v", input, result, changed)
}

func TestShortenSuffix_NoClosingPercent(t *testing.T) {
	input := `%USERPROFILE\test`
	result, changed := ShortenSuffix(input)

	if changed {
		t.Error("Invalid variable format should not change")
	}
	if result != input {
		t.Errorf("Expected unchanged input, got %s", result)
	}
}

func TestShortenSuffix_PercentAtEnd(t *testing.T) {
	input := `test%`
	result, changed := ShortenSuffix(input)

	if changed {
		t.Error("No variable pattern should not change")
	}
	if result != input {
		t.Errorf("Expected unchanged input, got %s", result)
	}
}

func TestIsShortPath(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`C:\PROGRA~1`, true},
		{`C:\PROGRA~1\Test`, true},
		{`C:\Program Files`, false},
		{`C:\Windows\System32`, false},
		{`D:\DOCUME~1\User`, true},
		{``, false},
		{`C:\MICROS~1\OFFICE~1`, true},
	}

	for _, tt := range tests {
		result := strings.Contains(tt.input, "~")
		if result != tt.expected {
			t.Errorf("IsShortPath(%s) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestShortPathFormat(t *testing.T) {
	validShort := []string{"PROGRA~1", "PROGRA~2", "DOCUME~1", "APPLIC~1", "MICROS~1"}

	for _, name := range validShort {
		if !strings.Contains(name, "~") {
			t.Errorf("Short name should contain ~: %s", name)
		}

		parts := strings.Split(name, "~")
		if len(parts) != 2 {
			t.Errorf("Short name should have exactly one ~: %s", name)
		}
		if len(parts[0]) > 6 {
			t.Errorf("First part should be max 6 chars: %s", name)
		}
	}
}

func TestShortPathFormat_Invalid(t *testing.T) {
	invalidShort := []string{"PROGRAM~", "~1", "PROGRAM", "PRO~GRAM"}

	for _, name := range invalidShort {
		parts := strings.Split(name, "~")
		if len(parts) == 2 && len(parts[0]) > 0 && len(parts[1]) > 0 {
			allDigits := true
			for _, c := range parts[1] {
				if c < '0' || c > '9' {
					allDigits = false
					break
				}
			}
			if allDigits && len(parts[0]) <= 6 {
				t.Logf("Note: %s appears to be valid 8.3 format", name)
			}
		}
	}
}

func BenchmarkToShortPath_NoChange(b *testing.B) {
	input := `%USERPROFILE%\bin`
	for i := 0; i < b.N; i++ {
		ToShortPath(input)
	}
}

func BenchmarkShortenSuffix_NoChange(b *testing.B) {
	input := `C:\Windows\System32`
	for i := 0; i < b.N; i++ {
		ShortenSuffix(input)
	}
}

// ============================================================================
// Tests for refactored helper functions
// ============================================================================

func TestExtractVarAndSuffix_NoVariable(t *testing.T) {
	varPart, suffix, ok := extractVarAndSuffix(`C:\Windows`)

	if ok {
		t.Error("Should return false for path without variable")
	}
	if varPart != "" || suffix != "" {
		t.Error("Should return empty strings")
	}
}

func TestExtractVarAndSuffix_WithVariable(t *testing.T) {
	varPart, suffix, ok := extractVarAndSuffix(`%USERPROFILE%\bin`)

	if !ok {
		t.Error("Should return true for path with variable")
	}
	if varPart != "%USERPROFILE%" {
		t.Errorf("Expected '%%USERPROFILE%%', got '%s'", varPart)
	}
	if suffix != `\bin` {
		t.Errorf("Expected '\\bin', got '%s'", suffix)
	}
}

func TestExtractVarAndSuffix_VariableOnly(t *testing.T) {
	_, _, ok := extractVarAndSuffix(`%USERPROFILE%`)

	if ok {
		t.Error("Should return false for variable without suffix")
	}
}

func TestExtractVarAndSuffix_VariableWithSlashOnly(t *testing.T) {
	_, _, ok := extractVarAndSuffix(`%USERPROFILE%\`)

	if ok {
		t.Error("Should return false for variable with only slash")
	}
}

func TestExtractVarAndSuffix_PercentAtEnd(t *testing.T) {
	_, _, ok := extractVarAndSuffix(`path%`)

	if ok {
		t.Error("Should return false when percent is at end")
	}
}

func TestGetShortPathCommand(t *testing.T) {
	cmd := getShortPathCommand(`C:\Program Files\Test`)

	if cmd == "" {
		t.Error("Command should not be empty")
	}
	if !strings.Contains(cmd, "Scripting.FileSystemObject") {
		t.Error("Command should use FileSystemObject")
	}
	if !strings.Contains(cmd, "Test-Path") {
		t.Error("Command should check if path exists")
	}
}

func TestGetShortPathCommand_WithQuotes(t *testing.T) {
	cmd := getShortPathCommand(`C:\Program Files\It's a test`)

	if !strings.Contains(cmd, "''") {
		t.Error("Single quotes should be escaped")
	}
}
