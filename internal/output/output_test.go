package output

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestUnwrapResponse_WrappedList(t *testing.T) {
	obj := map[string]interface{}{
		"posts": []interface{}{
			map[string]interface{}{"id": "abc", "title": "Test"},
		},
		"nextCursor": "cursor123",
		"context":    "Some context",
		"guidelines": map[string]interface{}{
			"general": "Be helpful",
		},
	}

	key, data, meta := unwrapResponse(obj)
	if key != "posts" {
		t.Errorf("expected key 'posts', got %q", key)
	}
	arr, ok := data.([]interface{})
	if !ok || len(arr) != 1 {
		t.Errorf("expected array of 1 item, got %v", data)
	}
	if meta == nil {
		t.Fatal("expected meta, got nil")
	}
	if meta.NextCursor != "cursor123" {
		t.Errorf("expected nextCursor 'cursor123', got %q", meta.NextCursor)
	}
	if meta.Context != "Some context" {
		t.Errorf("expected context 'Some context', got %q", meta.Context)
	}
}

func TestUnwrapResponse_WrappedSingleEntity(t *testing.T) {
	obj := map[string]interface{}{
		"post": map[string]interface{}{"id": "abc", "title": "Test"},
		"context":    "Some context",
		"guidelines": map[string]interface{}{"general": "Be helpful"},
	}

	key, data, meta := unwrapResponse(obj)
	if key != "post" {
		t.Errorf("expected key 'post', got %q", key)
	}
	entity, ok := data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object, got %T", data)
	}
	if entity["id"] != "abc" {
		t.Errorf("expected id 'abc', got %v", entity["id"])
	}
	if meta == nil {
		t.Fatal("expected meta, got nil")
	}
}

func TestUnwrapResponse_NotWrapped(t *testing.T) {
	// Context endpoint has guidelines as a string, not an object
	obj := map[string]interface{}{
		"scope":      "company",
		"stats":      map[string]interface{}{"agents": float64(5)},
		"guidelines": "Be helpful", // string, not object
	}

	_, data, meta := unwrapResponse(obj)
	if data != nil {
		t.Errorf("expected nil data for non-wrapped response, got %v", data)
	}
	if meta != nil {
		t.Errorf("expected nil meta for non-wrapped response, got %v", meta)
	}
}

func TestUnwrapResponse_ExtraFields(t *testing.T) {
	// Votes GET returns {vote: {...}, tally: {...}, context, guidelines}
	obj := map[string]interface{}{
		"vote":       map[string]interface{}{"id": "v1", "title": "Launch?"},
		"tally":      map[string]interface{}{"Yes": float64(3), "No": float64(1)},
		"context":    "ctx",
		"guidelines": map[string]interface{}{"voting": "Vote wisely"},
	}

	_, data, meta := unwrapResponse(obj)
	entity, ok := data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object, got %T", data)
	}
	// tally should be merged into the entity
	if _, hasTally := entity["tally"]; !hasTally {
		t.Error("expected tally to be merged into entity")
	}
	if meta == nil {
		t.Fatal("expected meta")
	}
}

func TestFormatCell_NestedObject(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			"object with name",
			map[string]interface{}{"id": "x", "name": "Bot Builder", "username": "bot"},
			"Bot Builder",
		},
		{
			"object with username only",
			map[string]interface{}{"id": "x", "username": "bot"},
			"bot",
		},
		{
			"object with title",
			map[string]interface{}{"id": "x", "title": "Some Title"},
			"Some Title",
		},
		{
			"string array",
			[]interface{}{"Yes", "No", "Maybe"},
			"Yes, No, Maybe",
		},
		{
			"object array with names",
			[]interface{}{
				map[string]interface{}{"name": "Product A"},
				map[string]interface{}{"name": "Product B"},
			},
			"Product A, Product B",
		},
		{
			"nil",
			nil,
			"",
		},
		{
			"empty array",
			[]interface{}{},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCell(tt.input)
			if result != tt.expected {
				t.Errorf("formatCell(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractID_Wrapped(t *testing.T) {
	response := map[string]interface{}{
		"post":       map[string]interface{}{"id": "post123", "title": "Test"},
		"context":    "ctx",
		"guidelines": map[string]interface{}{"general": "..."},
	}
	data, _ := json.Marshal(response)
	id := ExtractID(data)
	if id != "post123" {
		t.Errorf("expected 'post123', got %q", id)
	}
}

func TestExtractID_Direct(t *testing.T) {
	response := map[string]interface{}{"id": "direct123", "name": "Test"}
	data, _ := json.Marshal(response)
	id := ExtractID(data)
	if id != "direct123" {
		t.Errorf("expected 'direct123', got %q", id)
	}
}

func TestIsSectionValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{"nil", nil, false},
		{"string", "hello", false},
		{"number", float64(42), false},
		{"empty array", []interface{}{}, false},
		{"string array", []interface{}{"a", "b"}, false},
		{
			"array of objects",
			[]interface{}{
				map[string]interface{}{"id": "1", "name": "A"},
			},
			true,
		},
		{
			"small nested object (2 keys)",
			map[string]interface{}{"name": "X", "username": "x"},
			false,
		},
		{
			"large nested object (3+ keys)",
			map[string]interface{}{"a": 1, "b": 2, "c": 3},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSectionValue(tt.input)
			if result != tt.expected {
				t.Errorf("isSectionValue(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPrintObjectTable_Sections(t *testing.T) {
	// Context-like response: inline scalars + array-of-objects + nested object
	obj := map[string]interface{}{
		"scope":              "company",
		"summary":            "A summary",
		"summary_updated_at": "2026-03-15",
		"guidelines":         "Be helpful",
		"products": []interface{}{
			map[string]interface{}{"id": "abc", "name": "Recon", "status": "live"},
			map[string]interface{}{"id": "def", "name": "Federal", "status": "building"},
		},
		"stats": map[string]interface{}{
			"active_products": float64(3),
			"approved_tasks":  float64(127),
			"claimed_agents":  float64(8),
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printObjectTable(obj)

	w.Close()
	os.Stdout = old
	var buf [4096]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	// Inline fields should appear (not in a section)
	if !strings.Contains(output, "scope") || !strings.Contains(output, "company") {
		t.Errorf("expected inline field 'scope: company' in output:\n%s", output)
	}
	if !strings.Contains(output, "guidelines") || !strings.Contains(output, "Be helpful") {
		t.Errorf("expected inline field 'guidelines' in output:\n%s", output)
	}

	// Section headers should appear
	if !strings.Contains(output, "--- products ---") {
		t.Errorf("expected '--- products ---' section header in output:\n%s", output)
	}
	if !strings.Contains(output, "--- stats ---") {
		t.Errorf("expected '--- stats ---' section header in output:\n%s", output)
	}

	// Array section should contain IDs (not just names)
	if !strings.Contains(output, "abc") || !strings.Contains(output, "def") {
		t.Errorf("expected product IDs 'abc' and 'def' in output:\n%s", output)
	}
}

func TestPrintObjectTable_OnlyInline(t *testing.T) {
	obj := map[string]interface{}{
		"id":     "abc",
		"name":   "Test",
		"status": "active",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printObjectTable(obj)

	w.Close()
	os.Stdout = old
	var buf [4096]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	// Should have key-value pairs, no section headers
	if strings.Contains(output, "---") {
		t.Errorf("expected no section headers for simple object, got:\n%s", output)
	}
	if !strings.Contains(output, "abc") || !strings.Contains(output, "Test") {
		t.Errorf("expected inline values in output:\n%s", output)
	}
}

func TestPrintObjectTable_EmptyArrays(t *testing.T) {
	obj := map[string]interface{}{
		"id":    "abc",
		"name":  "Test",
		"tags":  []interface{}{},
		"items": []interface{}{},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printObjectTable(obj)

	w.Close()
	os.Stdout = old
	var buf [4096]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	// Empty arrays should stay inline, not become sections
	if strings.Contains(output, "---") {
		t.Errorf("expected no section headers for empty arrays, got:\n%s", output)
	}
}
