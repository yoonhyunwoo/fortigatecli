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
	{use: "vdoms", short: "List configured VDOMs", path: "system/vdom", kind: "cmdb"},
}

func newSystemCommand(rootOpts *rootOptions) *cobra.Command {
	systemCmd := &cobra.Command{
		Use: "system",
	}

	systemCmd.AddCommand(newSystemBackupCommand(rootOpts))
	for _, spec := range systemMonitorCompatibilitySpecs() {
		systemCmd.AddCommand(newSystemMonitorAliasCommand(rootOpts, spec))
	}
	for _, alias := range systemReadAliases {
		systemCmd.AddCommand(newSystemAliasCommand(rootOpts, alias))
	}
	return systemCmd
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

			envelope, err := client.GetCMDB(ctx, alias.path, readOpts.toAPIOptions())
			if err != nil {
				return err
			}

			return render(cmd, rootOpts.output, envelope)
		},
	}
	bindReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

func newSystemMonitorAliasCommand(rootOpts *rootOptions, spec monitorEndpointSpec) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   spec.use,
		Short: spec.short,
		RunE: func(cmd *cobra.Command, args []string) error {
			options, err := monitorOptionsOrError(readOpts, &spec)
			if err != nil {
				return err
			}
			return runMonitor(rootOpts, cmd, spec.path, options)
		},
	}
	bindMonitorReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}
