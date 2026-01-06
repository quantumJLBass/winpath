package path

import (
	"os/exec"
	"strings"
)

// ShellRunner interface for executing shell commands
// This allows mocking in tests
type ShellRunner interface {
	Run(command string) (string, error)
}

// RealShellRunner executes actual PowerShell commands
type RealShellRunner struct{}

// Run executes a PowerShell command
func (r *RealShellRunner) Run(command string) (string, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", command)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// DefaultRunner is the package-level shell runner
// Tests can replace this with a mock
var DefaultRunner ShellRunner = &RealShellRunner{}

// RunShell executes a command using the default runner
func RunShell(command string) (string, error) {
	return DefaultRunner.Run(command)
}

// MockShellRunner for testing
type MockShellRunner struct {
	Responses       map[string]string
	Errors          map[string]error
	Calls           []string
	DefaultResponse string
}

// NewMockShellRunner creates a new mock runner
func NewMockShellRunner() *MockShellRunner {
	return &MockShellRunner{
		Responses:       make(map[string]string),
		Errors:          make(map[string]error),
		Calls:           []string{},
		DefaultResponse: "",
	}
}

// Run returns mocked responses
func (m *MockShellRunner) Run(command string) (string, error) {
	m.Calls = append(m.Calls, command)

	// Check for exact error match first
	if err, ok := m.Errors[command]; ok {
		return "", err
	}

	// Check for exact response match
	if resp, ok := m.Responses[command]; ok {
		return resp, nil
	}

	// Check for partial matches in errors
	for pattern, err := range m.Errors {
		if strings.Contains(command, pattern) {
			return "", err
		}
	}

	// Check for partial matches in responses
	for pattern, resp := range m.Responses {
		if strings.Contains(command, pattern) {
			return resp, nil
		}
	}

	// Return default response
	return m.DefaultResponse, nil
}

// SetResponse sets a mock response for a command pattern
func (m *MockShellRunner) SetResponse(pattern, response string) {
	m.Responses[pattern] = response
}

// SetError sets a mock error for a command pattern
func (m *MockShellRunner) SetError(pattern string, err error) {
	m.Errors[pattern] = err
}

// Reset clears all mock data
func (m *MockShellRunner) Reset() {
	m.Responses = make(map[string]string)
	m.Errors = make(map[string]error)
	m.Calls = []string{}
}
