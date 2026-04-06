package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCMDBCommandShape(t *testing.T) {
	cmd := newCMDBCommand(&rootOptions{})
	subcommands := map[string]*cobra.Command{}
	for _, subcommand := range cmd.Commands() {
		subcommands[subcommand.Name()] = subcommand
	}

	for _, name := range []string{"get", "list", "show", "address", "policy", "addrgrp", "service"} {
		if _, ok := subcommands[name]; !ok {
			t.Fatalf("missing cmdb subcommand %q", name)
		}
	}

	service := subcommands["service"]
	custom, _, err := service.Find([]string{"custom"})
	if err != nil {
		t.Fatalf("service custom lookup failed: %v", err)
	}
	if custom == nil || custom.Name() != "custom" {
		t.Fatalf("service custom command = %#v", custom)
	}

	address := subcommands["address"]
	for _, name := range []string{"list", "get"} {
		if found := address.Commands(); len(found) == 0 {
			t.Fatalf("address aliases missing subcommands")
		}
		if _, _, err := address.Find([]string{name}); err != nil {
			t.Fatalf("address %s lookup failed: %v", name, err)
		}
	}
}

func TestCMDBHelpIncludesExamplesAndPaging(t *testing.T) {
	output := executeHelp(t, "cmdb", "--help")
	for _, needle := range []string{
		"fortigatecli cmdb show firewall/address branch-office",
		"address get branch-office",
		"raw get /api/v2/cmdb/firewall/address/branch-office",
	} {
		if !strings.Contains(output, needle) {
			t.Fatalf("help missing %q\n%s", needle, output)
		}
	}

	listHelp := executeHelp(t, "cmdb", "list", "--help")
	for _, needle := range []string{"--page-size", "--page", "--all"} {
		if !strings.Contains(listHelp, needle) {
			t.Fatalf("list help missing %q\n%s", needle, listHelp)
		}
	}
}

func executeHelp(t *testing.T, args ...string) string {
	t.Helper()

	cmd := newRootCommand()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	setCommandOutput(cmd, &stdout, &stderr)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	return stdout.String() + stderr.String()
}

func setCommandOutput(cmd *cobra.Command, stdout *bytes.Buffer, stderr *bytes.Buffer) {
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	for _, child := range cmd.Commands() {
		setCommandOutput(child, stdout, stderr)
	}
}
