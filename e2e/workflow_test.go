//go:build integration

package e2e_test

import (
	"strings"
	"testing"
)

// TestWorkflow_CreateAndValidate checks the most common usage path:
// generate a secret, then confirm it passes validation.
func TestWorkflow_CreateAndValidate(t *testing.T) {
	dir := t.TempDir()
	mustRunDir(t, dir, "generate",
		"--name", "app-secrets",
		"--namespace", "production",
		"--set", "API_KEY=abc",
		"--set", "DB_PASSWORD=s3cr3t",
		"--output", "secret.yaml",
	)

	mustRunDir(t, dir, "validate", "--input", "secret.yaml")
}

// TestWorkflow_UpdatePreservesUntouchedKeys confirms that updating one key
// does not affect the values of other keys in the same file.
func TestWorkflow_UpdatePreservesUntouchedKeys(t *testing.T) {
	dir := t.TempDir()
	mustRunDir(t, dir, "generate",
		"--name", "s",
		"--set", "KEEP=untouched",
		"--set", "CHANGE=old",
		"--output", "secret.yaml",
	)

	mustRunDir(t, dir, "update",
		"--input", "secret.yaml",
		"--set", "CHANGE=new",
	)

	assertEqual(t, showKey(t, dir, "secret.yaml", "KEEP"), "untouched")
	assertEqual(t, showKey(t, dir, "secret.yaml", "CHANGE"), "new")
}

// TestWorkflow_ExportImportRoundTrip generates a secret, exports it as .env,
// re-imports it, and asserts that all values survive the round-trip unchanged.
func TestWorkflow_ExportImportRoundTrip(t *testing.T) {
	dir := t.TempDir()
	mustRunDir(t, dir, "generate",
		"--name", "s",
		"--set", "API_KEY=mykey",
		"--set", "DB_HOST=localhost",
		"--set", "PORT=5432",
		"--output", "original.yaml",
	)

	mustRunDir(t, dir, "export-env",
		"--input", "original.yaml",
		"--output", "exported.env",
	)

	mustRunDir(t, dir, "from-env",
		"--name", "s",
		"--env-file", "exported.env",
		"--output", "reimported.yaml",
	)

	for _, tc := range []struct{ key, want string }{
		{"API_KEY", "mykey"},
		{"DB_HOST", "localhost"},
		{"PORT", "5432"},
	} {
		got := showKey(t, dir, "reimported.yaml", tc.key)
		if got != tc.want {
			t.Errorf("round-trip key %s: got %q, want %q", tc.key, got, tc.want)
		}
	}
}

// TestWorkflow_CopyDiff copies a secret and confirms the diff shows no
// differences when the name and namespace are preserved.
func TestWorkflow_CopyDiff(t *testing.T) {
	dir := t.TempDir()
	mustRunDir(t, dir, "generate",
		"--name", "original",
		"--namespace", "default",
		"--set", "KEY=value",
		"--output", "a.yaml",
	)

	mustRunDir(t, dir, "copy",
		"--input", "a.yaml",
		"--name", "original",
		"--namespace", "default",
		"--output", "b.yaml",
	)

	out, _ := mustRunDir(t, dir, "diff", "--from", "a.yaml", "--to", "b.yaml")
	assertContains(t, out, "no differences")
}

// TestWorkflow_EntryManagement exercises the full entry lifecycle:
// generate with entries → add more entries → remove one → verify final state.
func TestWorkflow_EntryManagement(t *testing.T) {
	dir := t.TempDir()
	mustRunDir(t, dir, "generate",
		"--name", "pgpool",
		"--entries-key", "USERS",
		"--entries-val", "PASSWORDS",
		"--entry", "alice:pass1",
		"--output", "secret.yaml",
	)

	mustRunDir(t, dir, "add-entry",
		"--input", "secret.yaml",
		"--entries-key", "USERS",
		"--entries-val", "PASSWORDS",
		"--key", "bob",
		"--value", "pass2",
	)
	mustRunDir(t, dir, "add-entry",
		"--input", "secret.yaml",
		"--entries-key", "USERS",
		"--entries-val", "PASSWORDS",
		"--key", "carol",
		"--value", "pass3",
	)

	assertEqual(t, showKey(t, dir, "secret.yaml", "USERS"), "alice;bob;carol")

	mustRunDir(t, dir, "remove-entry",
		"--input", "secret.yaml",
		"--entries-key", "USERS",
		"--entries-val", "PASSWORDS",
		"--key", "bob",
	)

	assertEqual(t, showKey(t, dir, "secret.yaml", "USERS"), "alice;carol")
	assertEqual(t, showKey(t, dir, "secret.yaml", "PASSWORDS"), "pass1;pass3")
}

// TestWorkflow_RotateAndVerify rotates a key and confirms that the new value
// differs from the original and is usable by subsequent commands.
func TestWorkflow_RotateAndVerify(t *testing.T) {
	dir := t.TempDir()
	generateBasic(t, dir, "s", "SECRET", "original-value", "secret.yaml")

	mustRunDir(t, dir, "rotate",
		"--input", "secret.yaml",
		"--key", "SECRET",
		"--length", "24",
	)

	newVal := showKey(t, dir, "secret.yaml", "SECRET")
	if newVal == "original-value" {
		t.Error("rotated value should differ from original")
	}
	if len(newVal) != 24 {
		t.Errorf("expected length 24, got %d", len(newVal))
	}

	// The file is still a valid secret after rotation.
	mustRunDir(t, dir, "validate", "--input", "secret.yaml")
}

// TestWorkflow_MultiStepUpdate chains several update operations on the same
// file and verifies the final state reflects all operations.
func TestWorkflow_MultiStepUpdate(t *testing.T) {
	dir := t.TempDir()
	mustRunDir(t, dir, "generate",
		"--name", "s",
		"--set", "A=1", "--set", "B=2", "--set", "C=3",
		"--output", "secret.yaml",
	)

	// Update A, delete B, add D.
	mustRunDir(t, dir, "update", "--input", "secret.yaml",
		"--set", "A=updated",
		"--delete-key", "B",
		"--set", "D=new",
	)

	assertEqual(t, showKey(t, dir, "secret.yaml", "A"), "updated")
	assertEqual(t, showKey(t, dir, "secret.yaml", "C"), "3")
	assertEqual(t, showKey(t, dir, "secret.yaml", "D"), "new")

	yaml := readFile(t, dir, "secret.yaml")
	assertNotContains(t, yaml, ": B") // B should be absent
}

// TestWorkflow_GenerateTLSValidate generates a TLS secret and confirms
// validate accepts it as a valid kubernetes.io/tls secret.
func TestWorkflow_GenerateTLSValidate(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "tls.crt", "CERT")
	writeFile(t, dir, "tls.key", "KEY")

	mustRunDir(t, dir, "generate",
		"--name", "tls-secret",
		"--tls-cert", "tls.crt",
		"--tls-key", "tls.key",
		"--output", "secret.yaml",
	)

	// Validation should pass (no errors).
	_, stderr := mustRunDir(t, dir, "validate", "--input", "secret.yaml")
	assertNotContains(t, strings.ToLower(stderr), "error")
}

// TestWorkflow_ShowAfterExport checks that export-env and show agree on values.
func TestWorkflow_ShowAfterExport(t *testing.T) {
	dir := t.TempDir()
	mustRunDir(t, dir, "generate",
		"--name", "s",
		"--set", "MY_KEY=hello-world",
		"--output", "secret.yaml",
	)

	// Verify via show --key
	assertEqual(t, showKey(t, dir, "secret.yaml", "MY_KEY"), "hello-world")

	// Verify via export-env output
	out, _ := mustRunDir(t, dir, "export-env", "--input", "secret.yaml")
	assertContains(t, out, "MY_KEY=hello-world")
}
