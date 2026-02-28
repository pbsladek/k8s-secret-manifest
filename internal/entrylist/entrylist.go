// Package entrylist manages two parallel delimiter-separated lists stored in
// two separate K8s Secret data keys, where entries match by index position.
//
// Example — two data keys whose values are index-paired (separator ";"):
//
//	BACKEND_USERS:     alice;bob;carol
//	BACKEND_PASSWORDS: pass1;pass2;pass3
//
// Position determines pairing: index 0 of keys ↔ index 0 of values.
// The separator is configurable (default ";") and values are stored as plain
// text (base64 encoding is handled by the manifest layer).
package entrylist

import (
	"fmt"
	"strings"
)

// Entry holds one key/value pair from a paired index list.
type Entry struct {
	Key   string
	Value string
}

// Parse decodes two plain-text delimiter-separated strings into an Entry slice.
// keysVal and valuesVal must already be plain text (not base64-encoded).
// sep is the list separator (e.g. ";"). Returns an error if the two lists
// have different lengths.
func Parse(keysVal, valuesVal, sep string) ([]Entry, error) {
	keys := splitTrimmed(keysVal, sep)
	values := splitTrimmed(valuesVal, sep)

	if len(keys) != len(values) {
		return nil, fmt.Errorf(
			"entry list mismatch: %d key(s) but %d value(s)",
			len(keys), len(values),
		)
	}

	entries := make([]Entry, len(keys))
	for i := range keys {
		if keys[i] == "" {
			return nil, fmt.Errorf("empty key at index %d", i)
		}
		entries[i] = Entry{Key: keys[i], Value: values[i]}
	}
	return entries, nil
}

// Serialize converts an Entry slice back into the two delimiter-separated
// plain-text strings ready to be stored in the Secret.
// sep is the list separator (e.g. ";").
func Serialize(entries []Entry, sep string) (keysVal, valuesVal string) {
	keys := make([]string, len(entries))
	values := make([]string, len(entries))
	for i, e := range entries {
		keys[i] = e.Key
		values[i] = e.Value
	}
	return strings.Join(keys, sep), strings.Join(values, sep)
}

// Add appends a new Entry. Returns an error if the key already exists.
func Add(entries []Entry, key, value string) ([]Entry, error) {
	if key == "" {
		return nil, fmt.Errorf("key must not be empty")
	}
	for _, e := range entries {
		if e.Key == key {
			return nil, fmt.Errorf("entry %q already exists", key)
		}
	}
	return append(entries, Entry{Key: key, Value: value}), nil
}

// Remove removes the entry with the given key.
// Returns an error if no entry with that key is found.
func Remove(entries []Entry, key string) ([]Entry, error) {
	result := make([]Entry, 0, len(entries))
	found := false
	for _, e := range entries {
		if e.Key == key {
			found = true
			continue
		}
		result = append(result, e)
	}
	if !found {
		return nil, fmt.Errorf("entry with key %q not found", key)
	}
	return result, nil
}

// RemoveByValue removes the first entry whose Value matches the given value.
// Returns an error if no matching entry is found.
func RemoveByValue(entries []Entry, value string) ([]Entry, error) {
	result := make([]Entry, 0, len(entries))
	found := false
	for _, e := range entries {
		if !found && e.Value == value {
			found = true
			continue
		}
		result = append(result, e)
	}
	if !found {
		return nil, fmt.Errorf("entry with value %q not found", value)
	}
	return result, nil
}

// Insert inserts a new Entry at index idx.
// Index 0 prepends; index len(entries) appends.
// Returns an error if idx is out of range or the key already exists.
func Insert(entries []Entry, idx int, key, value string) ([]Entry, error) {
	if key == "" {
		return nil, fmt.Errorf("key must not be empty")
	}
	if idx < 0 || idx > len(entries) {
		return nil, fmt.Errorf("index %d out of range [0, %d]", idx, len(entries))
	}
	for _, e := range entries {
		if e.Key == key {
			return nil, fmt.Errorf("entry %q already exists", key)
		}
	}
	result := make([]Entry, 0, len(entries)+1)
	result = append(result, entries[:idx]...)
	result = append(result, Entry{Key: key, Value: value})
	result = append(result, entries[idx:]...)
	return result, nil
}

// Keys returns the ordered list of entry keys.
func Keys(entries []Entry) []string {
	keys := make([]string, len(entries))
	for i, e := range entries {
		keys[i] = e.Key
	}
	return keys
}

// splitTrimmed splits s by sep, trims whitespace, and drops empty strings.
func splitTrimmed(s, sep string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
