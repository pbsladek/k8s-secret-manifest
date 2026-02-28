// Package validate checks Kubernetes Secret manifests for correctness.
package validate

import (
	"fmt"
	"regexp"

	corev1 "k8s.io/api/core/v1"
)

// Severity levels for Issue.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
)

// Issue represents a single validation finding.
type Issue struct {
	Severity string
	Message  string
}

func (i Issue) IsError() bool { return i.Severity == SeverityError }

func (i Issue) String() string { return i.Severity + ": " + i.Message }

var (
	// Secret names follow DNS subdomain rules: lowercase alphanumeric, hyphens, dots; max 253.
	nameRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9.\-]*[a-z0-9])?$`)

	// Namespace names follow DNS label rules: lowercase alphanumeric and hyphens; max 63.
	namespaceRe = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

	// Data key names: alphanumeric, hyphen, underscore, dot.
	dataKeyRe = regexp.MustCompile(`^[-._a-zA-Z0-9]+$`)
)

// Secret validates a corev1.Secret and returns all findings.
// Errors indicate spec violations; warnings indicate likely mistakes.
func Secret(s *corev1.Secret) []Issue {
	var issues []Issue

	issues = append(issues, checkName(s)...)
	issues = append(issues, checkNamespace(s)...)
	issues = append(issues, checkDataKeys(s)...)
	issues = append(issues, checkTypeRequirements(s)...)

	return issues
}

func checkName(s *corev1.Secret) []Issue {
	if s.Name == "" {
		return []Issue{{SeverityError, "name must not be empty"}}
	}
	if len(s.Name) > 253 {
		return []Issue{{SeverityError, fmt.Sprintf("name %q exceeds 253 characters", s.Name)}}
	}
	if !nameRe.MatchString(s.Name) {
		return []Issue{{SeverityError, fmt.Sprintf(
			"name %q is not a valid DNS subdomain (lowercase alphanumeric, hyphens, dots; must start and end with alphanumeric)",
			s.Name,
		)}}
	}
	return nil
}

func checkNamespace(s *corev1.Secret) []Issue {
	if s.Namespace == "" {
		return []Issue{{SeverityError, "namespace must not be empty"}}
	}
	if len(s.Namespace) > 63 {
		return []Issue{{SeverityError, fmt.Sprintf("namespace %q exceeds 63 characters", s.Namespace)}}
	}
	if !namespaceRe.MatchString(s.Namespace) {
		return []Issue{{SeverityError, fmt.Sprintf(
			"namespace %q is not a valid DNS label (lowercase alphanumeric and hyphens; must start and end with alphanumeric)",
			s.Namespace,
		)}}
	}
	return nil
}

func checkDataKeys(s *corev1.Secret) []Issue {
	var issues []Issue
	if len(s.Data) == 0 {
		issues = append(issues, Issue{SeverityWarning, "secret has no data keys"})
	}
	for k := range s.Data {
		if !dataKeyRe.MatchString(k) {
			issues = append(issues, Issue{SeverityError, fmt.Sprintf(
				"data key %q contains invalid characters (allowed: alphanumeric, '-', '_', '.')",
				k,
			)})
		}
	}
	return issues
}

func checkTypeRequirements(s *corev1.Secret) []Issue {
	var issues []Issue

	required := func(key string) {
		if _, ok := s.Data[key]; !ok {
			issues = append(issues, Issue{SeverityError, fmt.Sprintf(
				"type %s requires data key %q", s.Type, key,
			)})
		}
	}
	recommended := func(key string) {
		if _, ok := s.Data[key]; !ok {
			issues = append(issues, Issue{SeverityWarning, fmt.Sprintf(
				"type %s typically requires data key %q", s.Type, key,
			)})
		}
	}

	switch s.Type {
	case corev1.SecretTypeTLS:
		required("tls.crt")
		required("tls.key")
	case corev1.SecretTypeDockerConfigJson:
		required(corev1.DockerConfigJsonKey)
	case corev1.SecretTypeBasicAuth:
		recommended("username")
		recommended("password")
	case corev1.SecretTypeSSHAuth:
		required("ssh-privatekey")
	case corev1.SecretTypeServiceAccountToken:
		required("token")
	}

	return issues
}
