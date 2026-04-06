package cli

import "testing"

func TestSystemReadAliases(t *testing.T) {
	got := map[string]readAlias{}
	for _, alias := range systemReadAliases {
		got[alias.use] = alias
	}

	tests := []struct {
		name string
		path string
		kind string
	}{
		{name: "admins", path: "system/admin", kind: "cmdb"},
		{name: "dns", path: "system/dns", kind: "cmdb"},
		{name: "ntp", path: "system/ntp", kind: "cmdb"},
		{name: "vdoms", path: "system/vdom", kind: "cmdb"},
	}

	for _, tc := range tests {
		alias, ok := got[tc.name]
		if !ok {
			t.Fatalf("missing alias %q", tc.name)
		}
		if alias.path != tc.path {
			t.Fatalf("%s path = %q, want %q", tc.name, alias.path, tc.path)
		}
		if alias.kind != tc.kind {
			t.Fatalf("%s kind = %q, want %q", tc.name, alias.kind, tc.kind)
		}
	}
}

func TestSystemMonitorCompatibilitySpecs(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "status", path: "system/status"},
		{name: "interfaces", path: "system/interface"},
		{name: "ha-status", path: "system/ha-status"},
		{name: "license", path: "license/status"},
	}

	got := map[string]monitorEndpointSpec{}
	for _, spec := range systemMonitorCompatibilitySpecs() {
		got[spec.use] = spec
	}

	for _, tc := range tests {
		spec, ok := got[tc.name]
		if !ok {
			t.Fatalf("missing compatibility spec %q", tc.name)
		}
		if spec.path != tc.path {
			t.Fatalf("%s path = %q, want %q", tc.name, spec.path, tc.path)
		}
		canonical, ok := monitorEndpointSpecByUse(tc.name)
		if !ok {
			t.Fatalf("missing canonical monitor spec %q", tc.name)
		}
		if spec.capabilities != canonical.capabilities {
			t.Fatalf("%s capabilities = %v, want %v", tc.name, spec.capabilities, canonical.capabilities)
		}
	}
}

func TestReadFlagsSupportAllVDOMs(t *testing.T) {
	root := newRootCommand()
	tests := [][]string{
		{"cmdb", "get"},
		{"cmdb", "list"},
		{"monitor", "get"},
		{"raw", "get"},
	}

	for _, path := range tests {
		cmd, _, err := root.Find(path)
		if err != nil {
			t.Fatalf("Find(%v) error = %v", path, err)
		}
		if cmd.Flags().Lookup("all-vdoms") == nil {
			t.Fatalf("%v missing --all-vdoms flag", path)
		}
	}
}
