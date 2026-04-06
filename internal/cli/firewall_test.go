package cli

import "testing"

func TestFirewallReadAliases(t *testing.T) {
	got := map[string]readAlias{}
	for _, alias := range firewallReadAliases {
		got[alias.use] = alias
	}

	tests := []struct {
		name string
		path string
		kind string
	}{
		{name: "addresses", path: "firewall/address", kind: "cmdb"},
		{name: "address-groups", path: "firewall/addrgrp", kind: "cmdb"},
		{name: "policies", path: "firewall/policy", kind: "cmdb"},
		{name: "services", path: "firewall.service/custom", kind: "cmdb"},
		{name: "service-groups", path: "firewall.service/group", kind: "cmdb"},
		{name: "schedules-recurring", path: "firewall.schedule/recurring", kind: "cmdb"},
		{name: "schedules-onetime", path: "firewall.schedule/onetime", kind: "cmdb"},
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
