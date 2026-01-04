package path

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
)

const (
	SystemPathKey = `HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Environment`
	UserPathKey   = `HKCU:\Environment`
)

// RunPowerShell executes a PowerShell command and returns the output
func RunPowerShell(command string) (string, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", command)
	cmd.SysProcAttr = nil // Let it inherit, but we capture output
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
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
		command = `[Environment]::GetEnvironmentVariable('Path', 'Machine')`
	} else {
		command = `[Environment]::GetEnvironmentVariable('Path', 'User')`
	}
	return RunPowerShell(command)
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
	RunPowerShell(command)
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
