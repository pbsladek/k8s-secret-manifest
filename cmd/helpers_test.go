package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// ---- splitKeyValue ----

func TestSplitKeyValue_Simple(t *testing.T) {
	k, v, err := splitKeyValue("API_KEY=mysecret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k != "API_KEY" || v != "mysecret" {
		t.Errorf("got (%q, %q), want (\"API_KEY\", \"mysecret\")", k, v)
	}
}

func TestSplitKeyValue_ValueContainsEquals(t *testing.T) {
	k, v, err := splitKeyValue("TOKEN=abc=def==")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k != "TOKEN" || v != "abc=def==" {
		t.Errorf("got (%q, %q), want (\"TOKEN\", \"abc=def==\")", k, v)
	}
}

func TestSplitKeyValue_EmptyValue(t *testing.T) {
	k, v, err := splitKeyValue("KEY=")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k != "KEY" || v != "" {
		t.Errorf("got (%q, %q), want (\"KEY\", \"\")", k, v)
	}
}

func TestSplitKeyValue_MissingEquals(t *testing.T) {
	_, _, err := splitKeyValue("NOEQUALS")
	if err == nil {
		t.Error("expected error for missing '='")
	}
}

// ---- parseKeyValuePairs ----

func TestParseKeyValuePairs_HappyPath(t *testing.T) {
	m, err := parseKeyValuePairs([]string{"app=myapp", "env=prod"}, "--label")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["app"] != "myapp" || m["env"] != "prod" {
		t.Errorf("unexpected map: %v", m)
	}
}

func TestParseKeyValuePairs_Empty(t *testing.T) {
	m, err := parseKeyValuePairs(nil, "--label")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("want empty map, got %v", m)
	}
}

func TestParseKeyValuePairs_InvalidEntry(t *testing.T) {
	_, err := parseKeyValuePairs([]string{"noequalssign"}, "--label")
	if err == nil {
		t.Error("expected error for invalid entry")
	}
}

func TestParseKeyValuePairs_LastWins(t *testing.T) {
	m, err := parseKeyValuePairs([]string{"k=first", "k=second"}, "--label")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["k"] != "second" {
		t.Errorf("want last value to win, got %q", m["k"])
	}
}

// ---- parseEntryFlags ----

func TestParseEntryFlags_HappyPath(t *testing.T) {
	entries, err := parseEntryFlags([]string{"alice:pass1", "bob:pass2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
	if entries[0].Key != "alice" || entries[0].Value != "pass1" {
		t.Errorf("entries[0] = %+v", entries[0])
	}
}

func TestParseEntryFlags_ValueContainsColon(t *testing.T) {
	entries, err := parseEntryFlags([]string{"url:https://example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entries[0].Value != "https://example.com" {
		t.Errorf("value = %q, want \"https://example.com\"", entries[0].Value)
	}
}

func TestParseEntryFlags_MissingColon(t *testing.T) {
	_, err := parseEntryFlags([]string{"nocolon"})
	if err == nil {
		t.Error("expected error for missing ':'")
	}
}

func TestParseEntryFlags_EmptyKey(t *testing.T) {
	_, err := parseEntryFlags([]string{":value"})
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestParseEntryFlags_DuplicateKey(t *testing.T) {
	_, err := parseEntryFlags([]string{"alice:pass1", "alice:pass2"})
	if err == nil {
		t.Error("expected error for duplicate key")
	}
}

func TestParseEntryFlags_Empty(t *testing.T) {
	entries, err := parseEntryFlags(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("want empty, got %v", entries)
	}
}

// ---- parseEnvFile ----

func writeEnvFile(t *testing.T, content string) string {
	t.Helper()
	tmp := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(tmp, []byte(content), 0600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	return tmp
}

func TestParseEnvFile_Basic(t *testing.T) {
	path := writeEnvFile(t, "API_KEY=mysecret\nDB_HOST=localhost\n")
	pairs, err := parseEnvFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pairs["API_KEY"] != "mysecret" {
		t.Errorf("API_KEY = %q, want \"mysecret\"", pairs["API_KEY"])
	}
	if pairs["DB_HOST"] != "localhost" {
		t.Errorf("DB_HOST = %q, want \"localhost\"", pairs["DB_HOST"])
	}
}

func TestParseEnvFile_SkipsComments(t *testing.T) {
	path := writeEnvFile(t, "# this is a comment\nKEY=value\n")
	pairs, err := parseEnvFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 1 || pairs["KEY"] != "value" {
		t.Errorf("unexpected pairs: %v", pairs)
	}
}

func TestParseEnvFile_SkipsBlankLines(t *testing.T) {
	path := writeEnvFile(t, "\n\nKEY=value\n\n")
	pairs, err := parseEnvFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 1 {
		t.Errorf("want 1 pair, got %d", len(pairs))
	}
}

func TestParseEnvFile_ExportPrefix(t *testing.T) {
	path := writeEnvFile(t, "export KEY=value\n")
	pairs, err := parseEnvFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pairs["KEY"] != "value" {
		t.Errorf("KEY = %q, want \"value\"", pairs["KEY"])
	}
}

func TestParseEnvFile_DoubleQuotes(t *testing.T) {
	path := writeEnvFile(t, `KEY="quoted value"`)
	pairs, err := parseEnvFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pairs["KEY"] != "quoted value" {
		t.Errorf("KEY = %q, want \"quoted value\"", pairs["KEY"])
	}
}

func TestParseEnvFile_SingleQuotes(t *testing.T) {
	path := writeEnvFile(t, "KEY='quoted value'\n")
	pairs, err := parseEnvFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pairs["KEY"] != "quoted value" {
		t.Errorf("KEY = %q, want \"quoted value\"", pairs["KEY"])
	}
}

func TestParseEnvFile_ValueContainsEquals(t *testing.T) {
	path := writeEnvFile(t, "TOKEN=abc=def\n")
	pairs, err := parseEnvFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pairs["TOKEN"] != "abc=def" {
		t.Errorf("TOKEN = %q, want \"abc=def\"", pairs["TOKEN"])
	}
}

func TestParseEnvFile_MissingEquals(t *testing.T) {
	path := writeEnvFile(t, "NOEQUALSSIGN\n")
	_, err := parseEnvFile(path)
	if err == nil {
		t.Error("expected error for line without '='")
	}
}

func TestParseEnvFile_NotFound(t *testing.T) {
	_, err := parseEnvFile("/nonexistent/.env")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// ---- unquote ----

func TestUnquote_DoubleQuotes(t *testing.T) {
	if got := unquote(`"hello"`); got != "hello" {
		t.Errorf("got %q, want \"hello\"", got)
	}
}

func TestUnquote_SingleQuotes(t *testing.T) {
	if got := unquote("'hello'"); got != "hello" {
		t.Errorf("got %q, want \"hello\"", got)
	}
}

func TestUnquote_NoQuotes(t *testing.T) {
	if got := unquote("hello"); got != "hello" {
		t.Errorf("got %q, want \"hello\"", got)
	}
}

func TestUnquote_MismatchedQuotes(t *testing.T) {
	if got := unquote(`"hello'`); got != `"hello'` {
		t.Errorf("mismatched quotes should not be stripped, got %q", got)
	}
}

func TestUnquote_EmptyString(t *testing.T) {
	if got := unquote(""); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestUnquote_OnlyQuotes(t *testing.T) {
	if got := unquote(`""`); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}
