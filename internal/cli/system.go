package cli

import (
	"fortigatecli/internal/output"

	"github.com/spf13/cobra"
)

type systemAlias struct {
	use   string
	short string
	path  string
	kind  string
}

var systemReadAliases = []systemAlias{
	{use: "interfaces", short: "List system interfaces", path: "system/interface", kind: "monitor"},
	{use: "vdoms", short: "List configured VDOMs", path: "system/vdom", kind: "cmdb"},
	{use: "ha-status", short: "Fetch HA status", path: "system/ha-status", kind: "monitor"},
	{use: "license", short: "Fetch license status", path: "license/status", kind: "monitor"},
}

func newSystemCommand(rootOpts *rootOptions) *cobra.Command {
	systemCmd := &cobra.Command{
		Use: "system",
	}

	systemCmd.AddCommand(
		newSystemStatusCommand(rootOpts),
		newSystemBackupCommand(rootOpts),
	)
	for _, alias := range systemReadAliases {
		systemCmd.AddCommand(newSystemAliasCommand(rootOpts, alias))
	}
	return systemCmd
}

func newSystemStatusCommand(rootOpts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Fetch system status",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			envelope, err := client.Test(ctx)
			if err != nil {
				return err
			}

			return render(cmd, rootOpts.output, envelope)
		},
	}
	setDefaultStreams(cmd)
	return cmd
}

func newSystemBackupCommand(rootOpts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Print system config backup to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			data, err := client.Backup(ctx)
			if err != nil {
				return err
			}

			return writeStdout(cmd, data)
		},
	}
	setDefaultStreams(cmd)
	return cmd
}

func newSystemAliasCommand(rootOpts *rootOptions, alias systemAlias) *cobra.Command {
	readOpts := newReadOptions()
	shapeOpts := newShapeOptions()
	cmd := &cobra.Command{
		Use:   alias.use,
		Short: alias.short,
		RunE: func(cmd *cobra.Command, args []string) error {
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
			switch alias.kind {
			case "cmdb":
				envelope, err = client.GetCMDB(ctx, alias.path, readOpts.toAPIOptions())
			default:
				envelope, err = client.GetMonitor(ctx, alias.path, readOpts.toAPIOptions())
			}
			if err != nil {
				return err
			}

			return renderRead(cmd, rootOpts.output, envelope, shapeOpts)
		},
	}
	bindReadFlags(cmd, readOpts)
	bindShapeFlags(cmd, shapeOpts)
	setDefaultStreams(cmd)
	return cmd
}
