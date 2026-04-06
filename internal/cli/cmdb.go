package cli

import (
	"fortigatecli/internal/output"

	"github.com/spf13/cobra"
)

func newCMDBCommand(rootOpts *rootOptions) *cobra.Command {
	cmdbCmd := &cobra.Command{
		Use: "cmdb",
	}

	cmdbCmd.AddCommand(
		newCMDBGetCommand(rootOpts),
		newCMDBListCommand(rootOpts),
	)
	return cmdbCmd
}

func newCMDBGetCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   "get <path>",
		Short: "Get a CMDB resource",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCMDB(rootOpts, cmd, args[0], readOpts)
		},
	}
	bindReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

func newCMDBListCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   "list <path>",
		Short: "List CMDB resources",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCMDB(rootOpts, cmd, args[0], readOpts)
		},
	}
	bindReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

func runCMDB(rootOpts *rootOptions, cmd *cobra.Command, resourcePath string, readOpts *readOptions) error {
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

	envelope, err := client.GetCMDB(ctx, resourcePath, readOpts.toAPIOptions())
	if err != nil {
		return err
	}

	return render(cmd, rootOpts.output, envelope)
}
