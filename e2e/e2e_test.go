//go:build integration

// Package e2e_test contains end-to-end integration tests that compile and
// run the real binary. Run with:
//
//	go test -v -tags integration -count=1 ./e2e/
package e2e_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// binaryPath holds the path to the compiled binary, set once by TestMain.
var binaryPath string

// TestMain compiles the binary into a temporary directory before running any
// tests, then cleans up afterward. This ensures every test runs against the
// same freshly built binary.
func TestMain(m *testing.M) {
	binDir, err := os.MkdirTemp("", "k8s-secret-manifest-e2e-bin-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e setup: create temp dir: %v\n", err)
		os.Exit(1)
	}

	binaryPath = filepath.Join(binDir, "k8s-secret-manifest")

	// Build from the module root (parent of the e2e/ directory).
	build := exec.Command("go", "build", "-o", binaryPath, "..")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "e2e setup: build binary: %v\n", err)
		os.RemoveAll(binDir)
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(binDir)
	os.Exit(code)
}

// ── Execution helpers ────────────────────────────────────────────────────────

// runDir runs the binary with args, with its working directory set to dir.
func runDir(dir string, args ...string) (stdout, stderr string, err error) {
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = dir
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// mustRunDir runs the binary and fails the test immediately if it exits non-zero.
func mustRunDir(t *testing.T, dir string, args ...string) (stdout, stderr string) {
	t.Helper()
	out, errOut, err := runDir(dir, args...)
	if err != nil {
		t.Fatalf("command failed: %v\n  args:   %v\n  stdout: %s\n  stderr: %s",
			err, args, out, errOut)
	}
	return out, errOut
}

// mustFailDir runs the binary and fails the test if the command succeeds.
// Returns (stdout, stderr) of the failed invocation.
func mustFailDir(t *testing.T, dir string, args ...string) (stdout, stderr string) {
	t.Helper()
	out, errOut, err := runDir(dir, args...)
	if err == nil {
		t.Fatalf("command should have failed but succeeded\n  args:   %v\n  stdout: %s\n  stderr: %s",
			args, out, errOut)
	}
	return out, errOut
}

// ── Filesystem helpers ───────────────────────────────────────────────────────

// writeFile writes content to name inside dir and returns its absolute path.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("writeFile %s: %v", name, err)
	}
	return path
}

// readFile reads the contents of name inside dir.
func readFile(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("readFile %s: %v", name, err)
	}
	return string(data)
}

// ── Assertion helpers ────────────────────────────────────────────────────────

// assertContains fails if s does not contain substr.
func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("string does not contain expected substring\n  want: %q\n  in:   %q", substr, s)
	}
}

// assertNotContains fails if s contains substr.
func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Fatalf("string unexpectedly contains substring\n  unwanted: %q\n  in:       %q", substr, s)
	}
}

// assertEqual fails if a != b.
func assertEqual(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("value mismatch\n  got:  %q\n  want: %q", got, want)
	}
}

// ── Domain helpers ───────────────────────────────────────────────────────────

// showKey calls `show --input file --key key` and returns the trimmed plain value.
func showKey(t *testing.T, dir, file, key string) string {
	t.Helper()
	out, _, err := runDir(dir, "show", "--input", file, "--key", key)
	if err != nil {
		t.Fatalf("show --key %s failed: %v", key, err)
	}
	return strings.TrimSpace(out)
}

// generateBasic runs `generate --name name --set key=val --output out` and
// returns the output file name.
func generateBasic(t *testing.T, dir, name, key, val, out string) {
	t.Helper()
	mustRunDir(t, dir, "generate",
		"--name", name,
		"--set", key+"="+val,
		"--output", out,
	)
}
