package cmd

import (
	"os"
	"testing"
)

func TestResolveEditor_FromEnv(t *testing.T) {
	t.Setenv("EDITOR", "nano")
	if got := resolveEditor(); got != "nano" {
		t.Errorf("expected %q, got %q", "nano", got)
	}
}

func TestResolveEditor_Default(t *testing.T) {
	os.Unsetenv("EDITOR")
	if got := resolveEditor(); got != "vi" {
		t.Errorf("expected default %q, got %q", "vi", got)
	}
}

func TestResolveEditor_EmptyEnvFallsBack(t *testing.T) {
	t.Setenv("EDITOR", "")
	if got := resolveEditor(); got != "vi" {
		t.Errorf("empty EDITOR should fall back to vi, got %q", got)
	}
}
