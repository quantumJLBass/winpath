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
// extractVarAndSuffix extracts the variable part and suffix from a path
func extractVarAndSuffix(pathWithVar string) (varPart, suffix string, ok bool) {
	if !strings.Contains(pathWithVar, "%") {
		return "", "", false
	}

	lastPercent := strings.LastIndex(pathWithVar, "%")
	if lastPercent == -1 || lastPercent == len(pathWithVar)-1 {
		return "", "", false
	}

	varPart = pathWithVar[:lastPercent+1]
	suffix = pathWithVar[lastPercent+1:]

	if suffix == "" || suffix == "\\" {
		return "", "", false
	}

	return varPart, suffix, true
}

// getShortPathCommand returns the PowerShell command to get short path
func getShortPathCommand(expanded string) string {
	return fmt.Sprintf(`
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
}

func ShortenSuffix(pathWithVar string) (string, bool) {
	varPart, _, ok := extractVarAndSuffix(pathWithVar)
	if !ok {
		return pathWithVar, false
	}

	expanded := ExpandEnvVars(pathWithVar)
	if expanded == "" || expanded == pathWithVar {
		return pathWithVar, false
	}

	command := getShortPathCommand(expanded)
	shortExpanded, err := RunPowerShell(command)
	if err != nil || shortExpanded == "" || shortExpanded == expanded {
		return pathWithVar, false
	}

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
