package path

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestDefaultPathExt(t *testing.T) {
	if DefaultPathExt == "" {
		t.Error("DefaultPathExt should not be empty")
	}
	if !strings.Contains(DefaultPathExt, ".EXE") {
		t.Error("DefaultPathExt should contain .EXE")
	}
}

func TestOptimalOrder(t *testing.T) {
	if len(OptimalOrder) == 0 {
		t.Fatal("OptimalOrder is empty")
	}
	if OptimalOrder[0] != ".EXE" {
		t.Errorf("Expected .EXE first, got %s", OptimalOrder[0])
	}
}

func TestOptimalOrder_ContainsAllCommon(t *testing.T) {
	common := []string{".EXE", ".CMD", ".BAT", ".COM", ".PS1", ".MSC"}
	for _, ext := range common {
		found := false
		for _, o := range OptimalOrder {
			if o == ext {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s not found in OptimalOrder", ext)
		}
	}
}

func TestExtensionDatabase(t *testing.T) {
	required := []string{".EXE", ".CMD", ".BAT", ".COM", ".PY", ".PS1"}

	for _, ext := range required {
		info, ok := ExtensionDatabase[ext]
		if !ok {
			t.Errorf("ExtensionDatabase missing entry for %s", ext)
			continue
		}
		if info.Description == "" || info.Ext != ext {
			t.Errorf("ExtensionDatabase[%s] has invalid data", ext)
		}
	}
}

func TestExtensionDatabase_Legacy(t *testing.T) {
	if info, ok := ExtensionDatabase[".COM"]; ok && !info.IsLegacy {
		t.Error(".COM should be marked as legacy")
	}
	if info, ok := ExtensionDatabase[".BAT"]; ok && !info.IsLegacy {
		t.Error(".BAT should be marked as legacy")
	}
	if info, ok := ExtensionDatabase[".EXE"]; ok && info.IsLegacy {
		t.Error(".EXE should NOT be marked as legacy")
	}
}

func TestExtensionDatabase_Priority(t *testing.T) {
	exeInfo := ExtensionDatabase[".EXE"]
	if exeInfo.Priority != 1 {
		t.Errorf(".EXE priority should be 1, got %d", exeInfo.Priority)
	}
}

func TestGetExtensionInfo_Known(t *testing.T) {
	info := GetExtensionInfo(".EXE")

	if info.Description == "" || info.Ext != ".EXE" || info.Priority != 1 {
		t.Error("GetExtensionInfo(.EXE) returned invalid data")
	}
}

func TestGetExtensionInfo_Unknown(t *testing.T) {
	info := GetExtensionInfo(".UNKNOWN")

	if info.Description != "Unknown extension" || info.Priority != 99 {
		t.Error("Unknown extension should have default values")
	}
}

func TestGetExtensionInfo_CaseInsensitive(t *testing.T) {
	testCases := []string{".exe", ".EXE", ".Exe"}
	for _, ext := range testCases {
		info := GetExtensionInfo(ext)
		if info.Ext != ".EXE" {
			t.Errorf("GetExtensionInfo(%s).Ext = %s, want .EXE", ext, info.Ext)
		}
	}
}

func TestExtensionInfo_Struct(t *testing.T) {
	info := ExtensionInfo{
		Ext:          ".TEST",
		Description:  "Test extension",
		Priority:     50,
		IsLegacy:     true,
		IsRarelyUsed: true,
	}

	if info.Ext != ".TEST" || info.Priority != 50 || !info.IsLegacy {
		t.Error("ExtensionInfo struct not set correctly")
	}
}

func TestRemovableExtensions(t *testing.T) {
	expected := []string{".VBE", ".JSE", ".WSF", ".WSH"}
	if len(RemovableExtensions) != len(expected) {
		t.Errorf("Expected %d removable extensions, got %d", len(expected), len(RemovableExtensions))
	}
}

func TestParsePathExt(t *testing.T) {
	input := ".EXE;.CMD;.BAT"
	result := ParsePathExt(input)

	if len(result) != 3 {
		t.Errorf("Expected 3 extensions, got %d", len(result))
	}
}

func TestParsePathExt_Lowercase(t *testing.T) {
	input := ".exe;.cmd;.bat"
	result := ParsePathExt(input)

	for _, ext := range result {
		if ext != strings.ToUpper(ext) {
			t.Errorf("Extension should be uppercase: %s", ext)
		}
	}
}

func TestParsePathExt_EmptyEntries(t *testing.T) {
	input := ".EXE;;.CMD;;;.BAT"
	result := ParsePathExt(input)

	for _, ext := range result {
		if ext == "" {
			t.Error("Empty extension should be filtered")
		}
	}
}

func TestPathExtAnalysis_Struct(t *testing.T) {
	analysis := PathExtAnalysis{
		Current:   []string{".EXE", ".CMD"},
		IsOptimal: true,
	}
	analysis.CurrentWithInfo = []ExtensionInfo{{Ext: ".EXE"}}
	analysis.Issues = []PathExtIssue{{Type: "info", Message: "Test"}}

	if len(analysis.Current) != 2 {
		t.Error("PathExtAnalysis not set correctly")
	}
}

func TestPathExtOptimization_Struct(t *testing.T) {
	opt := PathExtOptimization{
		Original:        ".BAT;.EXE",
		Optimized:       []string{".EXE", ".BAT"},
		OptimizedString: ".EXE;.BAT",
		Changed:         true,
	}

	if opt.Original == "" || !opt.Changed {
		t.Error("PathExtOptimization not set correctly")
	}
}

func TestPathExtIssue_Struct(t *testing.T) {
	issue := PathExtIssue{Type: "order", Message: ".EXE is not first", Impact: "high"}

	if issue.Type != "order" || issue.Message == "" {
		t.Error("PathExtIssue not set correctly")
	}
}

func TestIndexOf(t *testing.T) {
	slice := []string{"a", "b", "c", "d"}

	tests := []struct {
		item     string
		expected int
	}{
		{"a", 0}, {"b", 1}, {"c", 2}, {"d", 3}, {"e", -1},
	}

	for _, tc := range tests {
		result := indexOf(slice, tc.item)
		if result != tc.expected {
			t.Errorf("indexOf(%s) = %d, want %d", tc.item, result, tc.expected)
		}
	}
}

func TestIndexOf_Empty(t *testing.T) {
	var slice []string
	if indexOf(slice, "a") != -1 {
		t.Error("indexOf(empty, 'a') should be -1")
	}
}

func TestContains(t *testing.T) {
	slice := []string{".EXE", ".CMD", ".BAT"}

	if !contains(slice, ".EXE") {
		t.Error("contains should return true for .EXE")
	}
	if contains(slice, ".COM") {
		t.Error("contains should return false for .COM")
	}
}

func TestContains_Empty(t *testing.T) {
	var slice []string
	if contains(slice, ".EXE") {
		t.Error("contains(empty) should be false")
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"}, {1, "1"}, {100, "100"}, {-1, "-1"}, {-99, "-99"},
	}

	for _, tc := range tests {
		if itoa(tc.input) != tc.expected {
			t.Errorf("itoa(%d) = %s, want %s", tc.input, itoa(tc.input), tc.expected)
		}
	}
}

func TestAnalyzePathExt(t *testing.T) {
	analysis := AnalyzePathExt()

	if len(analysis.Current) == 0 {
		t.Log("Current PATHEXT is empty (using default)")
	}
}

func TestOptimizePathExt(t *testing.T) {
	result := OptimizePathExt(true)

	if len(result.Optimized) > 0 && result.Optimized[0] != ".EXE" {
		t.Errorf("Expected .EXE first, got %s", result.Optimized[0])
	}
}

func TestOptimizePathExt_RemoveRarely(t *testing.T) {
	result := OptimizePathExt(false)

	for _, ext := range result.Optimized {
		for _, rem := range RemovableExtensions {
			if ext == rem {
				t.Errorf("Rarely used extension %s should be removed", ext)
			}
		}
	}
}

func TestGetCurrentPathExt_FromEnv(t *testing.T) {
	pathExt := os.Getenv("PATHEXT")
	if runtime.GOOS == "windows" && pathExt == "" {
		t.Log("PATHEXT is empty on Windows")
	}
	if pathExt != "" && !strings.Contains(strings.ToUpper(pathExt), ".EXE") {
		t.Error("PATHEXT should contain .EXE")
	}
}

func BenchmarkParsePathExt(b *testing.B) {
	input := ".EXE;.CMD;.BAT;.COM;.VBS;.VBE;.JS;.JSE;.WSF;.WSH;.MSC;.PY"
	for i := 0; i < b.N; i++ {
		ParsePathExt(input)
	}
}

func BenchmarkGetExtensionInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetExtensionInfo(".EXE")
	}
}

func BenchmarkItoa(b *testing.B) {
	for i := 0; i < b.N; i++ {
		itoa(12345)
	}
}

func BenchmarkIndexOf(b *testing.B) {
	slice := []string{".EXE", ".CMD", ".BAT", ".COM", ".VBS", ".VBE", ".JS", ".JSE", ".WSF", ".WSH"}
	for i := 0; i < b.N; i++ {
		indexOf(slice, ".WSH")
	}
}

func TestGetCurrentPathExt_User(t *testing.T) {
	result := GetCurrentPathExt("User")
	t.Logf("User PATHEXT: %s", result)
}

func TestGetCurrentPathExt_System(t *testing.T) {
	result := GetCurrentPathExt("Machine")
	t.Logf("System PATHEXT: %s", result)
}

func TestGetCurrentPathExt_Process(t *testing.T) {
	result := GetCurrentPathExt("Process")
	if result == "" {
		t.Log("Process PATHEXT is empty")
	} else {
		t.Logf("Process PATHEXT: %s", result)
	}
}

func TestHasUserPathExt(t *testing.T) {
	result := HasUserPathExt()
	t.Logf("HasUserPathExt: %v", result)
}

func TestApplyPathExt_User(t *testing.T) {
	// Get the mock runner to verify calls
	mock := getMockRunner(t)
	beforeCount := len(mock.Calls)

	err := ApplyPathExt(".EXE;.CMD;.BAT", "User")
	if err != nil {
		t.Logf("ApplyPathExt error: %v", err)
	}

	// Verify the mock was called (not real PowerShell)
	afterCount := len(mock.Calls)
	if afterCount <= beforeCount {
		t.Error("Mock should have been called for ApplyPathExt")
	}

	// Verify the command contained SetEnvironmentVariable
	lastCall := mock.Calls[len(mock.Calls)-1]
	if !strings.Contains(lastCall, "SetEnvironmentVariable") {
		// Check second to last call (might be broadcast)
		if len(mock.Calls) > 1 {
			prevCall := mock.Calls[len(mock.Calls)-2]
			if !strings.Contains(prevCall, "SetEnvironmentVariable") {
				t.Error("Expected SetEnvironmentVariable in mock calls")
			}
		}
	}
}

func TestApplyPathExt_System(t *testing.T) {
	// Get the mock runner to verify calls
	mock := getMockRunner(t)
	beforeCount := len(mock.Calls)

	err := ApplyPathExt(".EXE;.CMD;.BAT", "Machine")
	if err != nil {
		t.Logf("ApplyPathExt error (expected without admin): %v", err)
	}

	// Verify the mock was called (not real PowerShell)
	afterCount := len(mock.Calls)
	if afterCount <= beforeCount {
		t.Error("Mock should have been called for ApplyPathExt")
	}
}

func TestAnalyzePathExt_Issues(t *testing.T) {
	analysis := AnalyzePathExt()

	if analysis.CurrentWithInfo == nil {
		t.Log("No CurrentWithInfo")
	}

	for _, issue := range analysis.Issues {
		t.Logf("Issue: %s - %s (impact: %s)", issue.Type, issue.Message, issue.Impact)
	}
}

func TestOptimizePathExt_KeepAll(t *testing.T) {
	result := OptimizePathExt(true)

	if result.OptimizedString == "" && len(result.Optimized) > 0 {
		t.Error("OptimizedString should not be empty when Optimized has entries")
	}
	t.Logf("KeepAll - Changed: %v, Original: %s", result.Changed, result.Original)
}

func TestOptimizePathExt_RemoveRarely_Details(t *testing.T) {
	result := OptimizePathExt(false)

	t.Logf("RemoveRarely - Changed: %v, Optimized count: %d", result.Changed, len(result.Optimized))

	// Verify rarely used extensions are removed
	for _, ext := range RemovableExtensions {
		for _, opt := range result.Optimized {
			if ext == opt {
				t.Logf("Warning: Rarely used extension %s still present", ext)
			}
		}
	}
}

// ============================================================================
// Tests for refactored helper functions
// ============================================================================

func TestCheckUserPathExt(t *testing.T) {
	analysis := &PathExtAnalysis{
		HasUserPathExt: true,
	}
	checkUserPathExt(analysis)

	if len(analysis.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(analysis.Issues))
	}
	if analysis.Issues[0].Type != "info" {
		t.Error("Expected info issue type")
	}
}

func TestCheckUserPathExt_NoUser(t *testing.T) {
	analysis := &PathExtAnalysis{
		HasUserPathExt: false,
	}
	checkUserPathExt(analysis)

	if len(analysis.Issues) != 0 {
		t.Error("Expected no issues when HasUserPathExt is false")
	}
}

func TestCheckExePosition_First(t *testing.T) {
	analysis := &PathExtAnalysis{IsOptimal: true}
	current := []string{".EXE", ".CMD", ".BAT"}

	checkExePosition(analysis, current)

	if !analysis.IsOptimal {
		t.Error("Should remain optimal when .EXE is first")
	}
	if len(analysis.Issues) != 0 {
		t.Error("Expected no issues")
	}
}

func TestCheckExePosition_NotFirst(t *testing.T) {
	analysis := &PathExtAnalysis{IsOptimal: true}
	current := []string{".CMD", ".EXE", ".BAT"}

	checkExePosition(analysis, current)

	if analysis.IsOptimal {
		t.Error("Should not be optimal when .EXE is not first")
	}
	if len(analysis.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(analysis.Issues))
	}
	if analysis.Issues[0].Impact != "high" {
		t.Error("Expected high impact")
	}
}

func TestCheckCmdBatOrder_Correct(t *testing.T) {
	analysis := &PathExtAnalysis{IsOptimal: true}
	current := []string{".EXE", ".CMD", ".BAT"}

	checkCmdBatOrder(analysis, current)

	if !analysis.IsOptimal {
		t.Error("Should remain optimal")
	}
}

func TestCheckCmdBatOrder_Wrong(t *testing.T) {
	analysis := &PathExtAnalysis{IsOptimal: true}
	current := []string{".EXE", ".BAT", ".CMD"}

	checkCmdBatOrder(analysis, current)

	if analysis.IsOptimal {
		t.Error("Should not be optimal when .BAT before .CMD")
	}
}

func TestCheckPythonPosition_High(t *testing.T) {
	analysis := &PathExtAnalysis{}
	current := []string{".EXE", ".CMD", ".PY"}

	checkPythonPosition(analysis, current)

	if len(analysis.Issues) != 0 {
		t.Error("Expected no issues when .PY is in high position")
	}
}

func TestCheckPythonPosition_Low(t *testing.T) {
	analysis := &PathExtAnalysis{}
	current := []string{".EXE", ".CMD", ".BAT", ".COM", ".VBS", ".VBE", ".JS", ".PY"}

	checkPythonPosition(analysis, current)

	if len(analysis.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(analysis.Issues))
	}
	if analysis.Issues[0].Type != "suggestion" {
		t.Error("Expected suggestion type")
	}
}

func TestCheckPythonPosition_NotPresent(t *testing.T) {
	analysis := &PathExtAnalysis{}
	current := []string{".EXE", ".CMD", ".BAT"}

	checkPythonPosition(analysis, current)

	if len(analysis.Issues) != 0 {
		t.Error("Expected no issues when .PY not present")
	}
}

func TestCheckRemovableExtensions_None(t *testing.T) {
	analysis := &PathExtAnalysis{}
	current := []string{".EXE", ".CMD", ".BAT"}

	checkRemovableExtensions(analysis, current)

	if len(analysis.Issues) != 0 {
		t.Error("Expected no issues")
	}
}

func TestCheckRemovableExtensions_Present(t *testing.T) {
	analysis := &PathExtAnalysis{}
	current := []string{".EXE", ".CMD", ".VBE", ".JSE"}

	checkRemovableExtensions(analysis, current)

	if len(analysis.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(analysis.Issues))
	}
	if analysis.Issues[0].Type != "bloat" {
		t.Error("Expected bloat type")
	}
}

func TestCheckComBeforeExe_After(t *testing.T) {
	analysis := &PathExtAnalysis{IsOptimal: true}
	current := []string{".EXE", ".COM", ".CMD"}

	checkComBeforeExe(analysis, current)

	if !analysis.IsOptimal {
		t.Error("Should remain optimal when .COM after .EXE")
	}
}

func TestCheckComBeforeExe_Before(t *testing.T) {
	analysis := &PathExtAnalysis{IsOptimal: true}
	current := []string{".COM", ".EXE", ".CMD"}

	checkComBeforeExe(analysis, current)

	if analysis.IsOptimal {
		t.Error("Should not be optimal when .COM before .EXE")
	}
}

func TestCheckComBeforeExe_NoEither(t *testing.T) {
	analysis := &PathExtAnalysis{IsOptimal: true}
	current := []string{".CMD", ".BAT"}

	checkComBeforeExe(analysis, current)

	if !analysis.IsOptimal {
		t.Error("Should remain optimal when neither present")
	}
}
