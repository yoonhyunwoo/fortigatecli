package cli

import (
	"strings"
	"testing"
)

func TestRootCommandIncludesVPN(t *testing.T) {
	cmd := newRootCommand()
	if _, _, err := cmd.Find([]string{"vpn"}); err != nil {
		t.Fatalf("Find(vpn) error = %v", err)
	}
}

func TestVPNCommandTree(t *testing.T) {
	cmd := newVPNCommand(&rootOptions{})

	tests := []string{
		"ipsec",
		"ssl",
		"tunnels",
		"sessions",
		"settings",
	}
	for _, name := range tests {
		if _, _, err := cmd.Find([]string{name}); err != nil {
			t.Fatalf("Find(%s) error = %v", name, err)
		}
	}
}

func TestVPNTunnelCommandRequiresName(t *testing.T) {
	cmd := newVPNIPsecTunnelCommand(&rootOptions{})
	err := cmd.Args(cmd, nil)
	if err == nil {
		t.Fatal("expected argument validation error")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Fatalf("unexpected error: %v", err)
	}
}
