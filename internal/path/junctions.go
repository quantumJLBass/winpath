package path

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Junction represents a directory junction
type Junction struct {
	Name   string
	Path   string
	Target string
}

// JunctionSuggestion represents a suggested junction
type JunctionSuggestion struct {
	OriginalPath  string
	SuggestedName string
	JunctionPath  string
	SavedChars    int
}

// GetJunctionFolder returns the configured junction folder
func GetJunctionFolder() string {
	config := LoadConfig()
	return config.JunctionFolder
}

// SetJunctionFolder sets the junction folder
func SetJunctionFolder(folder string) error {
	config := LoadConfig()
	config.JunctionFolder = folder
	return SaveConfig(config)
}

// EnsureJunctionFolder creates the junction folder if it doesn't exist
func EnsureJunctionFolder() error {
	folder := GetJunctionFolder()
	return os.MkdirAll(folder, 0755)
}

// ListJunctions returns all junctions in the junction folder
func ListJunctions() []Junction {
	folder := GetJunctionFolder()

	// Use PowerShell to properly detect junctions
	command := fmt.Sprintf(`
		$folder = '%s'
		if (Test-Path $folder) {
			Get-ChildItem -Path $folder -Force | Where-Object { $_.Attributes -match 'ReparsePoint' } | ForEach-Object {
				$target = $_.Target
				if ($target -is [array]) { $target = $target[0] }
				"$($_.Name)|$target"
			}
		}
	`, strings.ReplaceAll(folder, "'", "''"))

	result, err := RunPowerShell(command)
	if err != nil || result == "" {
		return []Junction{}
	}

	junctions := make([]Junction, 0)
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) == 2 {
			junctions = append(junctions, Junction{
				Name:   parts[0],
				Path:   filepath.Join(folder, parts[0]),
				Target: parts[1],
			})
		}
	}

	return junctions
}

// CreateJunction creates a new junction
func CreateJunction(name, target string) error {
	folder := GetJunctionFolder()
	if err := EnsureJunctionFolder(); err != nil {
		return err
	}

	junctionPath := filepath.Join(folder, name)

	// Check if junction already exists
	if _, err := os.Stat(junctionPath); err == nil {
		return fmt.Errorf("junction %s already exists", name)
	}

	// Check if target exists
	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("target path does not exist: %s", target)
	}

	// Create junction using mklink /J (requires appropriate permissions)
	command := fmt.Sprintf(`cmd /c mklink /J "%s" "%s"`, junctionPath, target)
	_, err := RunPowerShell(command)
	return err
}

// RemoveJunction removes a junction
func RemoveJunction(name string) error {
	folder := GetJunctionFolder()
	junctionPath := filepath.Join(folder, name)

	// Use rmdir to remove junction without deleting target contents
	command := fmt.Sprintf(`cmd /c rmdir "%s"`, junctionPath)
	_, err := RunPowerShell(command)
	return err
}

// SuggestJunctionCandidates analyzes PATH and suggests junction candidates
func SuggestJunctionCandidates() []JunctionSuggestion {
	sysPath, _ := GetPathRaw("System")
	usrPath, _ := GetPathRaw("User")

	allPaths := append(ParsePath(sysPath), ParsePath(usrPath)...)
	folder := GetJunctionFolder()

	suggestions := make([]JunctionSuggestion, 0)
	seen := make(map[string]bool)
	usedNames := make(map[string]int) // Track how many times each name is used

	// First, get existing junctions to avoid conflicts
	existingJunctions := ListJunctions()
	for _, j := range existingJunctions {
		usedNames[strings.ToLower(j.Name)] = 1
	}

	for _, p := range allPaths {
		// Skip paths with variables or already short paths
		if strings.Contains(p, "%") || len(p) < 30 {
			continue
		}

		// Skip if already in junction folder
		if strings.HasPrefix(strings.ToLower(p), strings.ToLower(folder)) {
			continue
		}

		// Skip duplicates
		normalized := NormalizePath(p)
		if seen[normalized] {
			continue
		}
		seen[normalized] = true

		// Generate suggested name from path
		shortName := generateJunctionName(p, usedNames)
		if shortName == "" {
			continue
		}

		junctionPath := filepath.Join(folder, shortName)
		savedChars := len(p) - len(junctionPath)

		// Only suggest if it saves significant chars
		if savedChars > 20 {
			suggestions = append(suggestions, JunctionSuggestion{
				OriginalPath:  p,
				SuggestedName: shortName,
				JunctionPath:  junctionPath,
				SavedChars:    savedChars,
			})
			usedNames[strings.ToLower(shortName)]++
		}
	}

	// Sort by savings descending
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].SavedChars > suggestions[j].SavedChars
	})

	return suggestions
}

// generateJunctionName creates a unique short name for a junction
// cleanNameChars removes invalid characters from a name, keeping only alphanumeric, dash, underscore
func cleanNameChars(name string, keepDashUnderscore bool) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		if keepDashUnderscore && (r == '-' || r == '_') {
			return r
		}
		return -1
	}, strings.ToLower(name))
}

// truncateName truncates a name to maxLen characters
func truncateName(name string, maxLen int) string {
	if len(name) > maxLen {
		return name[:maxLen]
	}
	return name
}

// tryUniqueWithParent attempts to create a unique name using parent folder prefix
func tryUniqueWithParent(path, cleanName string, usedNames map[string]int) string {
	parent := filepath.Base(filepath.Dir(path))
	if parent == "" || parent == "." || parent == "\\" {
		return ""
	}

	parentClean := truncateName(cleanNameChars(parent, false), 4)
	if parentClean == "" {
		return ""
	}

	uniqueName := truncateName(parentClean+"-"+cleanName, 12)
	if usedNames[strings.ToLower(uniqueName)] == 0 {
		return uniqueName
	}
	return ""
}

// tryUniqueWithNumber attempts to create a unique name with numeric suffix
func tryUniqueWithNumber(cleanName string, usedNames map[string]int) string {
	for i := 2; i <= 99; i++ {
		suffix := fmt.Sprintf("%d", i)
		baseLen := 10 - len(suffix)
		if baseLen > len(cleanName) {
			baseLen = len(cleanName)
		}
		numName := cleanName[:baseLen] + suffix
		if usedNames[strings.ToLower(numName)] == 0 {
			return numName
		}
	}
	return ""
}

func generateJunctionName(path string, usedNames map[string]int) string {
	baseName := filepath.Base(path)
	if baseName == "" || baseName == "." || baseName == "\\" {
		return ""
	}

	cleanName := cleanNameChars(baseName, true)
	if cleanName == "" {
		cleanName = "dir"
	}
	cleanName = truncateName(cleanName, 8)

	// Check if name is already used
	if usedNames[strings.ToLower(cleanName)] == 0 {
		return cleanName
	}

	// Try with parent folder prefix
	if uniqueName := tryUniqueWithParent(path, cleanName, usedNames); uniqueName != "" {
		return uniqueName
	}

	// Try with numeric suffix
	return tryUniqueWithNumber(cleanName, usedNames)
}

// CalculateJunctionSavings calculates total chars saved if all suggested junctions were applied
func CalculateJunctionSavings(suggestions []JunctionSuggestion) int {
	total := 0
	for _, s := range suggestions {
		total += s.SavedChars
	}
	return total
}
