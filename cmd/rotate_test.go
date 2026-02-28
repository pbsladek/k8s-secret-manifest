package cmd

import (
	"strings"
	"testing"
)

// ---- resolveCharset ----

func TestResolveCharset_Alphanumeric(t *testing.T) {
	cs, err := resolveCharset("alphanumeric")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cs != charsetAlphanumeric {
		t.Errorf("unexpected charset returned")
	}
}

func TestResolveCharset_Hex(t *testing.T) {
	cs, err := resolveCharset("hex")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cs != charsetHex {
		t.Errorf("unexpected charset returned")
	}
}

func TestResolveCharset_Base64URL(t *testing.T) {
	cs, err := resolveCharset("base64url")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cs != charsetBase64URL {
		t.Errorf("unexpected charset returned")
	}
}

func TestResolveCharset_CaseInsensitive(t *testing.T) {
	_, err := resolveCharset("ALPHANUMERIC")
	if err != nil {
		t.Fatalf("charset lookup should be case-insensitive: %v", err)
	}
}

func TestResolveCharset_Unknown(t *testing.T) {
	_, err := resolveCharset("base58")
	if err == nil {
		t.Error("expected error for unknown charset")
	}
}

// ---- randomString ----

func TestRandomString_Length(t *testing.T) {
	for _, length := range []int{1, 16, 32, 64, 128} {
		got, err := randomString(length, charsetAlphanumeric)
		if err != nil {
			t.Fatalf("length %d: unexpected error: %v", length, err)
		}
		if len(got) != length {
			t.Errorf("length %d: got string of length %d", length, len(got))
		}
	}
}

func TestRandomString_OnlyCharsetChars(t *testing.T) {
	for _, tc := range []struct {
		name    string
		charset string
	}{
		{"alphanumeric", charsetAlphanumeric},
		{"hex", charsetHex},
		{"base64url", charsetBase64URL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := randomString(64, tc.charset)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, c := range got {
				if !strings.ContainsRune(tc.charset, c) {
					t.Errorf("character %q not in charset %q", c, tc.charset)
				}
			}
		})
	}
}

func TestRandomString_ZeroLength(t *testing.T) {
	_, err := randomString(0, charsetAlphanumeric)
	if err == nil {
		t.Error("expected error for length 0")
	}
}

func TestRandomString_NegativeLength(t *testing.T) {
	_, err := randomString(-5, charsetAlphanumeric)
	if err == nil {
		t.Error("expected error for negative length")
	}
}

func TestRandomString_Uniqueness(t *testing.T) {
	// Two 32-char random strings should essentially never be equal
	a, _ := randomString(32, charsetAlphanumeric)
	b, _ := randomString(32, charsetAlphanumeric)
	if a == b {
		t.Error("two random strings were identical (astronomically unlikely)")
	}
}
