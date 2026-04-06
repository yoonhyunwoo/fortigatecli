package output

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

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

	indexed, err := applySelector(value, ".results[0].ip")
	if err != nil {
		t.Fatalf("applySelector returned error: %v", err)
	}
	if indexed != "10.0.0.1" {
		t.Fatalf("indexed selector = %#v, want %#v", indexed, "10.0.0.1")
	}
}

func TestShapeValueFlattensNestedObjectsAndArrays(t *testing.T) {
	value := map[string]any{
		"results": []any{
			map[string]any{
				"name": "port1",
				"meta": map[string]any{
					"status": "up",
				},
				"ips": []any{"10.0.0.1", "10.0.0.2"},
			},
		},
	}

	got, err := shapeValue(value, ShapeOptions{Flatten: true, FlattenSep: "."})
	if err != nil {
		t.Fatalf("shapeValue returned error: %v", err)
	}

	wantValue := []any{
		map[string]any{
			"ips.0":       "10.0.0.1",
			"ips.1":       "10.0.0.2",
			"meta.status": "up",
			"name":        "port1",
		},
	}
	if !reflect.DeepEqual(got.value, wantValue) {
		t.Fatalf("shapeValue value = %#v, want %#v", got.value, wantValue)
	}
}

func TestShapeValueNormalizesHeterogeneousRows(t *testing.T) {
	value := []any{
		map[string]any{"name": "port1", "status": "up"},
		"orphan",
	}

	got, err := shapeValue(value, ShapeOptions{Columns: []string{"name", "status", "value"}})
	if err != nil {
		t.Fatalf("shapeValue returned error: %v", err)
	}

	wantRows := []map[string]any{
		{"name": "port1", "status": "up"},
		{"value": "orphan"},
	}
	if !reflect.DeepEqual(got.rows, wantRows) {
		t.Fatalf("shapeValue rows = %#v, want %#v", got.rows, wantRows)
	}
}

func TestWriteShapedTableWithQueryAndSelect(t *testing.T) {
	value := map[string]any{
		"results": []any{
			map[string]any{"name": "port1", "ip": "10.0.0.1", "status": "up"},
			map[string]any{"name": "port2", "ip": "10.0.0.2", "status": "down"},
		},
	}

	var out bytes.Buffer
	err := WriteShaped(&out, "table", value, ShapeOptions{
		Query:  ".results[*]",
		Select: []string{"name", "ip"},
	})
	if err != nil {
		t.Fatalf("WriteShaped returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("table output lines = %d, want 3\n%s", len(lines), out.String())
	}
	if !strings.Contains(lines[0], "name") || !strings.Contains(lines[0], "ip") {
		t.Fatalf("missing table headers in %q", lines[0])
	}
	if !strings.Contains(lines[1], "port1") || !strings.Contains(lines[1], "10.0.0.1") {
		t.Fatalf("missing first row values in %q", lines[1])
	}
	if !strings.Contains(lines[2], "port2") || !strings.Contains(lines[2], "10.0.0.2") {
		t.Fatalf("missing second row values in %q", lines[2])
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
		t.Fatalf("WriteShaped output differed without flags\nplain:\n%s\nshaped:\n%s", plain.String(), shaped.String())
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
	err := WriteShaped(&out, "json", value, ShapeOptions{
		Select: []string{"name"},
	})
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
