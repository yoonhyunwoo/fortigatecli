package cli

import "github.com/spf13/cobra"

const version = "0.1.0"

func newVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print fortigatecli version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return render(cmd, "json", map[string]any{
				"version": version,
			})
		},
	}
	setDefaultStreams(cmd)
	return cmd
}
