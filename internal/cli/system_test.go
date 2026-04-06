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
		{name: "interfaces", path: "system/interface", kind: "monitor"},
		{name: "vdoms", path: "system/vdom", kind: "cmdb"},
		{name: "ha-status", path: "system/ha-status", kind: "monitor"},
		{name: "license", path: "license/status", kind: "monitor"},
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
