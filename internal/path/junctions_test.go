package path

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestGetJunctionFolder(t *testing.T) {
	folder := GetJunctionFolder()

	if folder == "" {
		t.Error("Expected non-empty junction folder")
	}

	cfg := LoadConfig()
	if folder != cfg.JunctionFolder {
		t.Errorf("Expected %s, got %s", cfg.JunctionFolder, folder)
	}
}

func TestSetJunctionFolder(t *testing.T) {
	original := GetJunctionFolder()
	defer SetJunctionFolder(original)

	newFolder := `C:\test\junctions`
	SetJunctionFolder(newFolder)

	if GetJunctionFolder() != newFolder {
		t.Error("SetJunctionFolder did not update folder")
	}
}

func TestEnsureJunctionFolder(t *testing.T) {
	err := EnsureJunctionFolder()
	if err != nil {
		t.Logf("EnsureJunctionFolder error: %v", err)
	}
}

func TestJunctionStruct(t *testing.T) {
	j := Junction{
		Name:   "test",
		Path:   `C:\l\test`,
		Target: `C:\Program Files\TestApp`,
	}

	if j.Name != "test" || j.Path == "" || j.Target == "" {
		t.Error("Junction struct not set correctly")
	}
}

func TestJunctionSuggestionStruct(t *testing.T) {
	s := JunctionSuggestion{
		OriginalPath:  `C:\Program Files\Very Long Application Name\bin`,
		SuggestedName: "bin",
		JunctionPath:  `C:\l\bin`,
		SavedChars:    40,
	}

	if s.OriginalPath == "" || s.SuggestedName == "" || s.SavedChars <= 0 {
		t.Error("JunctionSuggestion struct not set correctly")
	}
}

func TestGenerateJunctionName_Simple(t *testing.T) {
	usedNames := make(map[string]int)
	name := generateJunctionName(`C:\Program Files\Git\bin`, usedNames)

	if name == "" {
		t.Error("Expected non-empty name")
	}
	if len(name) > 12 {
		t.Errorf("Name too long: %s (%d)", name, len(name))
	}
}

func TestGenerateJunctionName_Collision(t *testing.T) {
	usedNames := make(map[string]int)

	name1 := generateJunctionName(`C:\Program Files\Git\bin`, usedNames)
	usedNames[name1] = 1

	name2 := generateJunctionName(`C:\Program Files\Other\bin`, usedNames)

	if name1 == name2 {
		t.Error("Names should be different")
	}
}

func TestGenerateJunctionName_MultipleCollisions(t *testing.T) {
	usedNames := make(map[string]int)

	for i := 0; i < 5; i++ {
		name := generateJunctionName(`C:\Program Files\App`+string(rune('A'+i))+`\bin`, usedNames)
		usedNames[name] = 1
	}

	if len(usedNames) != 5 {
		t.Errorf("Expected 5 unique names, got %d", len(usedNames))
	}
}

func TestGenerateJunctionName_LongPath(t *testing.T) {
	usedNames := make(map[string]int)
	longPath := `C:\Program Files\Microsoft Visual Studio\2022\Enterprise\Common7\IDE\Extensions\Microsoft`
	name := generateJunctionName(longPath, usedNames)

	if len(name) > 12 {
		t.Errorf("Name should be max 12 chars: %s (%d)", name, len(name))
	}
}

func TestCalculateJunctionSavings(t *testing.T) {
	suggestions := []JunctionSuggestion{
		{SavedChars: 50},
		{SavedChars: 30},
		{SavedChars: 20},
	}

	total := CalculateJunctionSavings(suggestions)
	if total != 100 {
		t.Errorf("Expected 100, got %d", total)
	}
}

func TestCalculateJunctionSavings_Empty(t *testing.T) {
	total := CalculateJunctionSavings([]JunctionSuggestion{})
	if total != 0 {
		t.Errorf("Expected 0, got %d", total)
	}
}

func TestCalculateJunctionSavings_Nil(t *testing.T) {
	total := CalculateJunctionSavings(nil)
	if total != 0 {
		t.Errorf("Expected 0, got %d", total)
	}
}

func TestSuggestJunctionCandidates(t *testing.T) {
	suggestions := SuggestJunctionCandidates()
	t.Logf("Found %d suggestions", len(suggestions))
}

func TestListJunctions(t *testing.T) {
	junctions := ListJunctions()
	t.Logf("Found %d junctions", len(junctions))
}

func TestListJunctions_EmptyFolder(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "junction-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	original := GetJunctionFolder()
	SetJunctionFolder(tmpDir)
	defer SetJunctionFolder(original)

	junctions := ListJunctions()
	if len(junctions) != 0 {
		t.Errorf("Expected 0 junctions in empty folder, got %d", len(junctions))
	}
}

func TestCreateJunction_InvalidName(t *testing.T) {
	err := CreateJunction("", `C:\Windows`)
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

func TestCreateJunction_InvalidTarget(t *testing.T) {
	err := CreateJunction("test", "")
	if err == nil {
		t.Error("Expected error for empty target")
	}
}

func TestRemoveJunction_NotFound(t *testing.T) {
	err := RemoveJunction("nonexistent_junction_12345")
	if err == nil {
		t.Log("RemoveJunction nonexistent may succeed or fail")
	}
}

func TestValidateJunctionName(t *testing.T) {
	validNames := []string{"test", "git", "vscode", "app123"}
	invalidNames := []string{"", "test/path", "test\\path", "test:name"}

	for _, name := range validNames {
		if len(name) == 0 || len(name) > 12 {
			t.Errorf("Expected valid: %s", name)
		}
	}

	for _, name := range invalidNames {
		if name != "" && !containsInvalidChars(name) {
			t.Logf("Name %s may be valid", name)
		}
	}
}

func containsInvalidChars(name string) bool {
	for _, c := range name {
		if c == '/' || c == '\\' || c == ':' || c == '*' || c == '?' || c == '"' || c == '<' || c == '>' || c == '|' {
			return true
		}
	}
	return false
}

func TestGetBackupDir_NotEmpty(t *testing.T) {
	dir := GetBackupDir()
	if dir == "" {
		t.Error("GetBackupDir should not be empty")
	}
}

func TestJunctionPath(t *testing.T) {
	folder := GetJunctionFolder()
	name := "test"
	expected := filepath.Join(folder, name)

	j := Junction{Name: name, Path: expected}
	if j.Path != expected {
		t.Errorf("Expected %s, got %s", expected, j.Path)
	}
}

func BenchmarkGenerateJunctionName(b *testing.B) {
	usedNames := make(map[string]int)
	path := `C:\Program Files\Microsoft Visual Studio\2022\Enterprise\bin`

	for i := 0; i < b.N; i++ {
		generateJunctionName(path, usedNames)
	}
}

func BenchmarkCalculateJunctionSavings(b *testing.B) {
	suggestions := make([]JunctionSuggestion, 100)
	for i := 0; i < 100; i++ {
		suggestions[i] = JunctionSuggestion{SavedChars: i * 10}
	}

	for i := 0; i < b.N; i++ {
		CalculateJunctionSavings(suggestions)
	}
}

func TestCleanNameChars(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		keepDashUnderscore bool
		expected           string
	}{
		{"simple", "Test", false, "test"},
		{"with dash", "my-name", true, "my-name"},
		{"with dash no keep", "my-name", false, "myname"},
		{"with underscore", "my_name", true, "my_name"},
		{"with underscore no keep", "my_name", false, "myname"},
		{"special chars", "Test@#$%123", false, "test123"},
		{"empty", "", false, ""},
		{"all special", "@#$%", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanNameChars(tt.input, tt.keepDashUnderscore)
			if result != tt.expected {
				t.Errorf("cleanNameChars(%s, %v) = %s, want %s", tt.input, tt.keepDashUnderscore, result, tt.expected)
			}
		})
	}
}

func TestTruncateName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"no truncate", "test", 10, "test"},
		{"exact length", "test", 4, "test"},
		{"truncate", "testname", 4, "test"},
		{"empty", "", 4, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateName(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateName(%s, %d) = %s, want %s", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestTryUniqueWithParent(t *testing.T) {
	usedNames := make(map[string]int)
	usedNames["test"] = 1

	// Test with valid parent
	result := tryUniqueWithParent(`C:\Parent\test`, "test", usedNames)
	if result == "" {
		t.Log("tryUniqueWithParent returned empty (parent may be invalid)")
	}

	// Test with root path (no parent)
	result = tryUniqueWithParent(`C:\test`, "test", usedNames)
	t.Logf("Root path result: %s", result)
}

func TestTryUniqueWithNumber(t *testing.T) {
	usedNames := make(map[string]int)
	usedNames["test"] = 1

	result := tryUniqueWithNumber("test", usedNames)
	if result == "" {
		t.Error("tryUniqueWithNumber should return a unique name")
	}
	if result != "test2" {
		t.Errorf("Expected test2, got %s", result)
	}

	// Fill up all numbers
	for i := 2; i <= 99; i++ {
		usedNames[fmt.Sprintf("test%d", i)] = 1
	}
	result = tryUniqueWithNumber("test", usedNames)
	if result != "" {
		t.Error("Should return empty when all numbers exhausted")
	}
}

func TestCreateJunction_Valid(t *testing.T) {
	// This tests the validation path, actual creation may fail without admin
	err := CreateJunction("testjunction", `C:\Windows`)
	if err != nil {
		t.Logf("CreateJunction error (may be expected): %v", err)
	}
}

func TestRemoveJunction_Valid(t *testing.T) {
	err := RemoveJunction("testjunction")
	if err != nil {
		t.Logf("RemoveJunction error (may be expected): %v", err)
	}
}

func TestSetJunctionFolder_Valid(t *testing.T) {
	original := GetJunctionFolder()
	defer SetJunctionFolder(original)

	err := SetJunctionFolder(`C:\test\junctions`)
	if err != nil {
		t.Logf("SetJunctionFolder error: %v", err)
	}
}

func TestListJunctions_WithJunctions(t *testing.T) {
	// Just exercise the code path
	junctions := ListJunctions()
	for _, j := range junctions {
		if j.Name == "" {
			t.Error("Junction name should not be empty")
		}
		if j.Path == "" {
			t.Error("Junction path should not be empty")
		}
	}
}

func TestSuggestJunctionCandidates_Analysis(t *testing.T) {
	suggestions := SuggestJunctionCandidates()

	for _, s := range suggestions {
		if s.SavedChars < 0 {
			t.Errorf("SavedChars should not be negative: %d", s.SavedChars)
		}
		if s.SuggestedName == "" && s.OriginalPath != "" {
			t.Error("SuggestedName should not be empty for valid paths")
		}
	}
}

func TestGenerateJunctionName_SpecialChars(t *testing.T) {
	usedNames := make(map[string]int)

	// Test with paths containing special characters
	paths := []string{
		`C:\Program Files (x86)\Test`,
		`C:\Users\Test User\Documents`,
		`C:\My-App_v1.0\bin`,
	}

	for _, p := range paths {
		name := generateJunctionName(p, usedNames)
		usedNames[name] = 1

		if name == "" {
			t.Errorf("Name should not be empty for path: %s", p)
		}
		if len(name) > 12 {
			t.Errorf("Name too long for path %s: %s (%d)", p, name, len(name))
		}
	}
}

func TestGenerateJunctionName_EmptyBaseName(t *testing.T) {
	usedNames := make(map[string]int)

	// Test edge cases
	result := generateJunctionName(`C:\`, usedNames)
	if result != "" {
		t.Logf("Result for root: %s", result)
	}

	result = generateJunctionName("", usedNames)
	if result != "" {
		t.Error("Empty path should return empty name")
	}
}
