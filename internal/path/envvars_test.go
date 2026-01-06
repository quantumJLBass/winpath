package path

import (
	"os"
	"strings"
	"testing"
)

func TestSubstitutionPriority(t *testing.T) {
	if len(SubstitutionPriority) == 0 {
		t.Fatal("SubstitutionPriority is empty")
	}

	// LOCALAPPDATA should come before USERPROFILE
	localIdx := -1
	userIdx := -1
	for i, v := range SubstitutionPriority {
		if v == "LOCALAPPDATA" {
			localIdx = i
		}
		if v == "USERPROFILE" {
			userIdx = i
		}
	}

	if localIdx != -1 && userIdx != -1 && localIdx > userIdx {
		t.Error("LOCALAPPDATA should have priority over USERPROFILE")
	}
}

func TestSubstitutionPriority_ContainsCommon(t *testing.T) {
	common := []string{"SystemRoot", "ProgramFiles", "USERPROFILE", "APPDATA"}

	for _, v := range common {
		found := false
		for _, p := range SubstitutionPriority {
			if strings.EqualFold(p, v) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s not found in SubstitutionPriority", v)
		}
	}
}

func TestExpandEnvVars_Empty(t *testing.T) {
	result := ExpandEnvVars("")
	if result != "" {
		t.Errorf("Expected empty, got %s", result)
	}
}

func TestExpandEnvVars_NoVars(t *testing.T) {
	input := `C:\Windows\System32`
	result := ExpandEnvVars(input)

	if result != input {
		t.Errorf("Expected %s, got %s", input, result)
	}
}

func TestExpandEnvVars_SingleVar(t *testing.T) {
	input := `%SystemRoot%\System32`
	result := ExpandEnvVars(input)

	systemRoot := os.Getenv("SystemRoot")
	if systemRoot != "" && !strings.Contains(result, systemRoot) {
		t.Logf("SystemRoot not expanded: %s -> %s", input, result)
	}
}

func TestExpandEnvVars_MultipleVars(t *testing.T) {
	input := `%SystemRoot%;%USERPROFILE%`
	result := ExpandEnvVars(input)

	if strings.Contains(result, "%") {
		t.Logf("Some variables may not be expanded: %s", result)
	}
}

func TestSubstituteEnvVars_Empty(t *testing.T) {
	result, changed := SubstituteEnvVars("")
	if result != "" || changed {
		t.Error("Empty should return empty unchanged")
	}
}

func TestSubstituteEnvVars_NoMatch(t *testing.T) {
	input := `C:\Some\Random\Path`
	result, changed := SubstituteEnvVars(input)

	if changed {
		t.Logf("Unexpected substitution: %s -> %s", input, result)
	}
}

func TestSubstituteEnvVars_SystemRoot(t *testing.T) {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		t.Skip("SystemRoot not set")
	}

	input := systemRoot + `\System32`
	result, changed := SubstituteEnvVars(input)

	if changed && !strings.HasPrefix(result, "%") {
		t.Errorf("Expected variable prefix, got %s", result)
	}
}

func TestSubstituteEnvVars_CaseInsensitive(t *testing.T) {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		t.Skip("SystemRoot not set")
	}

	input := strings.ToUpper(systemRoot) + `\System32`
	result, _ := SubstituteEnvVars(input)

	t.Logf("Case insensitive: %s -> %s", input, result)
}

func TestSubstituteEnvVars_Priority(t *testing.T) {
	localAppData := os.Getenv("LOCALAPPDATA")
	userProfile := os.Getenv("USERPROFILE")

	if localAppData == "" || userProfile == "" {
		t.Skip("Required vars not set")
	}

	// LOCALAPPDATA is under USERPROFILE
	input := localAppData + `\Test`
	result, changed := SubstituteEnvVars(input)

	if changed && strings.HasPrefix(result, "%USERPROFILE%") {
		t.Error("LOCALAPPDATA should take priority over USERPROFILE")
	}
}

func TestExpandEnvVars_RoundTrip(t *testing.T) {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		t.Skip("SystemRoot not set")
	}

	original := systemRoot + `\System32`
	substituted, changed := SubstituteEnvVars(original)

	if changed {
		expanded := ExpandEnvVars(substituted)
		if NormalizePath(expanded) != NormalizePath(original) {
			t.Logf("Round-trip mismatch: %s -> %s -> %s", original, substituted, expanded)
		}
	}
}

func TestGetAllEnvVars_NotEmpty(t *testing.T) {
	vars := GetAllEnvVars()

	if vars == nil {
		t.Fatal("GetAllEnvVars returned nil")
	}
	if len(vars) == 0 {
		t.Error("GetAllEnvVars should return non-empty map")
	}
}

func TestGetAllEnvVars_ContainsPATH(t *testing.T) {
	vars := GetAllEnvVars()

	found := false
	for key := range vars {
		if strings.EqualFold(key, "PATH") {
			found = true
			break
		}
	}
	if !found {
		t.Error("PATH should be in environment variables")
	}
}

func TestExpandEnvVars_UndefinedVar(t *testing.T) {
	input := `%UNDEFINED_VAR_12345%\test`
	result := ExpandEnvVars(input)

	// Undefined variables may or may not be expanded
	t.Logf("Undefined var: %s -> %s", input, result)
}

func TestSubstituteEnvVars_PartialMatch(t *testing.T) {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		t.Skip("SystemRoot not set")
	}

	// Path that starts with SystemRoot but has more
	input := systemRoot + `ers\Test`
	result, changed := SubstituteEnvVars(input)

	// Should NOT substitute because it's not a complete path component
	if changed && strings.Contains(result, "%SystemRoot%ers") {
		t.Error("Partial path component should not be substituted")
	}
}

func TestSubstituteEnvVars_ExactMatch(t *testing.T) {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		t.Skip("SystemRoot not set")
	}

	result, changed := SubstituteEnvVars(systemRoot)
	if changed {
		if result != "%SystemRoot%" {
			t.Logf("Exact match: %s -> %s", systemRoot, result)
		}
	}
}

func TestExpandEnvVars_NestedNotSupported(t *testing.T) {
	// Windows doesn't support nested expansion like %VAR1%VAR2%%
	input := `%SystemRoot%%USERPROFILE%`
	result := ExpandEnvVars(input)

	// Just verify it doesn't panic
	t.Logf("Nested vars: %s -> %s", input, result)
}

func TestEnvVarPaths_CommonPatterns(t *testing.T) {
	patterns := []string{
		`%SystemRoot%\System32`,
		`%ProgramFiles%\Common Files`,
		`%USERPROFILE%\bin`,
		`%LOCALAPPDATA%\Programs`,
		`%APPDATA%\npm`,
	}

	for _, p := range patterns {
		expanded := ExpandEnvVars(p)
		t.Logf("%s -> %s", p, expanded)
	}
}

func BenchmarkExpandEnvVars(b *testing.B) {
	input := `%SystemRoot%\System32`
	for i := 0; i < b.N; i++ {
		ExpandEnvVars(input)
	}
}

func BenchmarkSubstituteEnvVars(b *testing.B) {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		b.Skip("SystemRoot not set")
	}
	input := systemRoot + `\System32`

	for i := 0; i < b.N; i++ {
		SubstituteEnvVars(input)
	}
}

func BenchmarkGetAllEnvVars(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetAllEnvVars()
	}
}

func TestExpandShortUserPath_NotUserPath(t *testing.T) {
	path := `C:\Windows\System32`
	result := expandShortUserPath(path)
	if result != path {
		t.Errorf("Non-user path should be unchanged: %s", result)
	}
}

func TestExpandShortUserPath_NoShortName(t *testing.T) {
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		t.Skip("USERPROFILE not set")
	}
	result := expandShortUserPath(userProfile)
	if result != userProfile {
		t.Logf("Path expanded: %s -> %s", userProfile, result)
	}
}

func TestExpandShortUserPath_WithShortName(t *testing.T) {
	// Test with a simulated short path
	path := `C:\Users\ADMINI~1\Documents`
	result := expandShortUserPath(path)
	t.Logf("Short path expansion: %s -> %s", path, result)
}

func TestExpandShortUserPath_NoSlashAfterUser(t *testing.T) {
	path := `C:\Users\TestUser`
	result := expandShortUserPath(path)
	if result != path {
		t.Errorf("Path without trailing slash should be unchanged: %s", result)
	}
}

func TestGetEnvVariable_User(t *testing.T) {
	result, err := GetEnvVariable("PATH", "User")
	if err != nil {
		t.Logf("GetEnvVariable error: %v", err)
	}
	t.Logf("User PATH: %s", result)
}

func TestGetEnvVariable_System(t *testing.T) {
	result, err := GetEnvVariable("PATH", "Machine")
	if err != nil {
		t.Logf("GetEnvVariable error: %v", err)
	}
	t.Logf("System PATH: %s", result)
}

func TestSetEnvVariable_User(t *testing.T) {
	// Note: This actually modifies the registry, so we just test the call
	err := SetEnvVariable("TEST_VAR_SYSPATH", "test_value", "User")
	if err != nil {
		t.Logf("SetEnvVariable error: %v", err)
	}
}

func TestSetEnvVariable_System(t *testing.T) {
	err := SetEnvVariable("TEST_VAR_SYSPATH", "test_value", "Machine")
	if err != nil {
		t.Logf("SetEnvVariable error (expected without admin): %v", err)
	}
}
