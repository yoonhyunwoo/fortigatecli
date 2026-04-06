package cli

import "github.com/spf13/cobra"

func newMonitorCommand(rootOpts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use: "monitor",
	}

	cmd.AddCommand(
		newMonitorPathCommand(rootOpts, "get", "Get a monitor resource"),
		newMonitorPathCommand(rootOpts, "list", "List a monitor resource"),
	)
	for _, spec := range monitorEndpointSpecs {
		cmd.AddCommand(newMonitorAliasCommand(rootOpts, spec))
	}

	setDefaultStreams(cmd)
	return cmd
}
