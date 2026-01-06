package path

import (
	"os"
	"strings"
	"testing"
)

func TestIntegration_FullOptimizationWorkflow(t *testing.T) {
	// Get PATH (mocked)
	userPath, err := GetPathRaw("User")
	if err != nil {
		t.Logf("GetPathRaw error: %v", err)
	}

	opts := DefaultOptions()
	opts.Scope = "User"
	opts.RemoveDeadPaths = false
	result := Optimize(userPath, opts)

	if result.Original.Count < 0 || result.Optimized.Count < 0 {
		t.Error("Counts should not be negative")
	}

	for _, entry := range result.Optimized.Entries {
		if entry == "" {
			t.Error("Optimized entries should not contain empty strings")
		}
	}

	reparsed := ParsePath(result.Optimized.Raw)
	if len(reparsed) != result.Optimized.Count {
		t.Errorf("Reparsed count mismatch: %d vs %d", len(reparsed), result.Optimized.Count)
	}
}

func TestIntegration_VariableSubstitution(t *testing.T) {
	userProfile := os.Getenv("USERPROFILE")
	localAppData := os.Getenv("LOCALAPPDATA")

	if userProfile == "" || localAppData == "" {
		t.Skip("Required environment variables not set")
	}

	testPath := localAppData + `\Programs\Test`
	result, changed := SubstituteEnvVars(testPath)

	if changed && strings.HasPrefix(result, "%USERPROFILE%") {
		t.Error("LOCALAPPDATA should take precedence over USERPROFILE")
	}

	if changed {
		expanded := ExpandEnvVars(result)
		if NormalizePath(expanded) != NormalizePath(testPath) {
			t.Logf("Round-trip: %s -> %s -> %s", testPath, result, expanded)
		}
	}
}

func TestIntegration_PathExtWorkflow(t *testing.T) {
	analysis := AnalyzePathExt()

	if len(analysis.Current) == 0 {
		t.Log("Current PATHEXT is empty")
	}

	opt := OptimizePathExt(true)

	for _, ext := range opt.Optimized {
		if !strings.HasPrefix(ext, ".") {
			t.Errorf("Extension should start with dot: %s", ext)
		}
	}

	if opt.Changed && len(opt.Optimized) > 1 {
		if !strings.Contains(opt.OptimizedString, ";") {
			t.Error("Optimized string should contain semicolons")
		}
	}
}

func TestIntegration_JunctionSuggestions(t *testing.T) {
	suggestions := SuggestJunctionCandidates()

	for _, s := range suggestions {
		if strings.Contains(s.OriginalPath, "%") {
			t.Errorf("Suggestions should not contain variables: %s", s.OriginalPath)
		}
		if s.SuggestedName == "" {
			t.Error("Suggested name should not be empty")
		}
	}
}

func TestIntegration_BackupConfig(t *testing.T) {
	cfg := Config{
		JunctionFolder: `C:\test\junctions`,
		MaxBackups:     5,
		AutoBackup:     true,
		HotPaths:       []string{`C:\Windows`, `C:\Program Files`},
	}

	err := SaveConfig(cfg)
	if err != nil {
		t.Logf("SaveConfig error: %v", err)
		return
	}

	loaded := LoadConfig()

	if loaded.MaxBackups != cfg.MaxBackups {
		t.Logf("MaxBackups mismatch: %d vs %d", loaded.MaxBackups, cfg.MaxBackups)
	}
}

func TestIntegration_FullAnalysis(t *testing.T) {
	opts := DefaultOptions()
	opts.RemoveDeadPaths = false
	result := AnalyzeAll(opts)

	t.Logf("System: %d entries, User: %d entries",
		result.System.Original.Count, result.User.Original.Count)
}

func TestIntegration_HotPathsApplication(t *testing.T) {
	entries := []string{`C:\First`, `C:\Second`, `C:\Third`, `C:\Fourth`}
	hotPaths := []string{`C:\Third`, `C:\First`}

	result := applyHotPaths(entries, hotPaths)

	if len(result) != len(entries) {
		t.Errorf("Entry count changed: %d -> %d", len(entries), len(result))
	}

	// Third should come before First, both before others
	thirdIdx := -1
	firstIdx := -1
	for i, e := range result {
		if NormalizePath(e) == NormalizePath(`C:\Third`) {
			thirdIdx = i
		}
		if NormalizePath(e) == NormalizePath(`C:\First`) {
			firstIdx = i
		}
	}

	if thirdIdx > firstIdx {
		t.Error("Third should come before First in hot paths order")
	}
}

func TestIntegration_NormalizationConsistency(t *testing.T) {
	paths := []string{
		`C:\Windows`,
		`C:\WINDOWS`,
		`c:\windows`,
		`C:\Windows\`,
		`C:/Windows`,
	}

	normalized := NormalizePath(paths[0])
	for _, p := range paths[1:] {
		if NormalizePath(p) != normalized {
			t.Errorf("Normalization inconsistent: %s != %s", p, paths[0])
		}
	}
}

func TestIntegration_ParseJoinRoundTrip(t *testing.T) {
	original := `C:\Windows;C:\Windows\System32;C:\Program Files`
	entries := ParsePath(original)
	rejoined := JoinPath(entries)

	if rejoined != original {
		t.Errorf("Round trip failed: %s -> %s", original, rejoined)
	}
}

func TestIntegration_OptimizePreservesValid(t *testing.T) {
	opts := DefaultOptions()
	opts.RemoveDuplicates = false
	opts.RemoveDeadPaths = false
	opts.ShortenPaths = false
	opts.SubstituteVars = false

	input := `C:\ValidPath1;C:\ValidPath2;C:\ValidPath3`
	result := Optimize(input, opts)

	if result.Optimized.Count != result.Original.Count {
		t.Errorf("Valid entries should be preserved: %d -> %d",
			result.Original.Count, result.Optimized.Count)
	}
}

func TestIntegration_EnvironmentVariableChain(t *testing.T) {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		t.Skip("SystemRoot not set")
	}

	original := systemRoot + `\System32`

	substituted, changed := SubstituteEnvVars(original)

	if changed {
		expanded := ExpandEnvVars(substituted)
		if NormalizePath(expanded) != NormalizePath(original) {
			t.Logf("Chain: %s -> %s -> %s", original, substituted, expanded)
		}
	}
}

func TestIntegration_ExtensionDatabaseConsistency(t *testing.T) {
	for ext, info := range ExtensionDatabase {
		if info.Ext != ext {
			t.Errorf("Extension mismatch in database: key=%s, value.Ext=%s", ext, info.Ext)
		}
		if info.Description == "" {
			t.Errorf("Missing description for %s", ext)
		}
		if info.Priority <= 0 {
			t.Errorf("Invalid priority for %s: %d", ext, info.Priority)
		}
	}
}

func TestIntegration_OptimalOrderCoverage(t *testing.T) {
	for _, ext := range OptimalOrder {
		if _, ok := ExtensionDatabase[ext]; !ok {
			t.Errorf("OptimalOrder contains %s not in ExtensionDatabase", ext)
		}
	}
}

func TestIntegration_SubstitutionPriorityOrder(t *testing.T) {
	seen := make(map[string]bool)
	for _, v := range SubstitutionPriority {
		if seen[v] {
			t.Errorf("Duplicate in SubstitutionPriority: %s", v)
		}
		seen[v] = true
	}
}

func TestIntegration_PathExistsConsistency(t *testing.T) {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		t.Skip("SystemRoot not set")
	}

	result1 := PathExists(systemRoot)
	result2 := PathExists(strings.ToLower(systemRoot))
	result3 := PathExists(strings.ToUpper(systemRoot))

	if result1 != result2 || result2 != result3 {
		t.Error("PathExists should be case-insensitive")
	}
}

func BenchmarkIntegration_FullOptimization(b *testing.B) {
	paths := make([]string, 20)
	for i := 0; i < 20; i++ {
		paths[i] = `C:\Program Files\App` + string(rune('A'+i))
	}
	input := strings.Join(paths, ";")

	opts := DefaultOptions()
	opts.RemoveDeadPaths = false

	for i := 0; i < b.N; i++ {
		Optimize(input, opts)
	}
}

func BenchmarkIntegration_AnalyzeAll(b *testing.B) {
	opts := DefaultOptions()
	opts.RemoveDeadPaths = false

	for i := 0; i < b.N; i++ {
		AnalyzeAll(opts)
	}
}

func BenchmarkIntegration_NormalizePath(b *testing.B) {
	input := `C:\Program Files\Microsoft Visual Studio\2022\Enterprise\`

	for i := 0; i < b.N; i++ {
		NormalizePath(input)
	}
}

func BenchmarkIntegration_ParseJoinPath(b *testing.B) {
	input := `C:\Windows;C:\Windows\System32;C:\Program Files;C:\Users\Test`

	for i := 0; i < b.N; i++ {
		entries := ParsePath(input)
		JoinPath(entries)
	}
}
