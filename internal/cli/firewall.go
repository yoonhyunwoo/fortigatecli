package cli

import "github.com/spf13/cobra"

var firewallReadAliases = []readAlias{
	{use: "addresses", short: "List firewall addresses", path: "firewall/address", kind: "cmdb"},
	{use: "address-groups", short: "List firewall address groups", path: "firewall/addrgrp", kind: "cmdb"},
	{use: "policies", short: "List firewall policies", path: "firewall/policy", kind: "cmdb"},
	{use: "services", short: "List custom firewall services", path: "firewall.service/custom", kind: "cmdb"},
	{use: "service-groups", short: "List firewall service groups", path: "firewall.service/group", kind: "cmdb"},
	{use: "schedules-recurring", short: "List recurring firewall schedules", path: "firewall.schedule/recurring", kind: "cmdb"},
	{use: "schedules-onetime", short: "List one-time firewall schedules", path: "firewall.schedule/onetime", kind: "cmdb"},
}

func newFirewallCommand(rootOpts *rootOptions) *cobra.Command {
	firewallCmd := &cobra.Command{
		Use: "firewall",
	}

	for _, alias := range firewallReadAliases {
		firewallCmd.AddCommand(newReadAliasCommand(rootOpts, alias))
	}

	return firewallCmd
}
