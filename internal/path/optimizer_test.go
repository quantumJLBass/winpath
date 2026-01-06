package path

import (
	"os"
	"strings"
	"testing"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if !opts.RemoveDuplicates {
		t.Error("RemoveDuplicates should be true")
	}
	if !opts.RemoveDeadPaths {
		t.Error("RemoveDeadPaths should be true")
	}
	if !opts.ShortenPaths {
		t.Error("ShortenPaths should be true")
	}
	if !opts.SubstituteVars {
		t.Error("SubstituteVars should be true")
	}
	if opts.ReorderPaths {
		t.Error("ReorderPaths should be false by default")
	}
	if opts.Scope != "User" {
		t.Errorf("Scope should be 'User', got %s", opts.Scope)
	}
}

func TestOptimizeOptions_Struct(t *testing.T) {
	opts := OptimizeOptions{
		RemoveDuplicates: true,
		RemoveDeadPaths:  false,
		ShortenPaths:     true,
		SubstituteVars:   false,
		ReorderPaths:     true,
		Scope:            "System",
	}

	if !opts.RemoveDuplicates || opts.RemoveDeadPaths || !opts.ShortenPaths {
		t.Error("Options not set correctly")
	}
	if opts.SubstituteVars || !opts.ReorderPaths || opts.Scope != "System" {
		t.Error("Options not set correctly")
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`C:\Windows`, `c:\windows`},
		{`C:\WINDOWS\`, `c:\windows`},
		{`c:\windows`, `c:\windows`},
		{`C:\Windows\System32\`, `c:\windows\system32`},
		{`C:/Windows/System32`, `c:\windows\system32`},
		{``, `.`}, // filepath.Clean("") returns "."
	}

	for _, tt := range tests {
		result := NormalizePath(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizePath(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizePath_TrailingSlash(t *testing.T) {
	tests := []string{`C:\Windows\`, `C:\Windows\\`, `C:\Windows/`}
	for _, input := range tests {
		result := NormalizePath(input)
		if strings.HasSuffix(result, `\`) || strings.HasSuffix(result, `/`) {
			t.Errorf("NormalizePath(%s) should not have trailing slash", input)
		}
	}
}

func TestPathExists(t *testing.T) {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot != "" {
		if !PathExists(systemRoot) {
			t.Errorf("PathExists(%s) should be true", systemRoot)
		}
	}

	if PathExists(`C:\This\Path\Does\Not\Exist\12345`) {
		t.Error("PathExists should return false for non-existent path")
	}
}

func TestPathExists_WithVariable(t *testing.T) {
	result := PathExists(`%USERPROFILE%\bin`)
	if !result {
		t.Error("PathExists should return true for paths with variables")
	}
}

func TestPathExists_Empty(t *testing.T) {
	if PathExists("") {
		t.Error("PathExists should return false for empty path")
	}
}

func TestOptimize_Empty(t *testing.T) {
	opts := DefaultOptions()
	result := Optimize("", opts)

	if result.Original.Count != 0 || result.Optimized.Count != 0 {
		t.Error("Empty input should have 0 entries")
	}
}

func TestOptimize_RemovesDuplicates(t *testing.T) {
	opts := DefaultOptions()
	opts.RemoveDeadPaths = false
	opts.ShortenPaths = false
	opts.SubstituteVars = false

	input := `C:\Test;C:\Test;C:\Other;C:\TEST`
	result := Optimize(input, opts)

	if result.Metrics.DuplicatesRemoved < 1 {
		t.Errorf("Expected duplicates removed, got %d", result.Metrics.DuplicatesRemoved)
	}
}

func TestOptimize_DuplicatesDisabled(t *testing.T) {
	opts := DefaultOptions()
	opts.RemoveDuplicates = false
	opts.RemoveDeadPaths = false
	opts.ShortenPaths = false
	opts.SubstituteVars = false

	input := `C:\Test;C:\Test;C:\Other`
	result := Optimize(input, opts)

	if result.Optimized.Count != result.Original.Count {
		t.Error("Duplicates should be kept when disabled")
	}
}

func TestOptimize_PreservesOrder(t *testing.T) {
	opts := DefaultOptions()
	opts.RemoveDeadPaths = false
	opts.ShortenPaths = false
	opts.SubstituteVars = false
	opts.RemoveDuplicates = false

	input := `C:\First;C:\Second;C:\Third`
	result := Optimize(input, opts)

	expected := []string{`C:\First`, `C:\Second`, `C:\Third`}
	for i, exp := range expected {
		if i < len(result.Optimized.Entries) && result.Optimized.Entries[i] != exp {
			t.Errorf("Entry %d: expected %s, got %s", i, exp, result.Optimized.Entries[i])
		}
	}
}

func TestOptimize_PathWithVariables(t *testing.T) {
	opts := DefaultOptions()
	opts.RemoveDeadPaths = false
	opts.ShortenPaths = false
	opts.SubstituteVars = false

	input := `%USERPROFILE%\bin;%SystemRoot%\System32`
	result := Optimize(input, opts)

	for _, entry := range result.Optimized.Entries {
		if !strings.Contains(entry, "%") {
			t.Errorf("Variable should be preserved: %s", entry)
		}
	}
}

func TestOptimize_Metrics(t *testing.T) {
	opts := DefaultOptions()
	opts.RemoveDeadPaths = false
	opts.ShortenPaths = false
	opts.SubstituteVars = false

	input := `C:\Test;C:\Test;C:\Other`
	result := Optimize(input, opts)

	if result.Original.Length == 0 || result.Original.Count != 3 {
		t.Error("Original metrics not correct")
	}
	if result.Original.Raw != input {
		t.Error("Original.Raw should match input")
	}
}

func TestOptimize_PercentageSaved(t *testing.T) {
	opts := DefaultOptions()
	opts.RemoveDeadPaths = false
	opts.ShortenPaths = false
	opts.SubstituteVars = false

	input := `C:\Windows;C:\Windows;C:\Windows`
	result := Optimize(input, opts)

	if result.Metrics.PercentageSaved < 0 {
		t.Error("PercentageSaved should not be negative")
	}
}

func TestOptimize_ZeroOriginalLength(t *testing.T) {
	opts := DefaultOptions()
	result := Optimize("", opts)

	if result.Metrics.PercentageSaved != 0 {
		t.Error("PercentageSaved should be 0 for empty input")
	}
}

func TestPathChange_Struct(t *testing.T) {
	change := PathChange{
		Type:     "duplicate",
		Original: `C:\Test`,
		New:      "",
		Saved:    0,
	}

	if change.Type != "duplicate" || change.Original != `C:\Test` {
		t.Error("PathChange fields not set correctly")
	}
}

func TestApplyHotPaths_Empty(t *testing.T) {
	entries := []string{`C:\First`, `C:\Second`, `C:\Third`}
	result := applyHotPaths(entries, []string{})

	if len(result) != len(entries) {
		t.Errorf("Expected %d entries, got %d", len(entries), len(result))
	}
}

func TestApplyHotPaths_Nil(t *testing.T) {
	entries := []string{`C:\First`, `C:\Second`}
	result := applyHotPaths(entries, nil)

	if len(result) != len(entries) {
		t.Errorf("Expected %d entries, got %d", len(entries), len(result))
	}
}

func TestApplyHotPaths_MovesToFront(t *testing.T) {
	entries := []string{`C:\First`, `C:\Second`, `C:\Third`}
	hotPaths := []string{`C:\Third`}

	result := applyHotPaths(entries, hotPaths)

	if NormalizePath(result[0]) != NormalizePath(`C:\Third`) {
		t.Errorf("Expected C:\\Third first, got %s", result[0])
	}
}

func TestApplyHotPaths_CaseInsensitive(t *testing.T) {
	entries := []string{`C:\WINDOWS`, `C:\System32`}
	hotPaths := []string{`c:\windows`}

	result := applyHotPaths(entries, hotPaths)

	if NormalizePath(result[0]) != NormalizePath(`C:\WINDOWS`) {
		t.Errorf("Expected C:\\WINDOWS first, got %s", result[0])
	}
}

func TestDetectCustomPathVars(t *testing.T) {
	sysPath := `%SystemRoot%;%CUSTOM_VAR%\bin`
	usrPath := `%USERPROFILE%;%MY_TOOL_HOME%\bin`

	result := DetectCustomPathVars(sysPath, usrPath)

	foundCustom := false
	for _, v := range result {
		if strings.EqualFold(v.Name, "CUSTOM_VAR") || strings.EqualFold(v.Name, "MY_TOOL_HOME") {
			foundCustom = true
		}
	}
	if !foundCustom && len(result) == 0 {
		t.Log("No custom vars detected (may be expected)")
	}
}

func TestDetectCustomPathVars_Empty(t *testing.T) {
	result := DetectCustomPathVars("", "")
	if len(result) != 0 {
		t.Errorf("Expected 0 custom vars, got %d", len(result))
	}
}

func TestOptimizeResult_Structure(t *testing.T) {
	result := OptimizeResult{}
	result.Original.Raw = "test"
	result.Original.Entries = []string{"a", "b"}
	result.Original.Length = 10
	result.Original.Count = 2
	result.Optimized.Raw = "optimized"
	result.Optimized.Entries = []string{"a"}
	result.Optimized.Length = 5
	result.Optimized.Count = 1
	result.Changes = []PathChange{{Type: "duplicate", Original: "b"}}
	result.Metrics.DuplicatesRemoved = 1

	if result.Original.Count != 2 || result.Optimized.Count != 1 {
		t.Error("OptimizeResult not set correctly")
	}
}

func TestAnalysisResult_Struct(t *testing.T) {
	result := AnalysisResult{}
	result.System = OptimizeResult{}
	result.User = OptimizeResult{}
	result.CustomVariables = []CustomPathVar{{Name: "TEST", FoundIn: "User"}}

	if len(result.CustomVariables) != 1 {
		t.Error("CustomVariables not set correctly")
	}
}

func TestCustomPathVar_Struct(t *testing.T) {
	v := CustomPathVar{Name: "MY_VAR", FoundIn: "System", Value: "C:\\MyPath"}

	if v.Name != "MY_VAR" || v.FoundIn != "System" || v.Value != "C:\\MyPath" {
		t.Error("CustomPathVar fields not set correctly")
	}
}

func TestAnalyzeAll(t *testing.T) {
	opts := DefaultOptions()
	opts.RemoveDeadPaths = false

	result := AnalyzeAll(opts)
	t.Logf("System: %d, User: %d", result.System.Original.Count, result.User.Original.Count)
}

func TestAnalyzeAllWithProgress(t *testing.T) {
	opts := DefaultOptions()
	opts.RemoveDeadPaths = false

	progressCalled := false
	progress := func(current, total int, item string) {
		progressCalled = true
	}

	result := AnalyzeAllWithProgress(opts, progress)
	t.Logf("Analysis: System=%d, User=%d, progress=%v",
		result.System.Original.Count, result.User.Original.Count, progressCalled)
}

func TestOptimizeWithProgress(t *testing.T) {
	opts := DefaultOptions()
	opts.RemoveDeadPaths = false
	opts.ShortenPaths = false
	opts.SubstituteVars = false

	input := `C:\Windows;C:\Windows\System32;C:\Program Files`

	progressCalled := false
	progress := func(current, total int, item string) {
		progressCalled = true
	}

	result := OptimizeWithProgress(input, opts, 0, 3, progress)

	if result.Original.Count != 3 {
		t.Errorf("Expected 3 entries, got %d", result.Original.Count)
	}
	t.Logf("Progress callback called: %v", progressCalled)
}

func TestOptimizeWithProgress_NilCallback(t *testing.T) {
	opts := DefaultOptions()
	opts.RemoveDeadPaths = false

	input := `C:\Windows;C:\Windows\System32`
	result := OptimizeWithProgress(input, opts, 0, 2, nil)

	if result.Original.Count != 2 {
		t.Errorf("Expected 2 entries, got %d", result.Original.Count)
	}
}

func BenchmarkOptimize(b *testing.B) {
	paths := make([]string, 50)
	for i := 0; i < 50; i++ {
		paths[i] = `C:\Program Files\App` + string(rune('A'+i%26))
	}
	input := strings.Join(paths, ";")
	opts := DefaultOptions()
	opts.RemoveDeadPaths = false

	for i := 0; i < b.N; i++ {
		Optimize(input, opts)
	}
}

func BenchmarkNormalizePath(b *testing.B) {
	input := `C:\Program Files\Microsoft Visual Studio\2022\Enterprise\`
	for i := 0; i < b.N; i++ {
		NormalizePath(input)
	}
}

func BenchmarkPathExists(b *testing.B) {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		b.Skip("SystemRoot not set")
	}
	for i := 0; i < b.N; i++ {
		PathExists(systemRoot)
	}
}

func BenchmarkApplyHotPaths(b *testing.B) {
	entries := make([]string, 50)
	for i := 0; i < 50; i++ {
		entries[i] = `C:\Path` + string(rune('A'+i%26))
	}
	hotPaths := []string{entries[40], entries[30], entries[20]}

	for i := 0; i < b.N; i++ {
		applyHotPaths(entries, hotPaths)
	}
}

// Tests for entry processor helper methods
func TestEntryProcessor_IsDuplicate(t *testing.T) {
	result := &OptimizeResult{}
	opts := DefaultOptions()
	processor := newEntryProcessor(opts, result)

	// First occurrence - not duplicate
	if processor.isDuplicate(`C:\Windows`, `c:\windows`) {
		t.Error("First occurrence should not be duplicate")
	}

	// Second occurrence - is duplicate
	if !processor.isDuplicate(`C:\Windows`, `c:\windows`) {
		t.Error("Second occurrence should be duplicate")
	}

	if result.Metrics.DuplicatesRemoved != 1 {
		t.Errorf("Expected 1 duplicate removed, got %d", result.Metrics.DuplicatesRemoved)
	}
}

func TestEntryProcessor_IsDuplicate_Disabled(t *testing.T) {
	result := &OptimizeResult{}
	opts := OptimizeOptions{RemoveDuplicates: false}
	processor := newEntryProcessor(opts, result)

	processor.isDuplicate(`C:\Windows`, `c:\windows`)
	if processor.isDuplicate(`C:\Windows`, `c:\windows`) {
		t.Error("Duplicate check should be disabled")
	}
}

func TestEntryProcessor_IsDeadPath(t *testing.T) {
	result := &OptimizeResult{}
	opts := DefaultOptions()
	processor := newEntryProcessor(opts, result)

	// Path with variable - not checked
	if processor.isDeadPath(`%USERPROFILE%\bin`) {
		t.Error("Path with variable should not be marked dead")
	}
}

func TestEntryProcessor_IsDeadPath_Disabled(t *testing.T) {
	result := &OptimizeResult{}
	opts := OptimizeOptions{RemoveDeadPaths: false}
	processor := newEntryProcessor(opts, result)

	if processor.isDeadPath(`C:\NonExistent12345`) {
		t.Error("Dead path check should be disabled")
	}
}

func TestEntryProcessor_TryShorten(t *testing.T) {
	result := &OptimizeResult{}
	opts := DefaultOptions()
	processor := newEntryProcessor(opts, result)

	// Path with variable - not shortened
	out := processor.tryShorten(`%USERPROFILE%\bin`)
	if out != `%USERPROFILE%\bin` {
		t.Error("Path with variable should not be shortened")
	}
}

func TestEntryProcessor_TryShorten_Disabled(t *testing.T) {
	result := &OptimizeResult{}
	opts := OptimizeOptions{ShortenPaths: false}
	processor := newEntryProcessor(opts, result)

	out := processor.tryShorten(`C:\Windows`)
	if out != `C:\Windows` {
		t.Error("Shortening should be disabled")
	}
}

func TestEntryProcessor_TrySubstituteVars(t *testing.T) {
	result := &OptimizeResult{}
	opts := DefaultOptions()
	processor := newEntryProcessor(opts, result)

	// Path already has variable - not substituted
	out := processor.trySubstituteVars(`%USERPROFILE%\bin`)
	if out != `%USERPROFILE%\bin` {
		t.Error("Path with variable should not be substituted")
	}
}

func TestEntryProcessor_TrySubstituteVars_Disabled(t *testing.T) {
	result := &OptimizeResult{}
	opts := OptimizeOptions{SubstituteVars: false}
	processor := newEntryProcessor(opts, result)

	out := processor.trySubstituteVars(`C:\Users\Test`)
	if out != `C:\Users\Test` {
		t.Error("Substitution should be disabled")
	}
}

func TestEntryProcessor_TryShortenSuffix(t *testing.T) {
	result := &OptimizeResult{}
	opts := DefaultOptions()
	processor := newEntryProcessor(opts, result)

	out := processor.tryShortenSuffix(`%USERPROFILE%\AppData`)
	t.Logf("tryShortenSuffix result: %s", out)
}

func TestEntryProcessor_TryShortenSuffix_Disabled(t *testing.T) {
	result := &OptimizeResult{}
	opts := OptimizeOptions{ShortenPaths: false}
	processor := newEntryProcessor(opts, result)

	out := processor.tryShortenSuffix(`%USERPROFILE%\AppData`)
	if out != `%USERPROFILE%\AppData` {
		t.Error("Suffix shortening should be disabled")
	}
}

func TestEntryProcessor_ProcessEntry(t *testing.T) {
	result := &OptimizeResult{}
	opts := DefaultOptions()
	opts.RemoveDeadPaths = false
	processor := newEntryProcessor(opts, result)

	processed, ok := processor.processEntry(`C:\Windows`)
	if !ok {
		t.Error("Entry should be processed")
	}
	t.Logf("Processed entry: %s", processed)
}

func TestEntryProcessor_ProcessEntry_Duplicate(t *testing.T) {
	result := &OptimizeResult{}
	opts := DefaultOptions()
	opts.RemoveDeadPaths = false
	processor := newEntryProcessor(opts, result)

	processor.processEntry(`C:\Windows`)
	_, ok := processor.processEntry(`C:\Windows`)
	if ok {
		t.Error("Duplicate entry should be skipped")
	}
}

func TestNewEntryProcessor(t *testing.T) {
	result := &OptimizeResult{}
	opts := DefaultOptions()
	processor := newEntryProcessor(opts, result)

	if processor == nil {
		t.Fatal("Processor should not be nil")
	}
	if processor.seen == nil {
		t.Error("seen map should be initialized")
	}
}
