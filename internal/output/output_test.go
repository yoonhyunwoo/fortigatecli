package output

import (
	"bytes"
	"strings"
	"testing"
)

type tableEnvelope struct {
	Results []map[string]any `json:"results"`
}

func TestWriteTableUsesResultsRows(t *testing.T) {
	buf := &bytes.Buffer{}
	err := Write(buf, "table", tableEnvelope{
		Results: []map[string]any{
			{"srcip": "10.0.0.1", "dstip": "10.0.0.2"},
			{"srcip": "10.0.0.3", "dstip": "10.0.0.4"},
		},
	})
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	output := buf.String()
	for _, expected := range []string{"dstip", "srcip", "10.0.0.1", "10.0.0.4"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("table output missing %q: %q", expected, output)
		}
	}
}

func TestWriteTableHandlesEmptyResults(t *testing.T) {
	buf := &bytes.Buffer{}
	err := Write(buf, "table", tableEnvelope{Results: []map[string]any{}})
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if got := strings.TrimSpace(buf.String()); got != "no results" {
		t.Fatalf("output = %q, want no results", got)
	}
}
