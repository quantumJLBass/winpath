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
func generateJunctionName(path string, usedNames map[string]int) string {
	// Get the last component
	baseName := filepath.Base(path)
	if baseName == "" || baseName == "." || baseName == "\\" {
		return ""
	}

	// Clean up the base name
	cleanName := strings.ToLower(baseName)
	cleanName = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, cleanName)

	if cleanName == "" {
		cleanName = "dir"
	}

	// Truncate to 8 chars max for the base
	if len(cleanName) > 8 {
		cleanName = cleanName[:8]
	}

	// Check if name is already used
	baseLower := strings.ToLower(cleanName)
	if usedNames[baseLower] == 0 {
		return cleanName
	}

	// Name collision - try to make it unique using parent folder
	parent := filepath.Base(filepath.Dir(path))
	if parent != "" && parent != "." && parent != "\\" {
		parentClean := strings.ToLower(parent)
		parentClean = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				return r
			}
			return -1
		}, parentClean)

		if len(parentClean) > 4 {
			parentClean = parentClean[:4]
		}

		if parentClean != "" {
			uniqueName := parentClean + "-" + cleanName
			if len(uniqueName) > 12 {
				uniqueName = uniqueName[:12]
			}
			uniqueLower := strings.ToLower(uniqueName)
			if usedNames[uniqueLower] == 0 {
				return uniqueName
			}
		}
	}

	// Still collision - add a number
	for i := 2; i <= 99; i++ {
		numName := fmt.Sprintf("%s%d", cleanName, i)
		if len(numName) > 10 {
			numName = cleanName[:8-len(fmt.Sprintf("%d", i))] + fmt.Sprintf("%d", i)
		}
		numLower := strings.ToLower(numName)
		if usedNames[numLower] == 0 {
			return numName
		}
	}

	return ""
}

// CalculateJunctionSavings calculates total chars saved if all suggested junctions were applied
func CalculateJunctionSavings(suggestions []JunctionSuggestion) int {
	total := 0
	for _, s := range suggestions {
		total += s.SavedChars
	}
	return total
}
