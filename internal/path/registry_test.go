package path

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestSystemPathKey(t *testing.T) {
	expected := `HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Environment`
	if SystemPathKey != expected {
		t.Errorf("SystemPathKey = %s, want %s", SystemPathKey, expected)
	}
}

func TestUserPathKey(t *testing.T) {
	expected := `HKCU:\Environment`
	if UserPathKey != expected {
		t.Errorf("UserPathKey = %s, want %s", UserPathKey, expected)
	}
}

func TestRunPowerShell(t *testing.T) {
	mock := getMockRunner(t)
	mock.SetResponse("test-command", "test-output")

	result, err := RunPowerShell("test-command")
	if err != nil {
		t.Errorf("RunPowerShell error: %v", err)
	}
	if result != "test-output" {
		t.Errorf("Expected 'test-output', got '%s'", result)
	}
}

func TestRunPowerShell_Error(t *testing.T) {
	withMockRunner(t, func(mock *MockShellRunner) {
		mock.SetError("error-command", errors.New("command failed"))
	}, func() {
		_, err := RunPowerShell("error-command")
		if err == nil {
			t.Error("Expected error")
		}
	})
}

func TestGetPathRaw_User(t *testing.T) {
	path, err := GetPathRaw("User")
	if err != nil {
		t.Logf("GetPathRaw(User) error: %v", err)
	}
	t.Logf("User PATH: %s", path)
}

func TestGetPathRaw_System(t *testing.T) {
	path, err := GetPathRaw("System")
	if err != nil {
		t.Logf("GetPathRaw(System) error: %v", err)
	}
	t.Logf("System PATH: %s", path)
}

func TestGetPathExpanded_User(t *testing.T) {
	path, err := GetPathExpanded("User")
	if err != nil {
		t.Logf("GetPathExpanded(User) error: %v", err)
	}
	t.Logf("User PATH expanded: %s", path)
}

func TestGetPathExpanded_System(t *testing.T) {
	path, err := GetPathExpanded("System")
	if err != nil {
		t.Logf("GetPathExpanded(System) error: %v", err)
	}
	t.Logf("System PATH expanded: %s", path)
}

func TestSetPath_User(t *testing.T) {
	mock := getMockRunner(t)
	beforeCalls := len(mock.Calls)

	err := SetPath(`C:\Test\Path`, "User")
	if err != nil {
		t.Logf("SetPath error: %v", err)
	}

	// Verify mock was called, not real PowerShell
	if len(mock.Calls) <= beforeCalls {
		t.Error("Mock should have been called for SetPath")
	}
}

func TestSetPath_System(t *testing.T) {
	mock := getMockRunner(t)
	beforeCalls := len(mock.Calls)

	err := SetPath(`C:\Test\Path`, "System")
	if err != nil {
		t.Logf("SetPath error: %v", err)
	}

	// Verify mock was called
	if len(mock.Calls) <= beforeCalls {
		t.Error("Mock should have been called for SetPath")
	}
}

func TestSetPath_WithQuotes(t *testing.T) {
	mock := getMockRunner(t)
	beforeCalls := len(mock.Calls)

	err := SetPath(`C:\Test's Path`, "User")
	if err != nil {
		t.Logf("SetPath with quotes error: %v", err)
	}

	// Verify mock was called
	if len(mock.Calls) <= beforeCalls {
		t.Error("Mock should have been called for SetPath")
	}
}

func TestIsAdmin(t *testing.T) {
	result := IsAdmin()
	t.Logf("IsAdmin: %v", result)
}

func TestIsAdmin_True(t *testing.T) {
	withMockRunner(t, func(mock *MockShellRunner) {
		mock.SetResponse("IsInRole", "True")
	}, func() {
		result := IsAdmin()
		if !result {
			t.Error("Expected IsAdmin to return true")
		}
	})
}

func TestIsAdmin_False(t *testing.T) {
	withMockRunner(t, func(mock *MockShellRunner) {
		mock.SetResponse("IsInRole", "False")
	}, func() {
		result := IsAdmin()
		if result {
			t.Error("Expected IsAdmin to return false")
		}
	})
}

func TestIsAdmin_Error(t *testing.T) {
	withMockRunner(t, func(mock *MockShellRunner) {
		mock.SetError("IsInRole", errors.New("access denied"))
	}, func() {
		result := IsAdmin()
		if result {
			t.Error("Expected IsAdmin to return false on error")
		}
	})
}

func TestBroadcastEnvChange(t *testing.T) {
	BroadcastEnvChange()
}

func TestGetHostname(t *testing.T) {
	hostname := GetHostname()
	if hostname == "" {
		t.Error("Hostname should not be empty")
	}
	t.Logf("Hostname: %s", hostname)
}

func TestGetHostname_Error(t *testing.T) {
	withMockRunner(t, func(mock *MockShellRunner) {
		mock.SetError("$env:COMPUTERNAME", errors.New("failed"))
	}, func() {
		hostname := GetHostname()
		if hostname != "UNKNOWN" {
			t.Errorf("Expected 'UNKNOWN' on error, got '%s'", hostname)
		}
	})
}

func TestGetAllEnvVars(t *testing.T) {
	vars := GetAllEnvVars()

	if vars == nil {
		t.Fatal("GetAllEnvVars returned nil")
	}
	if len(vars) == 0 {
		t.Error("GetAllEnvVars should return non-empty map")
	}
}

func TestGetAllEnvVars_HasCommonVars(t *testing.T) {
	vars := GetAllEnvVars()

	commonVars := []string{"PATH"}
	for _, v := range commonVars {
		found := false
		for key := range vars {
			if strings.EqualFold(key, v) {
				found = true
				break
			}
		}
		if !found {
			t.Logf("Warning: Common variable %s not found", v)
		}
	}
}

func TestGetAllEnvVars_Values(t *testing.T) {
	vars := GetAllEnvVars()

	for key, value := range vars {
		osValue := os.Getenv(key)
		if osValue != value {
			t.Logf("Value mismatch for %s", key)
		}
	}
}

func TestCopyToClipboard(t *testing.T) {
	err := CopyToClipboard("test content")
	if err != nil {
		t.Logf("CopyToClipboard error (may be expected): %v", err)
	}
}

func TestCopyToClipboard_Empty(t *testing.T) {
	err := CopyToClipboard("")
	if err != nil {
		t.Logf("CopyToClipboard empty error: %v", err)
	}
}

func TestGetRefreshCommand(t *testing.T) {
	cmd := GetRefreshCommand()

	if cmd == "" {
		t.Error("GetRefreshCommand should not return empty string")
	}
	if !strings.Contains(cmd, "GetEnvironmentVariable") {
		t.Error("Refresh command should contain GetEnvironmentVariable")
	}
	if !strings.Contains(cmd, "Machine") {
		t.Error("Refresh command should reference Machine scope")
	}
	if !strings.Contains(cmd, "User") {
		t.Error("Refresh command should reference User scope")
	}
}

func TestParsePath_Empty(t *testing.T) {
	result := ParsePath("")
	if len(result) != 0 {
		t.Errorf("Expected empty slice, got %d elements", len(result))
	}
}

func TestParsePath_Single(t *testing.T) {
	result := ParsePath(`C:\Windows`)
	if len(result) != 1 {
		t.Errorf("Expected 1 element, got %d", len(result))
	}
	if result[0] != `C:\Windows` {
		t.Errorf("Expected C:\\Windows, got %s", result[0])
	}
}

func TestParsePath_Multiple(t *testing.T) {
	result := ParsePath(`C:\Windows;C:\Windows\System32;C:\Program Files`)
	if len(result) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(result))
	}
}

func TestParsePath_EmptyEntries(t *testing.T) {
	result := ParsePath(`C:\Windows;;C:\System32;`)
	for _, entry := range result {
		if entry == "" {
			t.Error("Empty entry should be filtered")
		}
	}
}

func TestParsePath_Whitespace(t *testing.T) {
	result := ParsePath(`  C:\Windows  ; C:\System32  `)
	for _, entry := range result {
		if strings.HasPrefix(entry, " ") || strings.HasSuffix(entry, " ") {
			t.Errorf("Entry should be trimmed: '%s'", entry)
		}
	}
}

func TestJoinPath_Empty(t *testing.T) {
	result := JoinPath([]string{})
	if result != "" {
		t.Errorf("Expected empty string, got %s", result)
	}
}

func TestJoinPath_Single(t *testing.T) {
	result := JoinPath([]string{`C:\Windows`})
	if result != `C:\Windows` {
		t.Errorf("Expected C:\\Windows, got %s", result)
	}
}

func TestJoinPath_Multiple(t *testing.T) {
	entries := []string{`C:\Windows`, `C:\System32`, `C:\Program Files`}
	result := JoinPath(entries)
	expected := `C:\Windows;C:\System32;C:\Program Files`
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestJoinPath_Nil(t *testing.T) {
	result := JoinPath(nil)
	if result != "" {
		t.Errorf("Expected empty string for nil, got %s", result)
	}
}

func TestParsePath_JoinPath_RoundTrip(t *testing.T) {
	original := `C:\Windows;C:\System32;C:\Program Files`
	entries := ParsePath(original)
	rejoined := JoinPath(entries)

	if rejoined != original {
		t.Errorf("Round trip failed: %s -> %s", original, rejoined)
	}
}

func BenchmarkRunPowerShell(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RunPowerShell("$true")
	}
}

func BenchmarkGetPathRaw(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetPathRaw("User")
	}
}

func BenchmarkGetPathExpanded(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetPathExpanded("System")
	}
}

func BenchmarkParsePath(b *testing.B) {
	input := `C:\Windows;C:\Windows\System32;C:\Program Files;C:\Users\Test\bin`
	for i := 0; i < b.N; i++ {
		ParsePath(input)
	}
}

func BenchmarkJoinPath(b *testing.B) {
	entries := []string{`C:\Windows`, `C:\Windows\System32`, `C:\Program Files`}
	for i := 0; i < b.N; i++ {
		JoinPath(entries)
	}
}

func BenchmarkGetHostname(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetHostname()
	}
}

// Tests for expandShortName
func TestExpandShortName_NoTilde(t *testing.T) {
	result := expandShortName(`C:\Windows`)
	if result != `C:\Windows` {
		t.Errorf("Path without ~ should not change, got %s", result)
	}
}

func TestExpandShortName_WithVariable(t *testing.T) {
	result := expandShortName(`%USERPROFILE%\bin`)
	if result != `%USERPROFILE%\bin` {
		t.Errorf("Path with variable should not change, got %s", result)
	}
}

func TestExpandShortName_WithTilde(t *testing.T) {
	// This will attempt to expand - result depends on mock
	result := expandShortName(`C:\PROGRA~1`)
	t.Logf("expandShortName result: %s", result)
}

func TestGetPathExpanded_Expands8dot3(t *testing.T) {
	result, err := GetPathExpanded("System")
	if err != nil {
		t.Logf("GetPathExpanded error: %v", err)
	}
	t.Logf("GetPathExpanded result length: %d chars", len(result))
}
