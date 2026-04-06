package cli

import (
	"context"

	"fortigatecli/internal/fortigate"
	"fortigatecli/internal/output"

	"github.com/spf13/cobra"
)

type logsAlias struct {
	use           string
	short         string
	path          string
	kind          string
	supportsWatch bool
	supportsTable bool
}

var logsCommandAliases = map[string][]logsAlias{
	"traffic": {
		{use: "list", short: "List forward traffic logs", path: "disk/traffic/forward/system", kind: "log", supportsWatch: true, supportsTable: true},
	},
	"event": {
		{use: "list", short: "List event logs", path: "disk/event/system", kind: "log", supportsWatch: true, supportsTable: true},
	},
	"utm": {
		{use: "list", short: "List UTM webfilter logs", path: "disk/utm/webfilter", kind: "log", supportsWatch: true, supportsTable: true},
	},
	"system": {
		{use: "list", short: "List system event logs", path: "disk/event/system", kind: "log", supportsWatch: true, supportsTable: true},
	},
	"session": {
		{use: "list", short: "List active firewall sessions", path: "session-top", kind: "session", supportsWatch: true, supportsTable: true},
	},
	"performance": {
		{use: "cpu", short: "Watch CPU resource usage", path: "system/resource/usage?resource=cpu", kind: "performance", supportsWatch: true, supportsTable: true},
		{use: "memory", short: "Watch memory resource usage", path: "system/resource/usage?resource=memory", kind: "performance", supportsWatch: true, supportsTable: true},
		{use: "sessions", short: "Watch session resource usage", path: "system/resource/usage?resource=session", kind: "performance", supportsWatch: true, supportsTable: true},
		{use: "session-rate", short: "Watch session setup rate", path: "system/resource/usage?resource=session_setup_rate", kind: "performance", supportsWatch: true, supportsTable: true},
		{use: "npu-sessions", short: "Watch NPU session resource usage", path: "system/resource/usage?resource=npu_session&scope=global", kind: "performance", supportsWatch: true, supportsTable: true},
	},
}

func newLogsCommand(rootOpts *rootOptions) *cobra.Command {
	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "Logs and observability commands",
	}

	logsCmd.AddCommand(
		newLogsCategoryCommand(rootOpts, "traffic", "Traffic log reads"),
		newLogsCategoryCommand(rootOpts, "event", "Event log reads"),
		newLogsCategoryCommand(rootOpts, "utm", "UTM log reads"),
		newLogsCategoryCommand(rootOpts, "system", "System log reads"),
		newLogsCategoryCommand(rootOpts, "session", "Session observability reads"),
		newLogsCategoryCommand(rootOpts, "performance", "Performance observability reads"),
	)
	return logsCmd
}

func newLogsCategoryCommand(rootOpts *rootOptions, use string, short string) *cobra.Command {
	categoryCmd := &cobra.Command{
		Use:   use,
		Short: short,
	}
	for _, alias := range logsCommandAliases[use] {
		categoryCmd.AddCommand(newLogsAliasCommand(rootOpts, alias))
	}
	return categoryCmd
}

func newLogsAliasCommand(rootOpts *rootOptions, alias logsAlias) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   alias.use,
		Short: alias.short,
		RunE: func(cmd *cobra.Command, args []string) error {
			if rootOpts.output == "table" && !alias.supportsTable {
				return output.NewError("unsupported_output", "table output is not supported for this command", nil)
			}
			if readOpts.watchEnabled() && !alias.supportsWatch {
				return output.NewError("unsupported_watch", "watch mode is not supported for this command", nil)
			}

			cfg, err := loadRuntimeConfig(rootOpts.vdom)
			if err != nil {
				return err
			}

			client, err := newClient(cfg)
			if err != nil {
				return output.NewError("client_error", err.Error(), nil)
			}

			return runRead(cmd, rootOpts.output, readOpts.watchEnabled(), readOpts.interval, func(ctx context.Context) (*fortigate.Envelope, error) {
				return executeLogsRead(ctx, client, alias, readOpts.toAPIOptions())
			})
		},
	}
	bindObservabilityReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

func executeLogsRead(ctx context.Context, client *fortigate.Client, alias logsAlias, options fortigate.ReadOptions) (*fortigate.Envelope, error) {
	switch alias.kind {
	case "session":
		return client.GetSession(ctx, alias.path, options)
	case "performance":
		return client.GetPerformance(ctx, alias.path, options)
	default:
		return client.GetLog(ctx, alias.path, options)
	}
}
