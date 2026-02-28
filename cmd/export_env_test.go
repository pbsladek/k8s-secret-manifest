package cmd

import "testing"

// ---- quoteEnvValue ----

func TestQuoteEnvValue_PlainValue(t *testing.T) {
	if got := quoteEnvValue("simplevalue"); got != "simplevalue" {
		t.Errorf("plain value should not be quoted, got %q", got)
	}
}

func TestQuoteEnvValue_ValueWithSpace(t *testing.T) {
	got := quoteEnvValue("hello world")
	if got != `"hello world"` {
		t.Errorf("got %q, want %q", got, `"hello world"`)
	}
}

func TestQuoteEnvValue_ValueWithHash(t *testing.T) {
	got := quoteEnvValue("value#comment")
	if got != `"value#comment"` {
		t.Errorf("got %q, want %q", got, `"value#comment"`)
	}
}

func TestQuoteEnvValue_ValueWithDollar(t *testing.T) {
	got := quoteEnvValue("$VAR")
	if got != `"$VAR"` {
		t.Errorf("got %q, want %q", got, `"$VAR"`)
	}
}

func TestQuoteEnvValue_ValueWithDoubleQuote(t *testing.T) {
	got := quoteEnvValue(`say "hello"`)
	want := `"say \"hello\""`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestQuoteEnvValue_ValueWithBackslash(t *testing.T) {
	got := quoteEnvValue(`path\to\file`)
	want := `"path\\to\\file"`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestQuoteEnvValue_EmptyString(t *testing.T) {
	if got := quoteEnvValue(""); got != "" {
		t.Errorf("empty value should not be quoted, got %q", got)
	}
}

func TestQuoteEnvValue_ValueWithNewline(t *testing.T) {
	got := quoteEnvValue("line1\nline2")
	if got[0] != '"' {
		t.Errorf("value with newline should be quoted, got %q", got)
	}
}

func TestQuoteEnvValue_ValueWithEquals(t *testing.T) {
	got := quoteEnvValue("key=val")
	if got != `"key=val"` {
		t.Errorf("got %q, want %q", got, `"key=val"`)
	}
}
