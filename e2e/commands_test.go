//go:build integration

package e2e_test

import (
	"strings"
	"testing"
)

// ── generate ─────────────────────────────────────────────────────────────────

func TestGenerate(t *testing.T) {
	t.Run("BasicSet", func(t *testing.T) {
		dir := t.TempDir()
		mustRunDir(t, dir, "generate", "--name", "my-secret",
			"--set", "API_KEY=abc", "--output", "secret.yaml")

		assertEqual(t, showKey(t, dir, "secret.yaml", "API_KEY"), "abc")
	})

	t.Run("MultipleKeysSortedInYAML", func(t *testing.T) {
		dir := t.TempDir()
		mustRunDir(t, dir, "generate", "--name", "s",
			"--set", "ZZZ=last", "--set", "AAA=first", "--output", "secret.yaml")

		yaml := readFile(t, dir, "secret.yaml")
		posAAA := strings.Index(yaml, "AAA:")
		posZZZ := strings.Index(yaml, "ZZZ:")
		if posAAA < 0 || posZZZ < 0 {
			t.Fatal("both keys should appear in YAML")
		}
		if posAAA > posZZZ {
			t.Error("data keys should be sorted alphabetically (AAA before ZZZ)")
		}
	})

	t.Run("SetFile", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "ca.crt", "CERT_CONTENT")
		mustRunDir(t, dir, "generate", "--name", "s",
			"--set-file", "CERT=ca.crt", "--output", "secret.yaml")

		assertEqual(t, showKey(t, dir, "secret.yaml", "CERT"), "CERT_CONTENT")
	})

	t.Run("TLSType", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "tls.crt", "CERT_DATA")
		writeFile(t, dir, "tls.key", "KEY_DATA")
		mustRunDir(t, dir, "generate", "--name", "tls-secret",
			"--tls-cert", "tls.crt", "--tls-key", "tls.key", "--output", "secret.yaml")

		yaml := readFile(t, dir, "secret.yaml")
		assertContains(t, yaml, "kubernetes.io/tls")
		assertEqual(t, showKey(t, dir, "secret.yaml", "tls.crt"), "CERT_DATA")
		assertEqual(t, showKey(t, dir, "secret.yaml", "tls.key"), "KEY_DATA")
	})

	t.Run("DockerRegistryType", func(t *testing.T) {
		dir := t.TempDir()
		mustRunDir(t, dir, "generate", "--name", "reg",
			"--docker-server", "ghcr.io",
			"--docker-username", "user",
			"--docker-password", "token",
			"--output", "secret.yaml")

		yaml := readFile(t, dir, "secret.yaml")
		assertContains(t, yaml, "kubernetes.io/dockerconfigjson")
		assertContains(t, yaml, ".dockerconfigjson")
	})

	t.Run("PairedEntries", func(t *testing.T) {
		dir := t.TempDir()
		mustRunDir(t, dir, "generate", "--name", "s",
			"--entries-key", "USERS", "--entries-val", "PASSWORDS",
			"--entry", "alice:pass1", "--entry", "bob:pass2",
			"--output", "secret.yaml")

		assertEqual(t, showKey(t, dir, "secret.yaml", "USERS"), "alice;bob")
		assertEqual(t, showKey(t, dir, "secret.yaml", "PASSWORDS"), "pass1;pass2")
	})

	t.Run("OutputToStdout", func(t *testing.T) {
		dir := t.TempDir()
		out, _ := mustRunDir(t, dir, "generate", "--name", "s", "--set", "K=v")

		assertContains(t, out, "apiVersion: v1")
		assertContains(t, out, "kind: Secret")
	})

	t.Run("InvalidKeyName", func(t *testing.T) {
		dir := t.TempDir()
		_, stderr := mustFailDir(t, dir, "generate", "--name", "s",
			"--set", "invalid key!=val")
		assertContains(t, stderr, "invalid characters")
	})

	t.Run("TLSMissingOneFile", func(t *testing.T) {
		dir := t.TempDir()
		_, stderr := mustFailDir(t, dir, "generate", "--name", "s",
			"--tls-cert", "tls.crt")
		assertContains(t, stderr, "--tls-cert and --tls-key must both be provided")
	})
}

// ── from-env ─────────────────────────────────────────────────────────────────

func TestFromEnv(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, ".env", "API_KEY=abc\nDB_PASS=xyz\n")
		mustRunDir(t, dir, "from-env", "--name", "s",
			"--env-file", ".env", "--output", "secret.yaml")

		assertEqual(t, showKey(t, dir, "secret.yaml", "API_KEY"), "abc")
		assertEqual(t, showKey(t, dir, "secret.yaml", "DB_PASS"), "xyz")
	})

	t.Run("CommentsAndBlanksIgnored", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, ".env", "# comment\n\nKEY=val\n")
		mustRunDir(t, dir, "from-env", "--name", "s",
			"--env-file", ".env", "--output", "secret.yaml")

		assertEqual(t, showKey(t, dir, "secret.yaml", "KEY"), "val")
	})

	t.Run("SetOverridesEnvFile", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, ".env", "KEY=original\n")
		mustRunDir(t, dir, "from-env", "--name", "s",
			"--env-file", ".env", "--set", "KEY=override", "--output", "secret.yaml")

		assertEqual(t, showKey(t, dir, "secret.yaml", "KEY"), "override")
	})

	t.Run("QuotedValues", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, ".env", `KEY="quoted value"`)
		mustRunDir(t, dir, "from-env", "--name", "s",
			"--env-file", ".env", "--output", "secret.yaml")

		assertEqual(t, showKey(t, dir, "secret.yaml", "KEY"), "quoted value")
	})
}

// ── export-env ────────────────────────────────────────────────────────────────

func TestExportEnv(t *testing.T) {
	t.Run("BasicOutput", func(t *testing.T) {
		dir := t.TempDir()
		mustRunDir(t, dir, "generate", "--name", "s",
			"--set", "API_KEY=abc", "--set", "DB_PASS=xyz", "--output", "secret.yaml")
		mustRunDir(t, dir, "export-env", "--input", "secret.yaml", "--output", ".env")

		env := readFile(t, dir, ".env")
		assertContains(t, env, "API_KEY=abc")
		assertContains(t, env, "DB_PASS=xyz")
	})

	t.Run("SpecialCharsQuoted", func(t *testing.T) {
		dir := t.TempDir()
		mustRunDir(t, dir, "generate", "--name", "s",
			"--set", "KEY=value with spaces", "--output", "secret.yaml")
		out, _ := mustRunDir(t, dir, "export-env", "--input", "secret.yaml")

		assertContains(t, out, `"value with spaces"`)
	})
}

// ── update ────────────────────────────────────────────────────────────────────

func TestUpdate(t *testing.T) {
	t.Run("OverwriteKey", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "s", "KEY", "original", "secret.yaml")
		mustRunDir(t, dir, "update", "--input", "secret.yaml", "--set", "KEY=updated")

		assertEqual(t, showKey(t, dir, "secret.yaml", "KEY"), "updated")
	})

	t.Run("AddNewKey", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "s", "KEY", "val", "secret.yaml")
		mustRunDir(t, dir, "update", "--input", "secret.yaml", "--set", "NEW=added")

		assertEqual(t, showKey(t, dir, "secret.yaml", "KEY"), "val")
		assertEqual(t, showKey(t, dir, "secret.yaml", "NEW"), "added")
	})

	t.Run("DeleteKey", func(t *testing.T) {
		dir := t.TempDir()
		mustRunDir(t, dir, "generate", "--name", "s",
			"--set", "KEEP=yes", "--set", "DELETE=me", "--output", "secret.yaml")
		mustRunDir(t, dir, "update", "--input", "secret.yaml", "--delete-key", "DELETE")

		// KEEP should still exist
		assertEqual(t, showKey(t, dir, "secret.yaml", "KEEP"), "yes")
		// DELETE should be gone
		yaml := readFile(t, dir, "secret.yaml")
		assertNotContains(t, yaml, "DELETE")
	})

	t.Run("SetFileContent", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "s", "KEY", "val", "secret.yaml")
		writeFile(t, dir, "cert.pem", "PEM_CONTENT")
		mustRunDir(t, dir, "update", "--input", "secret.yaml",
			"--set-file", "CERT=cert.pem")

		assertEqual(t, showKey(t, dir, "secret.yaml", "CERT"), "PEM_CONTENT")
	})

	t.Run("InvalidKeyName", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "s", "KEY", "val", "secret.yaml")
		_, stderr := mustFailDir(t, dir, "update", "--input", "secret.yaml",
			"--set", "bad key!=x")
		assertContains(t, stderr, "invalid characters")
	})

	t.Run("DeleteMissingKeyErrors", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "s", "KEY", "val", "secret.yaml")
		_, stderr := mustFailDir(t, dir, "update", "--input", "secret.yaml",
			"--delete-key", "MISSING")
		assertContains(t, stderr, "not found")
	})
}

// ── rotate ────────────────────────────────────────────────────────────────────

func TestRotate(t *testing.T) {
	t.Run("ValueChanges", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "s", "SECRET", "original", "secret.yaml")
		mustRunDir(t, dir, "rotate", "--input", "secret.yaml", "--key", "SECRET")

		newVal := showKey(t, dir, "secret.yaml", "SECRET")
		if newVal == "original" {
			t.Error("rotated value should differ from original")
		}
		if len(newVal) != 32 { // default length
			t.Errorf("expected default length 32, got %d", len(newVal))
		}
	})

	t.Run("CustomLength", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "s", "SECRET", "x", "secret.yaml")
		mustRunDir(t, dir, "rotate", "--input", "secret.yaml",
			"--key", "SECRET", "--length", "16")

		val := showKey(t, dir, "secret.yaml", "SECRET")
		if len(val) != 16 {
			t.Errorf("expected rotated value length 16, got %d (%q)", len(val), val)
		}
	})

	t.Run("HexCharset", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "s", "SECRET", "x", "secret.yaml")
		mustRunDir(t, dir, "rotate", "--input", "secret.yaml",
			"--key", "SECRET", "--charset", "hex")

		val := showKey(t, dir, "secret.yaml", "SECRET")
		for _, c := range val {
			if !strings.ContainsRune("0123456789abcdef", c) {
				t.Errorf("hex value contains non-hex char %q: %s", c, val)
				break
			}
		}
	})

	t.Run("ExceedsMaxLength", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "s", "SECRET", "x", "secret.yaml")
		_, stderr := mustFailDir(t, dir, "rotate", "--input", "secret.yaml",
			"--key", "SECRET", "--length", "5000")
		assertContains(t, stderr, "exceeds maximum")
	})

	t.Run("KeyNotFound", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "s", "KEY", "val", "secret.yaml")
		_, stderr := mustFailDir(t, dir, "rotate", "--input", "secret.yaml",
			"--key", "MISSING")
		assertContains(t, stderr, "not found")
	})

	t.Run("PrintsNewValueToStderr", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "s", "SECRET", "x", "secret.yaml")
		_, stderr := mustRunDir(t, dir, "rotate", "--input", "secret.yaml",
			"--key", "SECRET")
		assertContains(t, stderr, "SECRET=")
	})
}

// ── add-entry / remove-entry ──────────────────────────────────────────────────

func TestAddEntry(t *testing.T) {
	t.Run("AppendToEmpty", func(t *testing.T) {
		dir := t.TempDir()
		mustRunDir(t, dir, "generate", "--name", "s",
			"--entries-key", "USERS", "--entries-val", "PASSES",
			"--entry", "alice:pass1",
			"--output", "secret.yaml")
		mustRunDir(t, dir, "add-entry", "--input", "secret.yaml",
			"--entries-key", "USERS", "--entries-val", "PASSES",
			"--key", "bob", "--value", "pass2")

		assertEqual(t, showKey(t, dir, "secret.yaml", "USERS"), "alice;bob")
		assertEqual(t, showKey(t, dir, "secret.yaml", "PASSES"), "pass1;pass2")
	})

	t.Run("InsertAtPosition", func(t *testing.T) {
		dir := t.TempDir()
		mustRunDir(t, dir, "generate", "--name", "s",
			"--entries-key", "USERS", "--entries-val", "PASSES",
			"--entry", "alice:pass1", "--entry", "charlie:pass3",
			"--output", "secret.yaml")
		mustRunDir(t, dir, "add-entry", "--input", "secret.yaml",
			"--entries-key", "USERS", "--entries-val", "PASSES",
			"--key", "bob", "--value", "pass2", "--index", "1")

		assertEqual(t, showKey(t, dir, "secret.yaml", "USERS"), "alice;bob;charlie")
	})

	t.Run("DuplicateKeyErrors", func(t *testing.T) {
		dir := t.TempDir()
		mustRunDir(t, dir, "generate", "--name", "s",
			"--entries-key", "USERS", "--entries-val", "PASSES",
			"--entry", "alice:pass1",
			"--output", "secret.yaml")
		_, stderr := mustFailDir(t, dir, "add-entry", "--input", "secret.yaml",
			"--entries-key", "USERS", "--entries-val", "PASSES",
			"--key", "alice", "--value", "other")
		assertContains(t, stderr, "alice")
	})
}

func TestRemoveEntry(t *testing.T) {
	setup := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		mustRunDir(t, dir, "generate", "--name", "s",
			"--entries-key", "USERS", "--entries-val", "PASSES",
			"--entry", "alice:pass1", "--entry", "bob:pass2",
			"--output", "secret.yaml")
		return dir
	}

	t.Run("ByKey", func(t *testing.T) {
		dir := setup(t)
		mustRunDir(t, dir, "remove-entry", "--input", "secret.yaml",
			"--entries-key", "USERS", "--entries-val", "PASSES",
			"--key", "alice")

		assertEqual(t, showKey(t, dir, "secret.yaml", "USERS"), "bob")
		assertEqual(t, showKey(t, dir, "secret.yaml", "PASSES"), "pass2")
	})

	t.Run("ByValue", func(t *testing.T) {
		dir := setup(t)
		mustRunDir(t, dir, "remove-entry", "--input", "secret.yaml",
			"--entries-key", "USERS", "--entries-val", "PASSES",
			"--value", "pass1")

		assertEqual(t, showKey(t, dir, "secret.yaml", "USERS"), "bob")
	})

	t.Run("KeyAndValueMutuallyExclusive", func(t *testing.T) {
		dir := setup(t)
		_, stderr := mustFailDir(t, dir, "remove-entry", "--input", "secret.yaml",
			"--entries-key", "USERS", "--entries-val", "PASSES",
			"--key", "alice", "--value", "pass1")
		assertContains(t, stderr, "mutually exclusive")
	})
}

// ── show / list ───────────────────────────────────────────────────────────────

func TestShow(t *testing.T) {
	t.Run("SingleKey", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "s", "API_KEY", "secret-value", "secret.yaml")
		out, _ := mustRunDir(t, dir, "show", "--input", "secret.yaml",
			"--key", "API_KEY")
		assertEqual(t, strings.TrimSpace(out), "secret-value")
	})

	t.Run("AllKeys", func(t *testing.T) {
		dir := t.TempDir()
		mustRunDir(t, dir, "generate", "--name", "s",
			"--set", "AAA=1", "--set", "BBB=2", "--output", "secret.yaml")
		out, _ := mustRunDir(t, dir, "show", "--input", "secret.yaml")

		assertContains(t, out, "AAA: 1")
		assertContains(t, out, "BBB: 2")
	})

	t.Run("MissingKeyErrors", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "s", "KEY", "val", "secret.yaml")
		_, stderr := mustFailDir(t, dir, "show", "--input", "secret.yaml",
			"--key", "MISSING")
		assertContains(t, stderr, "MISSING")
	})
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	mustRunDir(t, dir, "generate", "--name", "my-secret",
		"--set", "API_KEY=x", "--set", "DB_PASS=y", "--output", "secret.yaml")
	out, _ := mustRunDir(t, dir, "list", "--input", "secret.yaml")

	assertContains(t, out, "my-secret")
	assertContains(t, out, "API_KEY")
	assertContains(t, out, "DB_PASS")
}

// ── diff ──────────────────────────────────────────────────────────────────────

func TestDiff(t *testing.T) {
	t.Run("NoChanges", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "s", "KEY", "val", "a.yaml")
		generateBasic(t, dir, "s", "KEY", "val", "b.yaml")
		out, _ := mustRunDir(t, dir, "diff", "--from", "a.yaml", "--to", "b.yaml")
		assertContains(t, out, "no differences")
	})

	t.Run("AddedKey", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "s", "KEY", "val", "a.yaml")
		mustRunDir(t, dir, "generate", "--name", "s",
			"--set", "KEY=val", "--set", "NEW=added", "--output", "b.yaml")
		out, _ := mustRunDir(t, dir, "diff", "--from", "a.yaml", "--to", "b.yaml")
		assertContains(t, out, "+ NEW=added")
	})

	t.Run("ChangedValue", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "s", "KEY", "old", "a.yaml")
		generateBasic(t, dir, "s", "KEY", "new", "b.yaml")
		out, _ := mustRunDir(t, dir, "diff", "--from", "a.yaml", "--to", "b.yaml")
		assertContains(t, out, "- KEY=old")
		assertContains(t, out, "+ KEY=new")
	})

	t.Run("RemovedKey", func(t *testing.T) {
		dir := t.TempDir()
		mustRunDir(t, dir, "generate", "--name", "s",
			"--set", "KEY=val", "--set", "OLD=removed", "--output", "a.yaml")
		generateBasic(t, dir, "s", "KEY", "val", "b.yaml")
		out, _ := mustRunDir(t, dir, "diff", "--from", "a.yaml", "--to", "b.yaml")
		assertContains(t, out, "- OLD=removed")
	})
}

// ── copy ──────────────────────────────────────────────────────────────────────

func TestCopy(t *testing.T) {
	t.Run("NewName", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "original", "KEY", "val", "a.yaml")
		mustRunDir(t, dir, "copy", "--input", "a.yaml",
			"--name", "renamed", "--output", "b.yaml")

		yaml := readFile(t, dir, "b.yaml")
		assertContains(t, yaml, "name: renamed")
		assertEqual(t, showKey(t, dir, "b.yaml", "KEY"), "val")
	})

	t.Run("NewNamespace", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "s", "KEY", "val", "a.yaml")
		mustRunDir(t, dir, "copy", "--input", "a.yaml",
			"--name", "s", "--namespace", "production", "--output", "b.yaml")

		yaml := readFile(t, dir, "b.yaml")
		assertContains(t, yaml, "namespace: production")
	})

	t.Run("DataPreserved", func(t *testing.T) {
		dir := t.TempDir()
		mustRunDir(t, dir, "generate", "--name", "s",
			"--set", "A=1", "--set", "B=2", "--output", "a.yaml")
		mustRunDir(t, dir, "copy", "--input", "a.yaml",
			"--name", "s-copy", "--output", "b.yaml")

		assertEqual(t, showKey(t, dir, "b.yaml", "A"), "1")
		assertEqual(t, showKey(t, dir, "b.yaml", "B"), "2")
	})
}

// ── validate ──────────────────────────────────────────────────────────────────

func TestValidate(t *testing.T) {
	t.Run("ValidSecret", func(t *testing.T) {
		dir := t.TempDir()
		generateBasic(t, dir, "my-secret", "KEY", "val", "secret.yaml")
		mustRunDir(t, dir, "validate", "--input", "secret.yaml")
	})

	t.Run("MissingTLSKeys", func(t *testing.T) {
		dir := t.TempDir()
		// Generate a TLS-typed secret missing the required keys.
		mustRunDir(t, dir, "generate", "--name", "tls-bad",
			"--type", "kubernetes.io/tls",
			"--set", "OTHER=val", "--output", "secret.yaml")
		_, stderr := mustFailDir(t, dir, "validate", "--input", "secret.yaml")
		assertContains(t, stderr, "tls.crt")
	})

	t.Run("EmptyDataWarning", func(t *testing.T) {
		dir := t.TempDir()
		// An Opaque secret with no data keys produces a warning but exits 0.
		mustRunDir(t, dir, "generate", "--name", "empty", "--output", "secret.yaml")
		_, stderr := mustRunDir(t, dir, "validate", "--input", "secret.yaml")
		assertContains(t, stderr, "warning")
	})
}

// ── seal (no kubeseal binary required) ───────────────────────────────────────

func TestSeal_NoBinary(t *testing.T) {
	dir := t.TempDir()
	generateBasic(t, dir, "s", "KEY", "val", "secret.yaml")

	// Point --kubeseal-path at a name that definitely doesn't exist.
	_, stderr := mustFailDir(t, dir, "--kubeseal-path", "kubeseal-does-not-exist",
		"seal", "--input", "secret.yaml")
	assertContains(t, stderr, "kubeseal-does-not-exist")
}
