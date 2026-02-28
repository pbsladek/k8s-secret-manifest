package entrylist

import (
	"testing"
)

// ---- Parse ----

func TestParse_HappyPath(t *testing.T) {
	entries, err := Parse("alice;bob", "pass1;pass2", ";")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
	if entries[0].Key != "alice" || entries[0].Value != "pass1" {
		t.Errorf("entries[0] = %+v, want {alice pass1}", entries[0])
	}
	if entries[1].Key != "bob" || entries[1].Value != "pass2" {
		t.Errorf("entries[1] = %+v, want {bob pass2}", entries[1])
	}
}

func TestParse_CustomSeparator(t *testing.T) {
	entries, err := Parse("alice,bob", "pass1,pass2", ",")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
}

func TestParse_Empty(t *testing.T) {
	entries, err := Parse("", "", ";")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("want 0 entries, got %d", len(entries))
	}
}

func TestParse_LengthMismatch(t *testing.T) {
	_, err := Parse("alice;bob", "pass1", ";")
	if err == nil {
		t.Error("expected error for mismatched lengths")
	}
}

func TestParse_EmptyKey(t *testing.T) {
	_, err := Parse(";bob", "pass1;pass2", ";")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestParse_WhitespaceTrimmed(t *testing.T) {
	entries, err := Parse(" alice ; bob ", " pass1 ; pass2 ", ";")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entries[0].Key != "alice" {
		t.Errorf("want key trimmed to \"alice\", got %q", entries[0].Key)
	}
}

// ---- Serialize ----

func TestSerialize_HappyPath(t *testing.T) {
	entries := []Entry{{Key: "alice", Value: "pass1"}, {Key: "bob", Value: "pass2"}}
	keys, vals := Serialize(entries, ";")
	if keys != "alice;bob" {
		t.Errorf("keys = %q, want \"alice;bob\"", keys)
	}
	if vals != "pass1;pass2" {
		t.Errorf("vals = %q, want \"pass1;pass2\"", vals)
	}
}

func TestSerialize_Empty(t *testing.T) {
	keys, vals := Serialize(nil, ";")
	if keys != "" || vals != "" {
		t.Errorf("want empty strings, got %q %q", keys, vals)
	}
}

func TestSerialize_RoundTrip(t *testing.T) {
	original := []Entry{
		{Key: "alice", Value: "pass1"},
		{Key: "bob", Value: "pass2"},
		{Key: "carol", Value: "pass3"},
	}
	keys, vals := Serialize(original, ";")
	result, err := Parse(keys, vals, ";")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, e := range original {
		if result[i] != e {
			t.Errorf("round-trip mismatch at %d: got %+v, want %+v", i, result[i], e)
		}
	}
}

// ---- Add ----

func TestAdd_Append(t *testing.T) {
	entries := []Entry{{Key: "alice", Value: "pass1"}}
	result, err := Add(entries, "bob", "pass2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("want 2 entries, got %d", len(result))
	}
	if result[1].Key != "bob" || result[1].Value != "pass2" {
		t.Errorf("result[1] = %+v, want {bob pass2}", result[1])
	}
}

func TestAdd_EmptySlice(t *testing.T) {
	result, err := Add(nil, "alice", "pass1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 || result[0].Key != "alice" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestAdd_DuplicateKey(t *testing.T) {
	entries := []Entry{{Key: "alice", Value: "pass1"}}
	_, err := Add(entries, "alice", "other")
	if err == nil {
		t.Error("expected error for duplicate key")
	}
}

func TestAdd_EmptyKey(t *testing.T) {
	_, err := Add(nil, "", "val")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

// ---- Remove ----

func TestRemove_HappyPath(t *testing.T) {
	entries := []Entry{
		{Key: "alice", Value: "pass1"},
		{Key: "bob", Value: "pass2"},
		{Key: "carol", Value: "pass3"},
	}
	result, err := Remove(entries, "bob")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("want 2 entries, got %d", len(result))
	}
	if result[0].Key != "alice" || result[1].Key != "carol" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestRemove_NotFound(t *testing.T) {
	entries := []Entry{{Key: "alice", Value: "pass1"}}
	_, err := Remove(entries, "nobody")
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestRemove_LastEntry(t *testing.T) {
	entries := []Entry{{Key: "alice", Value: "pass1"}}
	result, err := Remove(entries, "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("want empty slice, got %+v", result)
	}
}

// ---- RemoveByValue ----

func TestRemoveByValue_HappyPath(t *testing.T) {
	entries := []Entry{
		{Key: "alice", Value: "pass1"},
		{Key: "bob", Value: "pass2"},
	}
	result, err := RemoveByValue(entries, "pass1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 || result[0].Key != "bob" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestRemoveByValue_NotFound(t *testing.T) {
	entries := []Entry{{Key: "alice", Value: "pass1"}}
	_, err := RemoveByValue(entries, "nope")
	if err == nil {
		t.Error("expected error for missing value")
	}
}

func TestRemoveByValue_RemovesFirstMatch(t *testing.T) {
	entries := []Entry{
		{Key: "alice", Value: "shared"},
		{Key: "bob", Value: "shared"},
	}
	result, err := RemoveByValue(entries, "shared")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 || result[0].Key != "bob" {
		t.Errorf("expected only bob remaining, got %+v", result)
	}
}

// ---- Insert ----

func TestInsert_AtBeginning(t *testing.T) {
	entries := []Entry{{Key: "bob", Value: "pass2"}}
	result, err := Insert(entries, 0, "alice", "pass1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result[0].Key != "alice" || result[1].Key != "bob" {
		t.Errorf("unexpected order: %+v", result)
	}
}

func TestInsert_AtMiddle(t *testing.T) {
	entries := []Entry{
		{Key: "alice", Value: "pass1"},
		{Key: "carol", Value: "pass3"},
	}
	result, err := Insert(entries, 1, "bob", "pass2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("want 3, got %d", len(result))
	}
	if result[0].Key != "alice" || result[1].Key != "bob" || result[2].Key != "carol" {
		t.Errorf("unexpected order: %+v", result)
	}
}

func TestInsert_AtEnd(t *testing.T) {
	entries := []Entry{{Key: "alice", Value: "pass1"}}
	result, err := Insert(entries, 1, "bob", "pass2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result[1].Key != "bob" {
		t.Errorf("expected bob at end, got %+v", result)
	}
}

func TestInsert_OutOfRange(t *testing.T) {
	entries := []Entry{{Key: "alice", Value: "pass1"}}
	_, err := Insert(entries, 5, "bob", "pass2")
	if err == nil {
		t.Error("expected error for out-of-range index")
	}
}

func TestInsert_NegativeIndex(t *testing.T) {
	_, err := Insert(nil, -1, "alice", "pass1")
	if err == nil {
		t.Error("expected error for negative index")
	}
}

func TestInsert_DuplicateKey(t *testing.T) {
	entries := []Entry{{Key: "alice", Value: "pass1"}}
	_, err := Insert(entries, 0, "alice", "other")
	if err == nil {
		t.Error("expected error for duplicate key")
	}
}

func TestInsert_EmptyKey(t *testing.T) {
	_, err := Insert(nil, 0, "", "val")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

// ---- Keys ----

func TestKeys(t *testing.T) {
	entries := []Entry{
		{Key: "alice", Value: "pass1"},
		{Key: "bob", Value: "pass2"},
	}
	keys := Keys(entries)
	if len(keys) != 2 || keys[0] != "alice" || keys[1] != "bob" {
		t.Errorf("unexpected keys: %v", keys)
	}
}

func TestKeys_Empty(t *testing.T) {
	keys := Keys(nil)
	if len(keys) != 0 {
		t.Errorf("want empty, got %v", keys)
	}
}
