package cli

import (
	"fortigatecli/internal/output"

	"github.com/spf13/cobra"
)

func newRawCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use: "raw",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "get <api-path>",
		Short: "Perform a raw GET request against the FortiGate API",
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

			envelope, err := client.RawGet(ctx, args[0], readOpts.toAPIOptions())
			if err != nil {
				return err
			}

			return render(cmd, rootOpts.output, envelope)
		},
	})
	bindReadFlags(cmd.Commands()[0], readOpts)
	setDefaultStreams(cmd)
	return cmd
}
