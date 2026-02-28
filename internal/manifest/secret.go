package manifest

import (
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// NewSecret returns an initialised corev1.Secret with sensible defaults.
func NewSecret(name, namespace string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: make(map[string][]byte),
	}
}

// SetPlainValue stores a plain-text value under key.
// Base64 encoding is handled automatically during YAML serialisation.
func SetPlainValue(s *corev1.Secret, key, plaintext string) {
	if s.Data == nil {
		s.Data = make(map[string][]byte)
	}
	s.Data[key] = []byte(plaintext)
}

// GetPlainValue returns the plain-text value for key.
// Base64 decoding is handled automatically during YAML deserialisation.
func GetPlainValue(s *corev1.Secret, key string) (string, error) {
	val, ok := s.Data[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret data", key)
	}
	return string(val), nil
}

// ToYAML serialises the secret to valid Kubernetes YAML.
// sigs.k8s.io/yaml marshals via JSON, so map[string][]byte values are
// automatically base64-encoded as required by the data: field.
func ToYAML(s *corev1.Secret) ([]byte, error) {
	out, err := yaml.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("serialize secret: %w", err)
	}
	return out, nil
}

// FromYAML parses a Kubernetes Secret manifest from YAML bytes.
// Base64 values in data: are decoded automatically into []byte.
func FromYAML(data []byte) (*corev1.Secret, error) {
	var s corev1.Secret
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse secret YAML: %w", err)
	}
	if s.Kind != "Secret" || s.APIVersion != "v1" {
		return nil, fmt.Errorf("expected apiVersion=v1 kind=Secret, got apiVersion=%s kind=%s", s.APIVersion, s.Kind)
	}
	if s.Data == nil {
		s.Data = make(map[string][]byte)
	}
	return &s, nil
}

// FromFile reads and parses a Secret manifest from disk.
func FromFile(path string) (*corev1.Secret, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", path, err)
	}
	return FromYAML(data)
}
