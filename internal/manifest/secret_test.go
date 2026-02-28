package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

// ---- NewSecret ----

func TestNewSecret_Defaults(t *testing.T) {
	s := NewSecret("my-secret", "staging")
	if s.Name != "my-secret" {
		t.Errorf("Name = %q, want \"my-secret\"", s.Name)
	}
	if s.Namespace != "staging" {
		t.Errorf("Namespace = %q, want \"staging\"", s.Namespace)
	}
	if s.APIVersion != "v1" {
		t.Errorf("APIVersion = %q, want \"v1\"", s.APIVersion)
	}
	if s.Kind != "Secret" {
		t.Errorf("Kind = %q, want \"Secret\"", s.Kind)
	}
	if s.Type != corev1.SecretTypeOpaque {
		t.Errorf("Type = %q, want Opaque", s.Type)
	}
	if s.Data == nil {
		t.Error("Data should be initialised, got nil")
	}
}

// ---- SetPlainValue / GetPlainValue ----

func TestSetGetPlainValue(t *testing.T) {
	s := NewSecret("s", "default")
	SetPlainValue(s, "API_KEY", "mysecret")

	val, err := GetPlainValue(s, "API_KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "mysecret" {
		t.Errorf("got %q, want \"mysecret\"", val)
	}
}

func TestSetPlainValue_NilDataMap(t *testing.T) {
	s := NewSecret("s", "default")
	s.Data = nil
	SetPlainValue(s, "KEY", "val")
	if string(s.Data["KEY"]) != "val" {
		t.Error("SetPlainValue should initialise nil Data map")
	}
}

func TestSetPlainValue_Overwrite(t *testing.T) {
	s := NewSecret("s", "default")
	SetPlainValue(s, "KEY", "original")
	SetPlainValue(s, "KEY", "updated")

	val, _ := GetPlainValue(s, "KEY")
	if val != "updated" {
		t.Errorf("got %q, want \"updated\"", val)
	}
}

func TestGetPlainValue_NotFound(t *testing.T) {
	s := NewSecret("s", "default")
	_, err := GetPlainValue(s, "MISSING")
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestGetPlainValue_EmptyValue(t *testing.T) {
	s := NewSecret("s", "default")
	SetPlainValue(s, "EMPTY", "")
	val, err := GetPlainValue(s, "EMPTY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "" {
		t.Errorf("got %q, want empty string", val)
	}
}

// ---- ToYAML ----

func TestToYAML_ContainsBase64(t *testing.T) {
	s := NewSecret("my-secret", "default")
	SetPlainValue(s, "API_KEY", "hello")

	out, err := ToYAML(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	yaml := string(out)
	// "hello" base64-encodes to "aGVsbG8="
	if !strings.Contains(yaml, "aGVsbG8=") {
		t.Errorf("YAML should contain base64 value, got:\n%s", yaml)
	}
	if !strings.Contains(yaml, "apiVersion: v1") {
		t.Errorf("YAML missing apiVersion, got:\n%s", yaml)
	}
	if !strings.Contains(yaml, "kind: Secret") {
		t.Errorf("YAML missing kind, got:\n%s", yaml)
	}
}

// ---- FromYAML ----

func TestFromYAML_RoundTrip(t *testing.T) {
	s := NewSecret("test-secret", "staging")
	SetPlainValue(s, "DB_PASS", "hunter2")
	SetPlainValue(s, "API_KEY", "abc123")

	data, err := ToYAML(s)
	if err != nil {
		t.Fatalf("ToYAML error: %v", err)
	}

	s2, err := FromYAML(data)
	if err != nil {
		t.Fatalf("FromYAML error: %v", err)
	}

	if s2.Name != "test-secret" {
		t.Errorf("Name = %q, want \"test-secret\"", s2.Name)
	}
	if s2.Namespace != "staging" {
		t.Errorf("Namespace = %q, want \"staging\"", s2.Namespace)
	}

	for _, key := range []string{"DB_PASS", "API_KEY"} {
		orig, _ := GetPlainValue(s, key)
		got, err := GetPlainValue(s2, key)
		if err != nil {
			t.Fatalf("GetPlainValue(%q): %v", key, err)
		}
		if got != orig {
			t.Errorf("key %q: got %q, want %q", key, got, orig)
		}
	}
}

func TestFromYAML_WrongKind(t *testing.T) {
	yaml := `apiVersion: v1
kind: ConfigMap
metadata:
  name: foo
`
	_, err := FromYAML([]byte(yaml))
	if err == nil {
		t.Error("expected error for wrong kind")
	}
}

func TestFromYAML_WrongAPIVersion(t *testing.T) {
	yaml := `apiVersion: apps/v1
kind: Secret
metadata:
  name: foo
`
	_, err := FromYAML([]byte(yaml))
	if err == nil {
		t.Error("expected error for wrong apiVersion")
	}
}

func TestFromYAML_NilDataInitialised(t *testing.T) {
	yaml := `apiVersion: v1
kind: Secret
metadata:
  name: empty
  namespace: default
type: Opaque
`
	s, err := FromYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Data == nil {
		t.Error("Data should be initialised to empty map, got nil")
	}
}

// ---- FromFile ----

func TestFromFile_HappyPath(t *testing.T) {
	s := NewSecret("file-secret", "default")
	SetPlainValue(s, "KEY", "value")

	data, _ := ToYAML(s)
	tmp := filepath.Join(t.TempDir(), "secret.yaml")
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	s2, err := FromFile(tmp)
	if err != nil {
		t.Fatalf("FromFile error: %v", err)
	}
	if s2.Name != "file-secret" {
		t.Errorf("Name = %q, want \"file-secret\"", s2.Name)
	}
}

func TestFromFile_NotFound(t *testing.T) {
	_, err := FromFile("/nonexistent/path/secret.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
