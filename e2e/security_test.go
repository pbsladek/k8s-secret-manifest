//go:build integration

package e2e_test

import (
	"testing"
)

// TestSecurity_PathTraversal verifies that every command rejects file paths
// containing ".." that would escape the current working directory.
func TestSecurity_PathTraversal(t *testing.T) {
	traversalPath := "../../etc/passwd"

	t.Run("SealInput", func(t *testing.T) {
		dir := t.TempDir()
		_, stderr := mustFailDir(t, dir, "seal", "--input", traversalPath)
		assertContains(t, stderr, "escapes current directory")
	})

	t.Run("ValidateInput", func(t *testing.T) {
		dir := t.TempDir()
		_, stderr := mustFailDir(t, dir, "validate", "--input", traversalPath)
		assertContains(t, stderr, "escapes current directory")
	})

	t.Run("ShowInput", func(t *testing.T) {
		dir := t.TempDir()
		_, stderr := mustFailDir(t, dir, "show", "--input", traversalPath)
		assertContains(t, stderr, "escapes current directory")
	})

	t.Run("ListInput", func(t *testing.T) {
		dir := t.TempDir()
		_, stderr := mustFailDir(t, dir, "list", "--input", traversalPath)
		assertContains(t, stderr, "escapes current directory")
	})

	t.Run("ExportEnvInput", func(t *testing.T) {
		dir := t.TempDir()
		_, stderr := mustFailDir(t, dir, "export-env", "--input", traversalPath)
		assertContains(t, stderr, "escapes current directory")
	})

	t.Run("CopyInput", func(t *testing.T) {
		dir := t.TempDir()
		_, stderr := mustFailDir(t, dir, "copy",
			"--input", traversalPath, "--name", "copy")
		assertContains(t, stderr, "escapes current directory")
	})

	t.Run("UpdateInput", func(t *testing.T) {
		dir := t.TempDir()
		_, stderr := mustFailDir(t, dir, "update",
			"--input", traversalPath, "--set", "K=v")
		assertContains(t, stderr, "escapes current directory")
	})

	t.Run("RotateInput", func(t *testing.T) {
		dir := t.TempDir()
		_, stderr := mustFailDir(t, dir, "rotate",
			"--input", traversalPath, "--key", "K")
		assertContains(t, stderr, "escapes current directory")
	})

	t.Run("AddEntryInput", func(t *testing.T) {
		dir := t.TempDir()
		_, stderr := mustFailDir(t, dir, "add-entry",
			"--input", traversalPath,
			"--entries-key", "U", "--entries-val", "P",
			"--key", "k", "--value", "v")
		assertContains(t, stderr, "escapes current directory")
	})

	t.Run("RemoveEntryInput", func(t *testing.T) {
		dir := t.TempDir()
		_, stderr := mustFailDir(t, dir, "remove-entry",
			"--input", traversalPath,
			"--entries-key", "U", "--entries-val", "P",
			"--key", "k")
		assertContains(t, stderr, "escapes current directory")
	})

	t.Run("DiffFrom", func(t *testing.T) {
		dir := t.TempDir()
		_, stderr := mustFailDir(t, dir, "diff",
			"--from", traversalPath, "--to", "b.yaml")
		assertContains(t, stderr, "escapes current directory")
	})

	t.Run("DiffTo", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "s", "K", "v", "a.yaml")
		_, stderr := mustFailDir(t, dir, "diff",
			"--from", "a.yaml", "--to", traversalPath)
		assertContains(t, stderr, "escapes current directory")
	})
}

// TestSecurity_SetFilePathTraversal verifies that --set-file rejects traversal paths.
func TestSecurity_SetFilePathTraversal(t *testing.T) {
	traversalPath := "../../etc/passwd"

	t.Run("GenerateSetFile", func(t *testing.T) {
		dir := t.TempDir()
		_, stderr := mustFailDir(t, dir, "generate",
			"--name", "s", "--set-file", "KEY="+traversalPath)
		assertContains(t, stderr, "escapes current directory")
	})

	t.Run("UpdateSetFile", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "s", "KEY", "val", "secret.yaml")
		_, stderr := mustFailDir(t, dir, "update",
			"--input", "secret.yaml", "--set-file", "KEY="+traversalPath)
		assertContains(t, stderr, "escapes current directory")
	})
}

// TestSecurity_TLSPathTraversal verifies that --tls-cert and --tls-key
// reject traversal paths.
func TestSecurity_TLSPathTraversal(t *testing.T) {
	traversalPath := "../../etc/passwd"

	t.Run("TLSCert", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "key.pem", "KEY")
		_, stderr := mustFailDir(t, dir, "generate",
			"--name", "s", "--tls-cert", traversalPath, "--tls-key", "key.pem")
		assertContains(t, stderr, "escapes current directory")
	})

	t.Run("TLSKey", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "cert.pem", "CERT")
		_, stderr := mustFailDir(t, dir, "generate",
			"--name", "s", "--tls-cert", "cert.pem", "--tls-key", traversalPath)
		assertContains(t, stderr, "escapes current directory")
	})
}

// TestSecurity_OutputPathTraversal verifies that --output rejects traversal paths.
func TestSecurity_OutputPathTraversal(t *testing.T) {
	dir := t.TempDir()
	_, stderr := mustFailDir(t, dir, "generate",
		"--name", "s", "--set", "K=v", "--output", "../../evil.yaml")
	assertContains(t, stderr, "escapes current directory")
}

// TestSecurity_EnvFilePathTraversal verifies that --env-file rejects traversal paths.
func TestSecurity_EnvFilePathTraversal(t *testing.T) {
	dir := t.TempDir()
	_, stderr := mustFailDir(t, dir, "from-env",
		"--name", "s", "--env-file", "../../etc/environment")
	assertContains(t, stderr, "escapes current directory")
}

// TestSecurity_InvalidDataKey verifies that keys with invalid characters are
// rejected by commands that accept --set.
func TestSecurity_InvalidDataKey(t *testing.T) {
	cases := []struct {
		name string
		key  string
	}{
		{"SpaceInKey", "bad key"},
		{"SlashInKey", "bad/key"},
		{"ExclamationInKey", "bad!key"},
		{"ColonInKey", "bad:key"},
		{"EmptyKey", ""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			_, stderr := mustFailDir(t, dir, "generate",
				"--name", "s", "--set", tc.key+"=val")
			// Either "invalid characters" (non-empty key) or format error (empty key).
			if len(stderr) == 0 {
				t.Error("expected error output for invalid key")
			}
		})
	}
}

// TestSecurity_RotateLengthBound confirms the --length cap prevents
// unreasonably large allocations.
func TestSecurity_RotateLengthBound(t *testing.T) {
	dir := t.TempDir()
	generateBasic(t, dir, "s", "SECRET", "x", "secret.yaml")

	_, stderr := mustFailDir(t, dir, "rotate",
		"--input", "secret.yaml",
		"--key", "SECRET",
		"--length", "4097",
	)
	assertContains(t, stderr, "exceeds maximum")
}

// TestSecurity_EnvFileKeyValidation verifies that invalid key names inside a
// .env file are caught when generating a secret from it.
func TestSecurity_EnvFileKeyValidation(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bad.env", "invalid key!=value\n")

	_, stderr := mustFailDir(t, dir, "from-env",
		"--name", "s", "--env-file", "bad.env")
	assertContains(t, stderr, "invalid characters")
}
