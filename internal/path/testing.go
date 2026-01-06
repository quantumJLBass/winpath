package path

import "strings"

// SetupTestMocks configures common mock responses for testing
// This is in a regular file (not _test.go) so it can be imported by other packages' tests
func SetupTestMocks(mock *MockShellRunner) {
	// Default response for any unmatched command
	mock.DefaultResponse = ""

	// Registry PATH access - System
	mock.SetResponse("LocalMachine.OpenSubKey", `C:\Windows\System32;C:\Windows;C:\Program Files\Git\bin`)

	// Registry PATH access - User
	mock.SetResponse("CurrentUser.OpenSubKey", `%USERPROFILE%\bin;%LOCALAPPDATA%\Programs\Test`)

	// Expanded paths
	mock.SetResponse("'Path', 'Machine'", `C:\Windows\System32;C:\Windows;C:\Program Files\Git\bin`)
	mock.SetResponse("'Path', 'User'", `C:\Users\Test\bin;C:\Users\Test\AppData\Local\Programs\Test`)

	// Set path (should succeed silently)
	mock.SetResponse("SetEnvironmentVariable", "")

	// IsAdmin check
	mock.SetResponse("IsInRole", "False")

	// Hostname
	mock.SetResponse("$env:COMPUTERNAME", "TESTPC")

	// PATHEXT from registry - full realistic list
	mock.SetResponse("'PATHEXT'", ".COM;.EXE;.BAT;.CMD;.VBS;.VBE;.JS;.JSE;.WSF;.WSH;.MSC;.PY;.PYW")

	// Short path conversions
	mock.SetResponse("Scripting.FileSystemObject", "True")
	mock.SetResponse("ShortPath", "True")
	mock.SetResponse("GetFolder", "")

	// Batch path expansion
	mock.SetResponse("$results -join '|'", `C:\Program Files`)
	mock.SetResponse("foreach ($p in $paths)", `C:\Program Files`)

	// Junction operations
	mock.SetResponse("Get-ChildItem", "")
	mock.SetResponse("New-Item -ItemType Junction", "")
	mock.SetResponse("Remove-Item", "")

	// Test-Path
	mock.SetResponse("Test-Path", "True")

	// Broadcast
	mock.SetResponse("SendMessageTimeout", "")

	// GetEnvVariable responses
	mock.SetResponse("GetEnvironmentVariable('PATH'", `C:\Windows;C:\Windows\System32`)
	mock.SetResponse("GetEnvironmentVariable('PATHEXT'", ".COM;.EXE;.BAT;.CMD;.VBS;.VBE;.JS;.JSE;.WSF;.WSH;.MSC")
}

// SetDefaultTestRunner replaces the default runner with a mock and returns a cleanup function
func SetDefaultTestRunner() (*MockShellRunner, func()) {
	original := DefaultRunner
	mock := NewMockShellRunner()
	SetupTestMocks(mock)
	DefaultRunner = mock
	return mock, func() {
		DefaultRunner = original
	}
}

// IsTestMockActive returns true if the default runner is a mock (safety check)
func IsTestMockActive() bool {
	_, ok := DefaultRunner.(*MockShellRunner)
	return ok
}

// ValidateMockForSystemModification panics if mock isn't active when trying to modify system
// This is a safety net for tests
func ValidateMockForSystemModification(command string) {
	if !IsTestMockActive() {
		// Check if this is a modifying command
		dangerousPatterns := []string{
			"SetEnvironmentVariable",
			"New-Item",
			"Remove-Item",
			"Set-ItemProperty",
		}
		for _, pattern := range dangerousPatterns {
			if strings.Contains(command, pattern) {
				panic("SAFETY: Attempted to modify system without mock! Command: " + command)
			}
		}
	}
}
