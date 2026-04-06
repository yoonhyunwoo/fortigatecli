package output

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"fortigatecli/internal/fortigate"
)

func TestWriteTableRendersDetailEnvelope(t *testing.T) {
	var out bytes.Buffer
	err := Write(&out, "table", &fortigate.Envelope{
		Results: map[string]any{
			"name":   "branch-office",
			"subnet": "10.0.0.0/24",
		},
	})
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	text := out.String()
	for _, needle := range []string{"name", "branch-office", "subnet"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("detail table missing %q\n%s", needle, text)
		}
	}
}

func TestWriteTableRendersListEnvelope(t *testing.T) {
	var out bytes.Buffer
	err := Write(&out, "table", &fortigate.Envelope{
		Results: []any{
			map[string]any{"id": 1, "name": "port1", "zone": "internal"},
			map[string]any{"id": 2, "name": "port2"},
		},
	})
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	text := out.String()
	for _, needle := range []string{"id", "name", "port1", "port2"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("list table missing %q\n%s", needle, text)
		}
	}
}

func TestApplySelectorSupportsDotIndexAndWildcard(t *testing.T) {
	value := map[string]any{
		"results": []any{
			map[string]any{"name": "port1", "ip": "10.0.0.1"},
			map[string]any{"name": "port2", "ip": "10.0.0.2"},
		},
	}
	got, err := applySelector(value, ".results[*].name")
	if err != nil {
		t.Fatalf("applySelector returned error: %v", err)
	}
	want := []any{"port1", "port2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("applySelector = %#v, want %#v", got, want)
	}
}

func TestWriteShapedWithoutFlagsMatchesWrite(t *testing.T) {
	value := map[string]any{
		"http_status": 200,
		"results":     []any{map[string]any{"name": "port1"}},
	}
	var plain bytes.Buffer
	if err := Write(&plain, "json", value); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	var shaped bytes.Buffer
	if err := WriteShaped(&shaped, "json", value, ShapeOptions{}); err != nil {
		t.Fatalf("WriteShaped returned error: %v", err)
	}
	if plain.String() != shaped.String() {
		t.Fatalf("WriteShaped output differed without flags")
	}
}

func TestWriteShapedSelectDefaultsToResults(t *testing.T) {
	value := map[string]any{
		"http_status": 200,
		"results": []any{
			map[string]any{"name": "port1", "ip": "10.0.0.1"},
		},
	}
	var out bytes.Buffer
	err := WriteShaped(&out, "json", value, ShapeOptions{Select: []string{"name"}})
	if err != nil {
		t.Fatalf("WriteShaped returned error: %v", err)
	}
	var got []map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	want := []map[string]any{{"name": "port1"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("WriteShaped default target = %#v, want %#v", got, want)
	}
}
