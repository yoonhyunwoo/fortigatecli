package cli

import (
	"strings"
	"testing"
)

func TestRoutingReadAliases(t *testing.T) {
	got := map[string]routingAlias{}
	for _, alias := range routingReadAliases {
		got[alias.use] = alias
	}

	tests := []struct {
		name string
		path string
		kind string
	}{
		{name: "table", path: "router/ipv4", kind: "monitor"},
		{name: "routes", path: "router/ipv4", kind: "monitor"},
		{name: "static", path: "router/static", kind: "cmdb"},
		{name: "static-routes", path: "router/static", kind: "cmdb"},
		{name: "interfaces", path: "system/interface", kind: "monitor"},
		{name: "interface-status", path: "system/interface", kind: "monitor"},
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

func TestRoutingDynamicProtocolMap(t *testing.T) {
	if got := routingDynamicMonitorPaths["bgp"]; got != "router/bgp/neighbors" {
		t.Fatalf("bgp path = %q, want %q", got, "router/bgp/neighbors")
	}

	cmd := newRoutingDynamicCommand(&rootOptions{})
	err := cmd.Args(cmd, []string{"rip"})
	if err == nil {
		t.Fatal("expected validation error for unsupported protocol")
	}
	if !strings.Contains(err.Error(), `unsupported dynamic routing protocol "rip"`) {
		t.Fatalf("unexpected validation error: %v", err)
	}

	if err := cmd.Args(cmd, []string{"bgp"}); err != nil {
		t.Fatalf("expected bgp to be accepted: %v", err)
	}

	if err := cmd.Args(cmd, nil); err == nil {
		t.Fatal("expected argument count validation error")
	}
}
