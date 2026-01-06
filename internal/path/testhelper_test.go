package path

import (
	"os"
	"testing"
)

// TestMain sets up mock runner for all tests
func TestMain(m *testing.M) {
	// Use temp directory for config to avoid polluting real config
	tempDir, err := os.MkdirTemp("", "syspath-test-*")
	if err != nil {
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)
	SetConfigDir(tempDir)

	// Replace default runner with mock for all tests
	mock, cleanup := SetDefaultTestRunner()
	_ = mock // available if tests need to modify it

	// Run tests
	code := m.Run()

	// Restore original runner
	cleanup()

	// Exit with test result code
	os.Exit(code)
}

// withMockRunner runs a test with a custom mock configuration
func withMockRunner(t *testing.T, setup func(*MockShellRunner), test func()) {
	t.Helper()

	mock, ok := DefaultRunner.(*MockShellRunner)
	if !ok {
		t.Skip("Not running with mock runner")
		return
	}

	// Save current state
	oldResponses := make(map[string]string)
	for k, v := range mock.Responses {
		oldResponses[k] = v
	}
	oldErrors := make(map[string]error)
	for k, v := range mock.Errors {
		oldErrors[k] = v
	}
	oldDefault := mock.DefaultResponse

	// Apply custom setup
	if setup != nil {
		setup(mock)
	}

	// Run test
	test()

	// Restore state
	mock.Responses = oldResponses
	mock.Errors = oldErrors
	mock.DefaultResponse = oldDefault
}

// getMockRunner returns the mock runner for custom assertions
func getMockRunner(t *testing.T) *MockShellRunner {
	t.Helper()
	mock, ok := DefaultRunner.(*MockShellRunner)
	if !ok {
		t.Fatal("Not running with mock runner")
		return nil
	}
	return mock
}
