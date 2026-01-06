package main

import (
	"testing"
)

func TestMain_Imports(t *testing.T) {
	// Verify main package compiles and imports work
	// This is a smoke test to catch import errors
	t.Log("Main package compiles successfully")
}

// Note: Testing the actual main() function is tricky because it starts the TUI.
// For comprehensive testing, the logic should be extracted into testable functions.
// The TUI model tests in internal/tui/model_test.go cover the core functionality.
