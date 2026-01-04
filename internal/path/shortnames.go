package path

import (
	"fmt"
	"strings"
)

// ToShortPath converts a path to its 8.3 short form if possible
func ToShortPath(path string) (string, bool) {
	if path == "" || strings.Contains(path, "%") {
		return path, false
	}

	command := fmt.Sprintf(`
		$ErrorActionPreference = 'SilentlyContinue'
		$path = '%s'
		if (Test-Path -LiteralPath $path) {
			$fso = New-Object -ComObject Scripting.FileSystemObject
			try {
				if ((Get-Item -LiteralPath $path).PSIsContainer) {
					$fso.GetFolder($path).ShortPath
				} else {
					$fso.GetFile($path).ShortPath
				}
			} catch { $path }
		} else { $path }
	`, strings.ReplaceAll(path, "'", "''"))

	result, err := RunPowerShell(command)
	if err != nil || result == "" || result == path {
		return path, false
	}

	// Only return if actually shorter
	if len(result) < len(path) {
		return result, true
	}
	return path, false
}

// ShortenSuffix shortens the suffix of a path that contains environment variables
// E.g., %LOCALAPPDATA%\Microsoft\WinGet -> %LOCALAPPDATA%\MICROS~1\WinGet
func ShortenSuffix(pathWithVar string) (string, bool) {
	if !strings.Contains(pathWithVar, "%") {
		return pathWithVar, false
	}

	// Find where the variable ends
	lastPercent := strings.LastIndex(pathWithVar, "%")
	if lastPercent == -1 || lastPercent == len(pathWithVar)-1 {
		return pathWithVar, false
	}

	varPart := pathWithVar[:lastPercent+1]
	suffix := pathWithVar[lastPercent+1:]

	if suffix == "" || suffix == "\\" {
		return pathWithVar, false
	}

	// Expand the variable to get the real path
	expanded := ExpandEnvVars(pathWithVar)
	if expanded == "" || expanded == pathWithVar {
		return pathWithVar, false
	}

	// Get the 8.3 short path of the expanded version
	command := fmt.Sprintf(`
		$ErrorActionPreference = 'SilentlyContinue'
		$path = '%s'
		if (Test-Path -LiteralPath $path) {
			$fso = New-Object -ComObject Scripting.FileSystemObject
			try {
				if ((Get-Item -LiteralPath $path).PSIsContainer) {
					$fso.GetFolder($path).ShortPath
				} else {
					$fso.GetFile($path).ShortPath
				}
			} catch { $path }
		} else { $path }
	`, strings.ReplaceAll(expanded, "'", "''"))

	shortExpanded, err := RunPowerShell(command)
	if err != nil || shortExpanded == "" || shortExpanded == expanded {
		return pathWithVar, false
	}

	// Find what the variable expands to
	expandedVarValue := ExpandEnvVars(varPart)
	if strings.HasPrefix(strings.ToLower(shortExpanded), strings.ToLower(expandedVarValue)) {
		shortSuffix := shortExpanded[len(expandedVarValue):]
		newPath := varPart + shortSuffix

		if len(newPath) < len(pathWithVar) {
			return newPath, true
		}
	}

	return pathWithVar, false
}
