package cli

import "testing"

func TestSystemReadAliases(t *testing.T) {
	got := map[string]systemAlias{}
	for _, alias := range systemReadAliases {
		got[alias.use] = alias
	}

	alias, ok := got["vdoms"]
	if !ok {
		t.Fatal("missing alias \"vdoms\"")
	}
	if alias.path != "system/vdom" {
		t.Fatalf("vdoms path = %q", alias.path)
	}
	if alias.kind != "cmdb" {
		t.Fatalf("vdoms kind = %q", alias.kind)
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
