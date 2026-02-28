package validate_test

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pbsladek/k8s-secret-manifest/internal/validate"
)

// makeSecret builds a minimal valid Opaque secret for use in tests.
func makeSecret(name, namespace string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{"key": []byte("value")},
	}
}

// ---- name checks ----

func TestCheckName_Empty(t *testing.T) {
	s := makeSecret("", "default")
	if !hasError(validate.Secret(s), "name must not be empty") {
		t.Error("expected error for empty name")
	}
}

func TestCheckName_TooLong(t *testing.T) {
	s := makeSecret(strings.Repeat("a", 254), "default")
	if !hasErrorContaining(validate.Secret(s), "exceeds 253") {
		t.Error("expected error for name > 253 chars")
	}
}

func TestCheckName_InvalidChars(t *testing.T) {
	for _, bad := range []string{"My_Secret", "UPPER", "has space", "-leading", "trailing-"} {
		s := makeSecret(bad, "default")
		if !hasErrorContaining(validate.Secret(s), "not a valid DNS subdomain") {
			t.Errorf("name %q should fail DNS subdomain check", bad)
		}
	}
}

func TestCheckName_Valid(t *testing.T) {
	for _, good := range []string{"my-secret", "my.secret", "a", "a1b2", "x.y-z"} {
		s := makeSecret(good, "default")
		if hasErrorContaining(validate.Secret(s), "name") {
			t.Errorf("name %q should be valid", good)
		}
	}
}

func TestCheckName_AtMaxLength(t *testing.T) {
	s := makeSecret(strings.Repeat("a", 253), "default")
	if hasErrorContaining(validate.Secret(s), "exceeds 253") {
		t.Error("253-char name should be allowed")
	}
}

// ---- namespace checks ----

func TestCheckNamespace_Empty(t *testing.T) {
	s := makeSecret("valid", "")
	if !hasError(validate.Secret(s), "namespace must not be empty") {
		t.Error("expected error for empty namespace")
	}
}

func TestCheckNamespace_TooLong(t *testing.T) {
	s := makeSecret("valid", strings.Repeat("a", 64))
	if !hasErrorContaining(validate.Secret(s), "exceeds 63") {
		t.Error("expected error for namespace > 63 chars")
	}
}

func TestCheckNamespace_InvalidChars(t *testing.T) {
	for _, bad := range []string{"My.Namespace", "UPPER", "has_underscore", "-leading", "trailing-"} {
		s := makeSecret("valid", bad)
		if !hasErrorContaining(validate.Secret(s), "not a valid DNS label") {
			t.Errorf("namespace %q should fail DNS label check", bad)
		}
	}
}

func TestCheckNamespace_Valid(t *testing.T) {
	for _, good := range []string{"default", "kube-system", "my-ns", "a"} {
		s := makeSecret("valid", good)
		if hasErrorContaining(validate.Secret(s), "namespace") {
			t.Errorf("namespace %q should be valid", good)
		}
	}
}

func TestCheckNamespace_AtMaxLength(t *testing.T) {
	s := makeSecret("valid", strings.Repeat("a", 63))
	if hasErrorContaining(validate.Secret(s), "exceeds 63") {
		t.Error("63-char namespace should be allowed")
	}
}

// ---- data key checks ----

func TestCheckDataKeys_Empty(t *testing.T) {
	s := makeSecret("valid", "default")
	s.Data = map[string][]byte{}
	if !hasWarningContaining(validate.Secret(s), "no data keys") {
		t.Error("expected warning for empty data map")
	}
}

func TestCheckDataKeys_InvalidKey(t *testing.T) {
	for _, bad := range []string{"invalid key!", "key/with/slash", "key:colon"} {
		s := makeSecret("valid", "default")
		s.Data = map[string][]byte{bad: []byte("v")}
		if !hasErrorContaining(validate.Secret(s), "invalid characters") {
			t.Errorf("data key %q should fail validation", bad)
		}
	}
}

func TestCheckDataKeys_ValidKey(t *testing.T) {
	for _, good := range []string{"KEY", "key-name", "key_name", "key.txt", "KEY1"} {
		s := makeSecret("valid", "default")
		s.Data = map[string][]byte{good: []byte("v")}
		if hasErrorContaining(validate.Secret(s), "invalid characters") {
			t.Errorf("data key %q should be valid", good)
		}
	}
}

// ---- TLS type ----

func TestTLS_MissingBoth(t *testing.T) {
	s := makeSecret("valid", "default")
	s.Type = corev1.SecretTypeTLS
	s.Data = map[string][]byte{}
	issues := validate.Secret(s)
	if !hasErrorContaining(issues, "tls.crt") || !hasErrorContaining(issues, "tls.key") {
		t.Errorf("expected errors for missing tls.crt and tls.key, got: %v", issues)
	}
}

func TestTLS_MissingKey(t *testing.T) {
	s := makeSecret("valid", "default")
	s.Type = corev1.SecretTypeTLS
	s.Data = map[string][]byte{"tls.crt": []byte("cert")}
	if !hasErrorContaining(validate.Secret(s), "tls.key") {
		t.Error("expected error for missing tls.key")
	}
}

func TestTLS_Valid(t *testing.T) {
	s := makeSecret("valid", "default")
	s.Type = corev1.SecretTypeTLS
	s.Data = map[string][]byte{"tls.crt": []byte("cert"), "tls.key": []byte("key")}
	if hasAnyError(validate.Secret(s)) {
		t.Error("valid TLS secret should have no errors")
	}
}

// ---- DockerConfigJson type ----

func TestDockerConfigJson_Missing(t *testing.T) {
	s := makeSecret("valid", "default")
	s.Type = corev1.SecretTypeDockerConfigJson
	s.Data = map[string][]byte{}
	if !hasErrorContaining(validate.Secret(s), ".dockerconfigjson") {
		t.Error("expected error for missing .dockerconfigjson")
	}
}

func TestDockerConfigJson_Valid(t *testing.T) {
	s := makeSecret("valid", "default")
	s.Type = corev1.SecretTypeDockerConfigJson
	s.Data = map[string][]byte{corev1.DockerConfigJsonKey: []byte("{}")}
	if hasAnyError(validate.Secret(s)) {
		t.Error("valid docker-registry secret should have no errors")
	}
}

// ---- BasicAuth type ----

func TestBasicAuth_MissingBoth_Warns(t *testing.T) {
	s := makeSecret("valid", "default")
	s.Type = corev1.SecretTypeBasicAuth
	s.Data = map[string][]byte{}
	issues := validate.Secret(s)
	if !hasWarningContaining(issues, "username") || !hasWarningContaining(issues, "password") {
		t.Errorf("expected warnings for missing username/password, got: %v", issues)
	}
}

func TestBasicAuth_Valid_NoErrors(t *testing.T) {
	s := makeSecret("valid", "default")
	s.Type = corev1.SecretTypeBasicAuth
	s.Data = map[string][]byte{"username": []byte("u"), "password": []byte("p")}
	if hasAnyError(validate.Secret(s)) {
		t.Error("valid basic-auth secret should have no errors")
	}
}

// ---- SSHAuth type ----

func TestSSHAuth_Missing(t *testing.T) {
	s := makeSecret("valid", "default")
	s.Type = corev1.SecretTypeSSHAuth
	s.Data = map[string][]byte{}
	if !hasErrorContaining(validate.Secret(s), "ssh-privatekey") {
		t.Error("expected error for missing ssh-privatekey")
	}
}

func TestSSHAuth_Valid(t *testing.T) {
	s := makeSecret("valid", "default")
	s.Type = corev1.SecretTypeSSHAuth
	s.Data = map[string][]byte{"ssh-privatekey": []byte("key")}
	if hasAnyError(validate.Secret(s)) {
		t.Error("valid ssh-auth secret should have no errors")
	}
}

// ---- ServiceAccountToken type ----

func TestServiceAccountToken_Missing(t *testing.T) {
	s := makeSecret("valid", "default")
	s.Type = corev1.SecretTypeServiceAccountToken
	s.Data = map[string][]byte{}
	if !hasErrorContaining(validate.Secret(s), "token") {
		t.Error("expected error for missing token")
	}
}

func TestServiceAccountToken_Valid(t *testing.T) {
	s := makeSecret("valid", "default")
	s.Type = corev1.SecretTypeServiceAccountToken
	s.Data = map[string][]byte{"token": []byte("t")}
	if hasAnyError(validate.Secret(s)) {
		t.Error("valid service-account-token secret should have no errors")
	}
}

// ---- Issue methods ----

func TestIssue_IsError(t *testing.T) {
	e := validate.Issue{Severity: validate.SeverityError, Message: "msg"}
	if !e.IsError() {
		t.Error("expected IsError=true for error severity")
	}
	w := validate.Issue{Severity: validate.SeverityWarning, Message: "msg"}
	if w.IsError() {
		t.Error("expected IsError=false for warning severity")
	}
}

func TestIssue_String(t *testing.T) {
	i := validate.Issue{Severity: validate.SeverityError, Message: "bad thing"}
	if got := i.String(); got != "error: bad thing" {
		t.Errorf("unexpected String: %q", got)
	}
}

// ---- helpers ----

func hasError(issues []validate.Issue, msg string) bool {
	for _, i := range issues {
		if i.IsError() && i.Message == msg {
			return true
		}
	}
	return false
}

func hasErrorContaining(issues []validate.Issue, substr string) bool {
	for _, i := range issues {
		if i.IsError() && strings.Contains(i.Message, substr) {
			return true
		}
	}
	return false
}

func hasWarningContaining(issues []validate.Issue, substr string) bool {
	for _, i := range issues {
		if !i.IsError() && strings.Contains(i.Message, substr) {
			return true
		}
	}
	return false
}

func hasAnyError(issues []validate.Issue) bool {
	for _, i := range issues {
		if i.IsError() {
			return true
		}
	}
	return false
}
