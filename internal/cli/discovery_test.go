package cli

import (
	"slices"
	"testing"
)

func TestRootRegistersDiscoveryCommand(t *testing.T) {
	root := newRootCommand()
	discovery, _, err := root.Find([]string{"discovery"})
	if err != nil {
		t.Fatalf("Find(discovery) error = %v", err)
	}
	if discovery == nil || discovery.Name() != "discovery" {
		t.Fatalf("discovery command = %#v", discovery)
	}
}

func TestParseDiscoveryTargetRejectsUnknownValue(t *testing.T) {
	if _, err := parseDiscoveryTarget("raw"); err == nil {
		t.Fatal("parseDiscoveryTarget(raw) error = nil, want error")
	}
}

func TestDiscoverySchemaFlagsAreRestricted(t *testing.T) {
	cmd := newDiscoverySchemaCommand(&rootOptions{})
	if cmd.Flags().Lookup("with-meta") == nil {
		t.Fatal("with-meta flag missing")
	}
	for _, disallowed := range []string{"filter", "count", "datasource", "field", "format", "sort", "start"} {
		if cmd.Flags().Lookup(disallowed) != nil {
			t.Fatalf("unexpected flag %q", disallowed)
		}
	}
}

func TestDiscoveryFieldsFlagsAreRestricted(t *testing.T) {
	cmd := newDiscoveryFieldsCommand(&rootOptions{})
	wantFlags := []string{"filter", "count", "with-meta", "datasource"}
	gotFlags := make([]string, 0, len(wantFlags))
	for _, name := range wantFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("missing flag %q", name)
		}
		gotFlags = append(gotFlags, name)
	}
	if !slices.Equal(gotFlags, wantFlags) {
		t.Fatalf("flags = %#v, want %#v", gotFlags, wantFlags)
	}
	for _, disallowed := range []string{"field", "format", "sort", "start"} {
		if cmd.Flags().Lookup(disallowed) != nil {
			t.Fatalf("unexpected flag %q", disallowed)
		}
	}
}

func TestDiscoveryCapabilitiesOnlySupportsProbeFlag(t *testing.T) {
	cmd := newDiscoveryCapabilitiesCommand(&rootOptions{})
	if cmd.Flags().Lookup("probe") == nil {
		t.Fatal("probe flag missing")
	}
	for _, disallowed := range []string{"filter", "count", "datasource", "field", "format", "sort", "start", "with-meta"} {
		if cmd.Flags().Lookup(disallowed) != nil {
			t.Fatalf("unexpected flag %q", disallowed)
		}
	}
}

func TestDiscoveryExamplesMatchContract(t *testing.T) {
	tests := map[string]string{
		"schema":       "fortigatecli discovery schema cmdb firewall/address --with-meta",
		"fields":       "fortigatecli discovery fields monitor system/interface --filter name==port1 --count 5",
		"capabilities": "fortigatecli discovery capabilities cmdb firewall/address --probe",
	}

	root := newDiscoveryCommand(&rootOptions{})
	for name, want := range tests {
		cmd, _, err := root.Find([]string{name})
		if err != nil {
			t.Fatalf("Find(%s) error = %v", name, err)
		}
		if cmd.Example != want {
			t.Fatalf("%s example = %q, want %q", name, cmd.Example, want)
		}
	}
}
