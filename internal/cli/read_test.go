package cli

import (
	"strings"
	"testing"
)

func TestReadOptionsToMonitorAPIOptionsTranslatesShortcutFilters(t *testing.T) {
	spec, ok := monitorEndpointSpecByUse("interfaces")
	if !ok {
		t.Fatal("missing monitor spec for interfaces")
	}

	opts := newReadOptions()
	opts.eqFilters = []string{"name=port1"}
	opts.neFilters = []string{"status=down"}
	opts.contains = []string{"alias=wan"}
	opts.prefix = []string{"name=port"}

	got, err := opts.toMonitorAPIOptions(&spec)
	if err != nil {
		t.Fatalf("toMonitorAPIOptions() error = %v", err)
	}

	want := []string{"name==port1", "status!=down", "alias=@wan", "name=@port*"}
	if len(got.Filters) != len(want) {
		t.Fatalf("filter count = %d, want %d", len(got.Filters), len(want))
	}
	for i := range want {
		if got.Filters[i] != want[i] {
			t.Fatalf("filter[%d] = %q, want %q", i, got.Filters[i], want[i])
		}
	}
}

func TestReadOptionsToMonitorAPIOptionsRejectsUnsupportedAliasOptions(t *testing.T) {
	spec, ok := monitorEndpointSpecByUse("license")
	if !ok {
		t.Fatal("missing monitor spec for license")
	}

	opts := newReadOptions()
	opts.sort = []string{"name"}

	_, err := opts.toMonitorAPIOptions(&spec)
	if err == nil {
		t.Fatal("toMonitorAPIOptions() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "does not support --sort") {
		t.Fatalf("error = %v", err)
	}
}

func TestReadOptionsToMonitorAPIOptionsAllowsRawMonitorPaths(t *testing.T) {
	opts := newReadOptions()
	opts.sort = []string{"name"}
	opts.eqFilters = []string{"name=port1"}

	got, err := opts.toMonitorAPIOptions(nil)
	if err != nil {
		t.Fatalf("toMonitorAPIOptions() error = %v", err)
	}
	if len(got.Sort) != 1 || got.Sort[0] != "name" {
		t.Fatalf("sort = %#v", got.Sort)
	}
	if len(got.Filters) != 1 || got.Filters[0] != "name==port1" {
		t.Fatalf("filters = %#v", got.Filters)
	}
}

func TestReadOptionsToMonitorAPIOptionsRejectsInvalidShortcutSyntax(t *testing.T) {
	spec, ok := monitorEndpointSpecByUse("interfaces")
	if !ok {
		t.Fatal("missing monitor spec for interfaces")
	}

	opts := newReadOptions()
	opts.eqFilters = []string{"name"}

	_, err := opts.toMonitorAPIOptions(&spec)
	if err == nil {
		t.Fatal("toMonitorAPIOptions() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "--eq expects field=value") {
		t.Fatalf("error = %v", err)
	}
}
