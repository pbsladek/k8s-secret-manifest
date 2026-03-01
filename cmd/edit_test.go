package cmd

import (
	"os"
	"testing"
)

func TestResolveEditor_FromEnv(t *testing.T) {
	t.Setenv("EDITOR", "sh") // sh is universally available
	got, err := resolveEditor()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == "" {
		t.Error("expected non-empty editor path")
	}
}

func TestResolveEditor_Default(t *testing.T) {
	os.Unsetenv("EDITOR")
	// Empty EDITOR falls back to "vi"; result depends on the test environment.
	got, err := resolveEditor()
	if err == nil && got == "" {
		t.Error("expected non-empty path when no error")
	}
}

func TestResolveEditor_EmptyEnvFallsBack(t *testing.T) {
	t.Setenv("EDITOR", "")
	got, err := resolveEditor()
	if err == nil && got == "" {
		t.Error("expected non-empty path when no error")
	}
}

func TestResolveEditor_InvalidEditor(t *testing.T) {
	t.Setenv("EDITOR", "definitely-not-a-real-editor-binary-xyz")
	_, err := resolveEditor()
	if err == nil {
		t.Error("expected error for non-existent editor binary")
	}
}
