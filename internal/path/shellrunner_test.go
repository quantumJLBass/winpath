package path

import (
	"errors"
	"testing"
)

func TestShellRunnerInterface(t *testing.T) {
	// Verify RealShellRunner implements ShellRunner interface
	// This is a compile-time check - if it compiles, the interface is implemented
	var _ ShellRunner = (*RealShellRunner)(nil)

	// Test that we can create a real runner (existence test)
	runner := &RealShellRunner{}
	_ = runner // Use runner to avoid unused variable warning
}

func TestMockShellRunner_NewMock(t *testing.T) {
	mock := NewMockShellRunner()

	if mock == nil {
		t.Fatal("NewMockShellRunner returned nil")
	}
	if mock.Responses == nil {
		t.Error("Responses map should not be nil")
	}
	if mock.Errors == nil {
		t.Error("Errors map should not be nil")
	}
	if mock.Calls == nil {
		t.Error("Calls slice should not be nil")
	}
}

func TestMockShellRunner_SetResponse(t *testing.T) {
	mock := NewMockShellRunner()
	mock.SetResponse("test-cmd", "test-response")

	result, err := mock.Run("test-cmd")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "test-response" {
		t.Errorf("Expected 'test-response', got '%s'", result)
	}
}

func TestMockShellRunner_SetError(t *testing.T) {
	mock := NewMockShellRunner()
	expectedErr := errors.New("test error")
	mock.SetError("error-cmd", expectedErr)

	_, err := mock.Run("error-cmd")
	if err != expectedErr {
		t.Errorf("Expected error '%v', got '%v'", expectedErr, err)
	}
}

func TestMockShellRunner_PartialMatch(t *testing.T) {
	mock := NewMockShellRunner()
	mock.SetResponse("partial", "matched")

	result, err := mock.Run("this contains partial in the middle")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "matched" {
		t.Errorf("Expected 'matched', got '%s'", result)
	}
}

func TestMockShellRunner_DefaultResponse(t *testing.T) {
	mock := NewMockShellRunner()
	mock.DefaultResponse = "default"

	result, err := mock.Run("unmatched-command")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "default" {
		t.Errorf("Expected 'default', got '%s'", result)
	}
}

func TestMockShellRunner_TracksCalls(t *testing.T) {
	mock := NewMockShellRunner()

	mock.Run("cmd1")
	mock.Run("cmd2")
	mock.Run("cmd3")

	if len(mock.Calls) != 3 {
		t.Errorf("Expected 3 calls, got %d", len(mock.Calls))
	}
	if mock.Calls[0] != "cmd1" || mock.Calls[1] != "cmd2" || mock.Calls[2] != "cmd3" {
		t.Error("Calls not tracked correctly")
	}
}

func TestMockShellRunner_Reset(t *testing.T) {
	mock := NewMockShellRunner()
	mock.SetResponse("test", "value")
	mock.SetError("error", errors.New("err"))
	mock.Run("tracked")

	mock.Reset()

	if len(mock.Responses) != 0 {
		t.Error("Responses should be empty after reset")
	}
	if len(mock.Errors) != 0 {
		t.Error("Errors should be empty after reset")
	}
	if len(mock.Calls) != 0 {
		t.Error("Calls should be empty after reset")
	}
}

func TestMockShellRunner_ErrorPriority(t *testing.T) {
	mock := NewMockShellRunner()
	mock.SetResponse("cmd", "response")
	mock.SetError("cmd", errors.New("error"))

	_, err := mock.Run("cmd")
	if err == nil {
		t.Error("Error should take priority over response")
	}
}

func TestMockShellRunner_PartialErrorMatch(t *testing.T) {
	mock := NewMockShellRunner()
	mock.SetError("partial-error", errors.New("partial error"))

	_, err := mock.Run("this contains partial-error pattern")
	if err == nil {
		t.Error("Partial error match should return error")
	}
}

func TestRunShell(t *testing.T) {
	// RunShell uses DefaultRunner
	result, err := RunShell("test")
	if err != nil {
		t.Logf("RunShell error: %v", err)
	}
	t.Logf("RunShell result: %s", result)
}

func TestDefaultRunner_NotNil(t *testing.T) {
	if DefaultRunner == nil {
		t.Error("DefaultRunner should not be nil")
	}
}

func BenchmarkMockShellRunner_Run(b *testing.B) {
	mock := NewMockShellRunner()
	mock.SetResponse("bench", "result")

	for i := 0; i < b.N; i++ {
		mock.Run("bench")
	}
}

func BenchmarkMockShellRunner_PartialMatch(b *testing.B) {
	mock := NewMockShellRunner()
	mock.SetResponse("partial", "result")

	for i := 0; i < b.N; i++ {
		mock.Run("this is a longer command with partial in it")
	}
}
