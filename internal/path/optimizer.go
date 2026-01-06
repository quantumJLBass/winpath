package path

import (
	"os"
	"path/filepath"
	"strings"
)

// OptimizeOptions configures optimization behavior
type OptimizeOptions struct {
	RemoveDuplicates bool
	RemoveDeadPaths  bool
	ShortenPaths     bool
	SubstituteVars   bool
	ReorderPaths     bool
	Scope            string
}

// DefaultOptions returns sensible default optimization options
func DefaultOptions() OptimizeOptions {
	return OptimizeOptions{
		RemoveDuplicates: true,
		RemoveDeadPaths:  true,
		ShortenPaths:     true,
		SubstituteVars:   true,
		ReorderPaths:     false,
		Scope:            "User",
	}
}

// PathChange represents a single change made during optimization
type PathChange struct {
	Type     string // duplicate, dead, shortened, variable, reordered
	Original string
	New      string
	Saved    int
}

// PathInfo contains path metadata
type PathInfo struct {
	Raw     string
	Entries []string
	Length  int
	Count   int
}

// OptimizeMetrics contains optimization statistics
type OptimizeMetrics struct {
	DuplicatesRemoved int
	DeadPathsRemoved  int
	PathsShortened    int
	VarsSubstituted   int
	TotalSaved        int
	PercentageSaved   float64
}

// OptimizeResult contains the results of path optimization
type OptimizeResult struct {
	Original  PathInfo
	Optimized PathInfo
	Changes   []PathChange
	Metrics   OptimizeMetrics
}

// NormalizePath normalizes a path for comparison
func NormalizePath(p string) string {
	p = strings.ToLower(p)
	p = strings.TrimRight(p, "\\/")
	p = filepath.Clean(p)
	return p
}

// PathExists checks if a path exists on disk
func PathExists(path string) bool {
	// Don't check paths with unexpanded variables
	if strings.Contains(path, "%") {
		return true
	}
	_, err := os.Stat(path)
	return err == nil
}

// Optimize optimizes a PATH string
func Optimize(pathStr string, opts OptimizeOptions) OptimizeResult {
	return OptimizeWithProgress(pathStr, opts, 0, 0, nil)
}

// OptimizeWithProgress optimizes a PATH string with progress reporting
// entryProcessor handles optimization of a single PATH entry
type entryProcessor struct {
	opts   OptimizeOptions
	result *OptimizeResult
	seen   map[string]bool
}

// newEntryProcessor creates a new entry processor
func newEntryProcessor(opts OptimizeOptions, result *OptimizeResult) *entryProcessor {
	return &entryProcessor{
		opts:   opts,
		result: result,
		seen:   make(map[string]bool),
	}
}

// isDuplicate checks if entry is a duplicate
func (p *entryProcessor) isDuplicate(entry, normalized string) bool {
	if !p.opts.RemoveDuplicates {
		return false
	}
	if p.seen[normalized] {
		p.result.Changes = append(p.result.Changes, PathChange{
			Type:     "duplicate",
			Original: entry,
		})
		p.result.Metrics.DuplicatesRemoved++
		return true
	}
	p.seen[normalized] = true
	return false
}

// isDeadPath checks if entry is a dead path
func (p *entryProcessor) isDeadPath(entry string) bool {
	if !p.opts.RemoveDeadPaths {
		return false
	}
	if strings.Contains(entry, "%") {
		return false
	}
	if PathExists(entry) {
		return false
	}
	p.result.Changes = append(p.result.Changes, PathChange{
		Type:     "dead",
		Original: entry,
	})
	p.result.Metrics.DeadPathsRemoved++
	return true
}

// tryShorten attempts to shorten the path using 8.3 names
func (p *entryProcessor) tryShorten(current string) string {
	if !p.opts.ShortenPaths || strings.Contains(current, "%") {
		return current
	}
	short, shortened := ToShortPath(current)
	if !shortened || len(short) >= len(current) {
		return current
	}
	saved := len(current) - len(short)
	p.result.Changes = append(p.result.Changes, PathChange{
		Type:     "shortened",
		Original: current,
		New:      short,
		Saved:    saved,
	})
	p.result.Metrics.PathsShortened++
	p.result.Metrics.TotalSaved += saved
	return short
}

// trySubstituteVars attempts to substitute environment variables
func (p *entryProcessor) trySubstituteVars(current string) string {
	if !p.opts.SubstituteVars || strings.Contains(current, "%") {
		return current
	}
	subst, substituted := SubstituteEnvVars(current)
	if !substituted || len(subst) >= len(current) {
		return current
	}
	saved := len(current) - len(subst)
	p.result.Changes = append(p.result.Changes, PathChange{
		Type:     "variable",
		Original: current,
		New:      subst,
		Saved:    saved,
	})
	p.result.Metrics.VarsSubstituted++
	p.result.Metrics.TotalSaved += saved
	return subst
}

// tryShortenSuffix attempts to shorten the suffix after variable substitution
func (p *entryProcessor) tryShortenSuffix(current string) string {
	if !p.opts.ShortenPaths {
		return current
	}
	shortSuffix, shortened := ShortenSuffix(current)
	if !shortened || len(shortSuffix) >= len(current) {
		return current
	}
	saved := len(current) - len(shortSuffix)
	p.result.Changes = append(p.result.Changes, PathChange{
		Type:     "shortened",
		Original: current,
		New:      shortSuffix,
		Saved:    saved,
	})
	p.result.Metrics.PathsShortened++
	p.result.Metrics.TotalSaved += saved
	return shortSuffix
}

// processEntry processes a single entry and returns the optimized version or empty if skipped
func (p *entryProcessor) processEntry(entry string) (string, bool) {
	normalized := NormalizePath(entry)

	if p.isDuplicate(entry, normalized) {
		return "", false
	}
	if p.isDeadPath(entry) {
		return "", false
	}

	current := entry
	current = p.tryShorten(current)

	beforeSubst := current
	current = p.trySubstituteVars(current)
	if current != beforeSubst {
		current = p.tryShortenSuffix(current)
	}

	return current, true
}

func OptimizeWithProgress(pathStr string, opts OptimizeOptions, startIdx, total int, progress ProgressFunc) OptimizeResult {
	result := OptimizeResult{}
	entries := ParsePath(pathStr)

	result.Original.Raw = pathStr
	result.Original.Entries = entries
	result.Original.Length = len(pathStr)
	result.Original.Count = len(entries)

	processor := newEntryProcessor(opts, &result)
	optimized := make([]string, 0, len(entries))

	for i, entry := range entries {
		if progress != nil && total > 0 {
			progress(startIdx+i, total, entry)
		}

		if processed, ok := processor.processEntry(entry); ok {
			optimized = append(optimized, processed)
		}
	}

	// Apply hot paths prioritization
	config := LoadConfig()
	if len(config.HotPaths) > 0 {
		optimized = applyHotPaths(optimized, config.HotPaths)
	}

	result.Optimized.Entries = optimized
	result.Optimized.Raw = JoinPath(optimized)
	result.Optimized.Length = len(result.Optimized.Raw)
	result.Optimized.Count = len(optimized)

	if result.Original.Length > 0 {
		result.Metrics.PercentageSaved = float64(result.Original.Length-result.Optimized.Length) / float64(result.Original.Length) * 100
	}

	return result
}

// applyHotPaths moves hot paths to the front of the list
func applyHotPaths(entries []string, hotPaths []string) []string {
	if len(hotPaths) == 0 {
		return entries
	}

	// Build a set of hot paths (normalized)
	hotPathsSet := make(map[string]int)
	for i, hp := range hotPaths {
		hotPathsSet[NormalizePath(hp)] = i
	}

	// Separate into hot and regular
	hot := make([]string, len(hotPaths))
	hotFound := make([]bool, len(hotPaths))
	regular := make([]string, 0, len(entries))

	for _, entry := range entries {
		normalized := NormalizePath(entry)
		if idx, ok := hotPathsSet[normalized]; ok {
			hot[idx] = entry
			hotFound[idx] = true
		} else {
			regular = append(regular, entry)
		}
	}

	// Build result: hot paths first (in order), then regular
	result := make([]string, 0, len(entries))
	for i, h := range hot {
		if hotFound[i] {
			result = append(result, h)
		}
	}
	result = append(result, regular...)

	return result
}

// AnalyzeAll analyzes both System and User PATH
type AnalysisResult struct {
	System          OptimizeResult
	User            OptimizeResult
	CustomVariables []CustomPathVar
}

type CustomPathVar struct {
	Name    string
	FoundIn string
	Value   string
}

func AnalyzeAll(opts OptimizeOptions) AnalysisResult {
	return AnalyzeAllWithProgress(opts, nil)
}

// ProgressFunc is a callback for reporting progress
type ProgressFunc func(current, total int, item string)

func AnalyzeAllWithProgress(opts OptimizeOptions, progress ProgressFunc) AnalysisResult {
	result := AnalysisResult{}

	sysPath, _ := GetPathRaw("System")
	usrPath, _ := GetPathRaw("User")

	sysEntries := ParsePath(sysPath)
	usrEntries := ParsePath(usrPath)
	totalEntries := len(sysEntries) + len(usrEntries)

	sysOpts := opts
	sysOpts.Scope = "System"
	result.System = OptimizeWithProgress(sysPath, sysOpts, 0, totalEntries, progress)

	usrOpts := opts
	usrOpts.Scope = "User"
	result.User = OptimizeWithProgress(usrPath, usrOpts, len(sysEntries), totalEntries, progress)

	// Detect custom path variables
	if progress != nil {
		progress(totalEntries, totalEntries, "Detecting custom variables...")
	}
	result.CustomVariables = DetectCustomPathVars(sysPath, usrPath)

	return result
}

// DetectCustomPathVars finds custom PATH-like variables in the PATH strings
func DetectCustomPathVars(sysPath, usrPath string) []CustomPathVar {
	systemVars := map[string]bool{
		"systemroot": true, "windir": true, "userprofile": true,
		"appdata": true, "localappdata": true, "programfiles": true,
		"programfiles(x86)": true, "programw6432": true,
		"commonprogramfiles": true, "commonprogramfiles(x86)": true,
		"systemdrive": true, "homedrive": true, "homepath": true,
		"java_home": true, "gopath": true, "goroot": true,
		"cargo_home": true, "rustup_home": true, "nvm_home": true,
		"pnpm_home": true,
	}

	found := make(map[string]CustomPathVar)

	checkPath := func(pathStr, scope string) {
		start := 0
		for {
			i := strings.Index(pathStr[start:], "%")
			if i == -1 {
				break
			}
			i += start
			j := strings.Index(pathStr[i+1:], "%")
			if j == -1 {
				break
			}
			j += i + 1

			varName := pathStr[i+1 : j]
			if !systemVars[strings.ToLower(varName)] {
				if _, exists := found[strings.ToLower(varName)]; !exists {
					found[strings.ToLower(varName)] = CustomPathVar{
						Name:    varName,
						FoundIn: scope,
					}
				}
			}
			start = j + 1
		}
	}

	checkPath(sysPath, "System")
	checkPath(usrPath, "User")

	result := make([]CustomPathVar, 0, len(found))
	for _, v := range found {
		result = append(result, v)
	}
	return result
}
