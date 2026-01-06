package path

import (
	"strings"
)

// DefaultPathExt is the default Windows PATHEXT
const DefaultPathExt = ".COM;.EXE;.BAT;.CMD;.VBS;.VBE;.JS;.JSE;.WSF;.WSH;.MSC"

// ExtensionInfo contains information about a file extension
type ExtensionInfo struct {
	Ext          string
	Description  string
	Priority     int // Lower = higher priority
	IsLegacy     bool
	IsRarelyUsed bool
}

// ExtensionDatabase contains descriptions for known extensions
var ExtensionDatabase = map[string]ExtensionInfo{
	".EXE": {".EXE", "Windows Executable (most common)", 1, false, false},
	".CMD": {".CMD", "Windows Command Script (modern)", 2, false, false},
	".BAT": {".BAT", "Batch File (legacy, use .CMD)", 3, true, false},
	".COM": {".COM", "DOS Executable (obsolete)", 10, true, false},
	".PS1": {".PS1", "PowerShell Script", 4, false, false},
	".PY":  {".PY", "Python Script", 5, false, false},
	".PYW": {".PYW", "Python Script (no console)", 6, false, false},
	".JS":  {".JS", "JScript (Windows Script Host)", 7, false, true},
	".MSC": {".MSC", "Microsoft Management Console", 8, false, false},
	".VBS": {".VBS", "VBScript", 9, true, true},
	".VBE": {".VBE", "Encoded VBScript (rarely used)", 11, true, true},
	".JSE": {".JSE", "Encoded JScript (rarely used)", 12, true, true},
	".WSF": {".WSF", "Windows Script File (rarely used)", 13, true, true},
	".WSH": {".WSH", "Windows Script Host Settings (rarely used)", 14, true, true},
}

// OptimalOrder is the recommended extension order
var OptimalOrder = []string{
	".EXE", // Most common executable
	".CMD", // Modern command scripts
	".BAT", // Legacy batch files
	".PS1", // PowerShell scripts
	".PY",  // Python scripts
	".PYW", // Python scripts (no console)
	".COM", // Legacy DOS (after modern formats)
	".MSC", // Management console
	".JS",  // JScript (rarely used standalone)
	".VBS", // VBScript (legacy)
	".VBE", // Encoded VBScript
	".JSE", // Encoded JScript
	".WSF", // Windows Script File
	".WSH", // Windows Script Host
}

// RemovableExtensions are rarely used and can be removed
var RemovableExtensions = []string{".VBE", ".JSE", ".WSF", ".WSH"}

// PathExtIssue represents an issue found in PATHEXT
type PathExtIssue struct {
	Type    string // order, bloat, legacy, info, suggestion
	Message string
	Impact  string // high, medium, low, info
}

// PathExtAnalysis contains the analysis results
type PathExtAnalysis struct {
	Current         []string
	CurrentWithInfo []ExtensionInfo
	Issues          []PathExtIssue
	Recommendations []string
	IsOptimal       bool
	HasUserPathExt  bool
}

// PathExtOptimization contains optimization results
type PathExtOptimization struct {
	Original        string
	Optimized       []string
	OptimizedString string
	Changed         bool
}

// GetCurrentPathExt gets the current PATHEXT value
func GetCurrentPathExt(scope string) string {
	var command string
	switch scope {
	case "System":
		command = `[Environment]::GetEnvironmentVariable('PATHEXT', 'Machine')`
	case "User":
		command = `[Environment]::GetEnvironmentVariable('PATHEXT', 'User')`
	default:
		// Effective: User overrides System
		command = `
			$user = [Environment]::GetEnvironmentVariable('PATHEXT', 'User')
			$system = [Environment]::GetEnvironmentVariable('PATHEXT', 'Machine')
			if ($user) { $user } else { $system }
		`
	}

	result, err := RunPowerShell(command)
	if err != nil || result == "" {
		return DefaultPathExt
	}
	return result
}

// HasUserPathExt checks if user has a custom PATHEXT defined
func HasUserPathExt() bool {
	result, err := RunPowerShell(`[Environment]::GetEnvironmentVariable('PATHEXT', 'User')`)
	return err == nil && result != ""
}

// ParsePathExt parses PATHEXT into a slice
func ParsePathExt(pathext string) []string {
	if pathext == "" {
		pathext = GetCurrentPathExt("Effective")
	}
	parts := strings.Split(pathext, ";")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToUpper(p))
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// GetExtensionInfo returns info about an extension
func GetExtensionInfo(ext string) ExtensionInfo {
	ext = strings.ToUpper(ext)
	if info, ok := ExtensionDatabase[ext]; ok {
		return info
	}
	return ExtensionInfo{
		Ext:          ext,
		Description:  "Unknown extension",
		Priority:     99,
		IsLegacy:     false,
		IsRarelyUsed: false,
	}
}

// AnalyzePathExt analyzes PATHEXT for optimization opportunities
// checkUserPathExt checks if user PATHEXT is defined
func checkUserPathExt(analysis *PathExtAnalysis) {
	if analysis.HasUserPathExt {
		analysis.Issues = append(analysis.Issues, PathExtIssue{
			Type:    "info",
			Message: "User-level PATHEXT is defined (overrides System)",
			Impact:  "info",
		})
	}
}

// checkExePosition checks if .EXE is in the first position
func checkExePosition(analysis *PathExtAnalysis, current []string) {
	exeIndex := indexOf(current, ".EXE")
	if exeIndex != 0 {
		analysis.Issues = append(analysis.Issues, PathExtIssue{
			Type:    "order",
			Message: ".EXE is not first (currently position " + itoa(exeIndex+1) + ")",
			Impact:  "high",
		})
		analysis.Recommendations = append(analysis.Recommendations, "Move .EXE to the first position")
		analysis.IsOptimal = false
	}
}

// checkCmdBatOrder checks if .CMD comes before .BAT
func checkCmdBatOrder(analysis *PathExtAnalysis, current []string) {
	cmdIndex := indexOf(current, ".CMD")
	batIndex := indexOf(current, ".BAT")
	if cmdIndex > batIndex && batIndex >= 0 {
		analysis.Issues = append(analysis.Issues, PathExtIssue{
			Type:    "order",
			Message: ".CMD comes after .BAT (modern scripts use .CMD)",
			Impact:  "medium",
		})
		analysis.Recommendations = append(analysis.Recommendations, "Move .CMD before .BAT")
		analysis.IsOptimal = false
	}
}

// checkPythonPosition checks if .PY is in a reasonable position
func checkPythonPosition(analysis *PathExtAnalysis, current []string) {
	pyIndex := indexOf(current, ".PY")
	if pyIndex > 5 && pyIndex != -1 {
		analysis.Issues = append(analysis.Issues, PathExtIssue{
			Type:    "suggestion",
			Message: ".PY is at position " + itoa(pyIndex+1) + " (consider moving higher if you use Python)",
			Impact:  "low",
		})
	}
}

// checkRemovableExtensions checks for rarely-used extensions
func checkRemovableExtensions(analysis *PathExtAnalysis, current []string) {
	var removable []string
	for _, ext := range current {
		if contains(RemovableExtensions, ext) {
			removable = append(removable, ext)
		}
	}
	if len(removable) > 0 {
		analysis.Issues = append(analysis.Issues, PathExtIssue{
			Type:    "bloat",
			Message: "Rarely-used extensions present: " + strings.Join(removable, ", "),
			Impact:  "low",
		})
		analysis.Recommendations = append(analysis.Recommendations, "Consider removing: "+strings.Join(removable, ", "))
	}
}

// checkComBeforeExe checks if .COM is checked before .EXE
func checkComBeforeExe(analysis *PathExtAnalysis, current []string) {
	comIndex := indexOf(current, ".COM")
	exeIndex := indexOf(current, ".EXE")
	if comIndex >= 0 && comIndex < exeIndex {
		analysis.Issues = append(analysis.Issues, PathExtIssue{
			Type:    "legacy",
			Message: ".COM is checked before .EXE (wastes cycles on obsolete format)",
			Impact:  "medium",
		})
		analysis.Recommendations = append(analysis.Recommendations, "Move .COM after .EXE")
		analysis.IsOptimal = false
	}
}

func AnalyzePathExt() PathExtAnalysis {
	current := ParsePathExt("")
	analysis := PathExtAnalysis{
		Current:        current,
		IsOptimal:      true,
		HasUserPathExt: HasUserPathExt(),
	}

	// Build current with info
	for _, ext := range current {
		analysis.CurrentWithInfo = append(analysis.CurrentWithInfo, GetExtensionInfo(ext))
	}

	checkUserPathExt(&analysis)
	checkExePosition(&analysis, current)
	checkCmdBatOrder(&analysis, current)
	checkPythonPosition(&analysis, current)
	checkRemovableExtensions(&analysis, current)
	checkComBeforeExe(&analysis, current)

	return analysis
}

// OptimizePathExt creates an optimized PATHEXT
func OptimizePathExt(keepAll bool) PathExtOptimization {
	current := ParsePathExt("")
	original := strings.Join(current, ";")

	// Build optimized list
	optimized := make([]string, 0, len(current))
	added := make(map[string]bool)

	// Add extensions in optimal order (only if they exist in current)
	for _, ext := range OptimalOrder {
		if contains(current, ext) {
			if keepAll || !contains(RemovableExtensions, ext) {
				optimized = append(optimized, ext)
				added[ext] = true
			}
		}
	}

	// Add any remaining extensions not in optimal order
	for _, ext := range current {
		if !added[ext] {
			if keepAll || !contains(RemovableExtensions, ext) {
				optimized = append(optimized, ext)
			}
		}
	}

	optimizedStr := strings.Join(optimized, ";")

	return PathExtOptimization{
		Original:        original,
		Optimized:       optimized,
		OptimizedString: optimizedStr,
		Changed:         original != optimizedStr,
	}
}

// ApplyPathExt applies optimized PATHEXT
func ApplyPathExt(value, scope string) error {
	target := "User"
	if scope == "System" {
		target = "Machine"
	}

	command := `[Environment]::SetEnvironmentVariable('PATHEXT', '` + value + `', '` + target + `')`
	_, err := RunPowerShell(command)
	if err == nil {
		BroadcastEnvChange()
	}
	return err
}

// Helper functions
func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

func contains(slice []string, item string) bool {
	return indexOf(slice, item) >= 0
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	negative := i < 0
	if negative {
		i = -i
	}
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	if negative {
		s = "-" + s
	}
	return s
}
