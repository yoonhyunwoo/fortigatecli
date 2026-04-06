package cli

import (
	"fmt"
	"slices"

	"fortigatecli/internal/output"

	"github.com/spf13/cobra"
)

type routingAlias struct {
	use   string
	short string
	path  string
	kind  string
}

var routingReadAliases = []routingAlias{
	{use: "table", short: "Fetch the IPv4 routing table", path: "router/ipv4", kind: "monitor"},
	{use: "routes", short: "Fetch the IPv4 routing table", path: "router/ipv4", kind: "monitor"},
	{use: "static", short: "List static routes", path: "router/static", kind: "cmdb"},
	{use: "static-routes", short: "List static routes", path: "router/static", kind: "cmdb"},
	{use: "interfaces", short: "List routing interface status", path: "system/interface", kind: "monitor"},
	{use: "interface-status", short: "List routing interface status", path: "system/interface", kind: "monitor"},
}

var routingDynamicMonitorPaths = map[string]string{
	"bgp": "router/bgp/neighbors",
}

func newRoutingCommand(rootOpts *rootOptions) *cobra.Command {
	routingCmd := &cobra.Command{
		Use: "routing",
	}

	for _, alias := range routingReadAliases {
		routingCmd.AddCommand(newRoutingAliasCommand(rootOpts, alias))
	}

	routingCmd.AddCommand(
		newRoutingDynamicCommand(rootOpts),
		newRoutingDynamicAliasCommand(rootOpts, "bgp"),
	)

	return routingCmd
}

func newRoutingAliasCommand(rootOpts *rootOptions, alias routingAlias) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   alias.use,
		Short: alias.short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRoutingRead(rootOpts, cmd, alias.kind, alias.path, readOpts)
		},
	}
	bindReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

func newRoutingDynamicCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   "dynamic <protocol>",
		Short: "Fetch dynamic routing protocol status",
		Args:  validateRoutingDynamicArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRoutingRead(rootOpts, cmd, "monitor", routingDynamicMonitorPaths[args[0]], readOpts)
		},
	}
	bindReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

func newRoutingDynamicAliasCommand(rootOpts *rootOptions, protocol string) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   protocol,
		Short: fmt.Sprintf("Fetch %s dynamic routing status", protocol),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRoutingRead(rootOpts, cmd, "monitor", routingDynamicMonitorPaths[protocol], readOpts)
		},
	}
	bindReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

func validateRoutingDynamicArgs(cmd *cobra.Command, args []string) error {
	if err := cobra.ExactArgs(1)(cmd, args); err != nil {
		return err
	}

	if _, ok := routingDynamicMonitorPaths[args[0]]; ok {
		return nil
	}

	return output.NewError(
		"validation_error",
		fmt.Sprintf("unsupported dynamic routing protocol %q", args[0]),
		supportedRoutingProtocols(),
	)
}

func supportedRoutingProtocols() []string {
	protocols := make([]string, 0, len(routingDynamicMonitorPaths))
	for protocol := range routingDynamicMonitorPaths {
		protocols = append(protocols, protocol)
	}
	slices.Sort(protocols)
	return protocols
}

func runRoutingRead(rootOpts *rootOptions, cmd *cobra.Command, kind string, resourcePath string, readOpts *readOptions) error {
	cfg, err := loadRuntimeConfig(rootOpts.vdom)
	if err != nil {
		return err
	}

	client, err := newClient(cfg)
	if err != nil {
		return output.NewError("client_error", err.Error(), nil)
	}

	ctx, cancel := commandContext()
	defer cancel()

	var envelope any
	switch kind {
	case "cmdb":
		envelope, err = client.GetCMDB(ctx, resourcePath, readOpts.toAPIOptions())
	default:
		envelope, err = client.GetMonitor(ctx, resourcePath, readOpts.toAPIOptions())
	}
	if err != nil {
		return err
	}

	return render(cmd, rootOpts.output, envelope)
}
