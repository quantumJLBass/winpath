package path

import (
	"fmt"
	"os"
	"strings"

	"github.com/atotto/clipboard"
)

const (
	SystemPathKey = `HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Environment`
	UserPathKey   = `HKCU:\Environment`
)

// RunPowerShell executes a PowerShell command and returns the output
// Uses DefaultRunner which can be mocked for testing
func RunPowerShell(command string) (string, error) {
	return DefaultRunner.Run(command)
}

// GetPathRaw gets the raw PATH value from registry without expanding variables
func GetPathRaw(scope string) (string, error) {
	var command string
	if scope == "System" {
		command = `
			$key = [Microsoft.Win32.Registry]::LocalMachine.OpenSubKey('SYSTEM\CurrentControlSet\Control\Session Manager\Environment')
			if ($key) { $key.GetValue('Path', '', [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames) }
		`
	} else {
		command = `
			$key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Environment')
			if ($key) { $key.GetValue('Path', '', [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames) }
		`
	}
	return RunPowerShell(command)
}

// GetPathExpanded gets the expanded PATH value with variables resolved
func GetPathExpanded(scope string) (string, error) {
	var command string
	if scope == "System" {
		// Get raw then expand environment variables
		command = `[System.Environment]::ExpandEnvironmentVariables([Environment]::GetEnvironmentVariable('Path', 'Machine'))`
	} else {
		command = `[System.Environment]::ExpandEnvironmentVariables([Environment]::GetEnvironmentVariable('Path', 'User'))`
	}
	result, err := RunPowerShell(command)
	if err != nil {
		return result, err
	}

	// Expand 8.3 short names in a single batched PowerShell call
	entries := ParsePath(result)
	expanded := expandShortNamesBatch(entries)
	return JoinPath(expanded), nil
}

// expandShortNamesBatch expands all 8.3 short names in a single PowerShell call
// This avoids the overhead of spawning a new PowerShell process for each path
func expandShortNamesBatch(paths []string) []string {
	// Separate paths that need expansion from those that don't
	needsExpansion := make([]int, 0)
	for i, p := range paths {
		if strings.Contains(p, "~") && !strings.Contains(p, "%") {
			needsExpansion = append(needsExpansion, i)
		}
	}

	// If nothing needs expansion, return as-is
	if len(needsExpansion) == 0 {
		return paths
	}

	// Build a single PowerShell script that expands all paths at once
	var sb strings.Builder
	sb.WriteString("$paths = @(\n")
	for i, idx := range needsExpansion {
		escaped := strings.ReplaceAll(paths[idx], "'", "''")
		if i > 0 {
			sb.WriteString(",\n")
		}
		sb.WriteString(fmt.Sprintf("    '%s'", escaped))
	}
	sb.WriteString("\n)\n")
	sb.WriteString(`
$results = @()
foreach ($p in $paths) {
    $expanded = $p
    if (Test-Path -LiteralPath $p -ErrorAction SilentlyContinue) {
        try {
            $item = Get-Item -LiteralPath $p -Force -ErrorAction Stop
            $expanded = $item.FullName
        } catch {}
    }
    $results += $expanded
}
$results -join '|'
`)

	result, err := RunPowerShell(sb.String())
	if err != nil || result == "" {
		return paths // Return original on error
	}

	// Parse results and update paths
	expanded := strings.Split(strings.TrimSpace(result), "|")
	output := make([]string, len(paths))
	copy(output, paths)

	for i, idx := range needsExpansion {
		if i < len(expanded) && expanded[i] != "" {
			output[idx] = expanded[i]
		}
	}

	return output
}

// expandShortName expands a single 8.3 short name (used by other code if needed)
func expandShortName(p string) string {
	// Skip paths with environment variables
	if strings.Contains(p, "%") {
		return p
	}
	// Skip paths without 8.3 pattern
	if !strings.Contains(p, "~") {
		return p
	}
	// For single paths, just use the batch function with one item
	result := expandShortNamesBatch([]string{p})
	return result[0]
}

// SetPath sets the PATH value in registry
func SetPath(value, scope string) error {
	var target string
	if scope == "System" {
		target = "Machine"
	} else {
		target = "User"
	}
	escaped := strings.ReplaceAll(value, "'", "''")
	command := fmt.Sprintf(`[Environment]::SetEnvironmentVariable('Path', '%s', '%s')`, escaped, target)
	_, err := RunPowerShell(command)
	return err
}

// IsAdmin checks if running with administrator privileges
func IsAdmin() bool {
	command := `([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)`
	result, err := RunPowerShell(command)
	if err != nil {
		return false
	}
	return strings.ToLower(result) == "true"
}

// BroadcastEnvChange notifies Windows of environment variable changes
func BroadcastEnvChange() {
	command := `
		Add-Type -TypeDefinition @"
			using System;
			using System.Runtime.InteropServices;
			public class EnvBroadcast {
				[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)]
				public static extern IntPtr SendMessageTimeout(
					IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam,
					uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
				public static void Broadcast() {
					UIntPtr result;
					SendMessageTimeout((IntPtr)0xFFFF, 0x001A, UIntPtr.Zero, "Environment", 0x0002, 5000, out result);
				}
			}
"@
		[EnvBroadcast]::Broadcast()
	`
	_, _ = RunPowerShell(command) // Best effort broadcast
}

// GetHostname returns the computer name
func GetHostname() string {
	result, err := RunPowerShell("$env:COMPUTERNAME")
	if err != nil {
		return "UNKNOWN"
	}
	return result
}

// GetAllEnvVars returns all environment variables
func GetAllEnvVars() map[string]string {
	vars := make(map[string]string)

	// Get from process environment
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			vars[parts[0]] = parts[1]
		}
	}

	return vars
}

// CopyToClipboard copies text to the Windows clipboard
func CopyToClipboard(text string) error {
	return clipboard.WriteAll(text)
}

// GetRefreshCommand returns the PowerShell command to refresh PATH
func GetRefreshCommand() string {
	return `$env:Path = [Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [Environment]::GetEnvironmentVariable('Path','User')`
}

// ParsePath splits a PATH string into entries
func ParsePath(pathStr string) []string {
	if pathStr == "" {
		return []string{}
	}
	entries := strings.Split(pathStr, ";")
	result := make([]string, 0, len(entries))
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e != "" {
			result = append(result, e)
		}
	}
	return result
}

// JoinPath joins PATH entries into a string
func JoinPath(entries []string) string {
	return strings.Join(entries, ";")
}
