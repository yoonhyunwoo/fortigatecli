package cli

import (
	"fortigatecli/internal/output"

	"github.com/spf13/cobra"
)

func newMonitorCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	shapeOpts := newShapeOptions()
	cmd := &cobra.Command{
		Use: "monitor",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "get <path>",
			Short: "Get a monitor resource",
			Args:  cobra.ExactArgs(1),
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
				if readOpts.allVDOMs {
					envelope, err := client.GetMonitorAcrossVDOMs(ctx, args[0], readOpts.toAPIOptions())
					if err != nil {
						return err
					}
					return render(cmd, rootOpts.output, envelope)
				}
				envelope, err := client.GetMonitor(ctx, args[0], readOpts.toAPIOptions())
				if err != nil {
					return err
				}
				return renderRead(cmd, rootOpts.output, envelope, shapeOpts)
			},
		},
		newMonitorPathCommand(rootOpts, "list", "List a monitor resource"),
	)
	for _, spec := range monitorEndpointSpecs {
		cmd.AddCommand(newMonitorAliasCommand(rootOpts, spec))
	}

	bindReadFlags(cmd.Commands()[0], readOpts)
	bindShapeFlags(cmd.Commands()[0], shapeOpts)

	setDefaultStreams(cmd)
	return cmd
}
