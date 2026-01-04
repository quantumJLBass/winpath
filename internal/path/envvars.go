package path

import (
	"os"
	"strings"
)

// SubstitutionPriority defines the order of variable substitution (most specific first)
var SubstitutionPriority = []string{
	"LOCALAPPDATA",
	"APPDATA",
	"USERPROFILE",
	"ProgramFiles(x86)",
	"ProgramFiles",
	"ProgramW6432",
	"CommonProgramFiles(x86)",
	"CommonProgramFiles",
	"SystemRoot",
	"WINDIR",
	"SystemDrive",
	"JAVA_HOME",
	"GOPATH",
	"GOROOT",
	"CARGO_HOME",
	"RUSTUP_HOME",
	"NVM_HOME",
	"NVM_SYMLINK",
	"PNPM_HOME",
}

// SubstituteEnvVars replaces path prefixes with environment variables where beneficial
func SubstituteEnvVars(path string) (string, bool) {
	if path == "" || strings.Contains(path, "%") {
		return path, false
	}

	// First, normalize the path to handle 8.3 short names in the user folder
	// E.g., C:\Users\JEREMY~1\AppData\Local -> C:\Users\Jeremy\AppData\Local
	normalizedPath := expandShortUserPath(path)
	pathLower := strings.ToLower(normalizedPath)
	
	envVars := GetAllEnvVars()

	var bestMatch struct {
		varName   string
		remaining string
		saved     int
	}

	for _, varName := range SubstitutionPriority {
		value, ok := envVars[varName]
		if !ok || value == "" {
			continue
		}

		valueLower := strings.ToLower(value)
		if strings.HasPrefix(pathLower, valueLower) {
			// Make sure we're at a path boundary
			remaining := normalizedPath[len(value):]
			if remaining == "" || remaining[0] == '\\' || remaining[0] == '/' {
				newPath := "%" + varName + "%" + remaining
				saved := len(path) - len(newPath)
				
				// Keep the match that saves the most characters (most specific)
				if saved > bestMatch.saved {
					bestMatch.varName = varName
					bestMatch.remaining = remaining
					bestMatch.saved = saved
				}
			}
		}
	}

	if bestMatch.varName != "" && bestMatch.saved > 0 {
		return "%" + bestMatch.varName + "%" + bestMatch.remaining, true
	}

	return path, false
}

// expandShortUserPath expands 8.3 short names in the user profile portion of a path
// E.g., C:\Users\JEREMY~1\AppData -> C:\Users\Jeremy\AppData
func expandShortUserPath(path string) string {
	pathLower := strings.ToLower(path)
	systemDrive := strings.ToLower(os.Getenv("SystemDrive"))
	if systemDrive == "" {
		systemDrive = "c:"
	}
	
	usersDir := systemDrive + "\\users\\"
	if !strings.HasPrefix(pathLower, usersDir) {
		return path
	}

	// Extract username portion
	afterUsers := path[len(usersDir):]
	nextSlash := strings.Index(afterUsers, "\\")
	if nextSlash == -1 {
		return path
	}
	
	userPart := afterUsers[:nextSlash]
	
	// Check if it's a short name (contains ~)
	if !strings.Contains(userPart, "~") {
		return path
	}

	// Get the real user profile path
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		return path
	}

	// Extract the real username from USERPROFILE
	userProfileLower := strings.ToLower(userProfile)
	if !strings.HasPrefix(userProfileLower, usersDir) {
		return path
	}
	
	realUserPart := userProfile[len(usersDir):]
	if idx := strings.Index(realUserPart, "\\"); idx != -1 {
		realUserPart = realUserPart[:idx]
	}

	// Replace the short username with the real one
	remaining := afterUsers[nextSlash:]
	return userProfile[:len(usersDir)] + realUserPart + remaining
}

// ExpandEnvVars expands environment variables in a path
func ExpandEnvVars(path string) string {
	if path == "" || !strings.Contains(path, "%") {
		return path
	}

	result := path
	envVars := GetAllEnvVars()

	// Replace %VAR% patterns
	for {
		start := strings.Index(result, "%")
		if start == -1 {
			break
		}
		end := strings.Index(result[start+1:], "%")
		if end == -1 {
			break
		}
		end += start + 1

		varName := result[start+1 : end]
		value, ok := envVars[varName]
		if !ok {
			value = os.Getenv(varName)
		}
		if value != "" {
			result = result[:start] + value + result[end+1:]
		} else {
			// Skip this variable and continue
			break
		}
	}

	return result
}

// GetEnvVariable gets a specific environment variable value
func GetEnvVariable(name, scope string) (string, error) {
	var command string
	if scope == "System" {
		command = `[Microsoft.Win32.Registry]::LocalMachine.OpenSubKey('SYSTEM\CurrentControlSet\Control\Session Manager\Environment')?.GetValue('` + name + `', '', [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)`
	} else {
		command = `[Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Environment')?.GetValue('` + name + `', '', [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)`
	}
	return RunPowerShell(command)
}

// SetEnvVariable sets an environment variable
func SetEnvVariable(name, value, scope string) error {
	target := "User"
	if scope == "System" {
		target = "Machine"
	}
	escaped := strings.ReplaceAll(value, "'", "''")
	command := `[Environment]::SetEnvironmentVariable('` + name + `', '` + escaped + `', '` + target + `')`
	_, err := RunPowerShell(command)
	if err == nil {
		BroadcastEnvChange()
	}
	return err
}
