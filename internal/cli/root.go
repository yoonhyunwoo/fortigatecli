package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"fortigatecli/internal/config"
	"fortigatecli/internal/fortigate"
	"fortigatecli/internal/output"

	"github.com/spf13/cobra"
)

type rootOptions struct {
	output string
	vdom   string
}

func Execute() int {
	cmd := newRootCommand()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := cmd.ExecuteContext(ctx); err != nil {
		output.WriteError(normalizeError(err))
		return exitCode(err)
	}
	return 0
}

func newRootCommand() *cobra.Command {
	opts := &rootOptions{}

	cmd := &cobra.Command{
		Use:           "fortigatecli",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringVar(&opts.output, "output", "json", "output format: json or table")
	cmd.PersistentFlags().StringVar(&opts.vdom, "vdom", "", "override VDOM")

	cmd.AddCommand(
		newAuthCommand(opts),
		newLogsCommand(opts),
		newSystemCommand(opts),
		newRoutingCommand(opts),
		newFirewallCommand(opts),
		newCMDBCommand(opts),
		newMonitorCommand(opts),
		newDiscoveryCommand(opts),
		newVPNCommand(opts),
		newRawCommand(opts),
		newVersionCommand(),
	)

	return cmd
}

func loadRuntimeConfig(vdomOverride string) (config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return config.Config{}, err
	}
	if vdomOverride != "" {
		cfg.VDOM = vdomOverride
	}
	return cfg, nil
}

func newClient(cfg config.Config) (*fortigate.Client, error) {
	return fortigate.NewClient(fortigate.Config{
		BaseURL:  cfg.Host,
		Token:    cfg.Token,
		VDOM:     cfg.VDOM,
		Insecure: cfg.Insecure,
		Timeout:  cfg.Timeout,
	})
}

func commandContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

func render(cmd *cobra.Command, format string, value any) error {
	if err := output.Write(cmd.OutOrStdout(), format, value); err != nil {
		return output.NewError("output_error", err.Error(), nil)
	}
	return nil
}

func normalizeError(err error) error {
	var cfgErr *output.CLIError
	if errors.As(err, &cfgErr) {
		return cfgErr
	}

	if errors.Is(err, config.ErrNotConfigured) {
		path, pathErr := config.Path()
		if pathErr != nil {
			path = "~/.fortigatecli/config.yaml"
		}
		return output.NewError(
			"not_configured",
			fmt.Sprintf("configuration not found, run `fortigatecli auth init` first (%s)", path),
			nil,
		)
	}

	var apiErr *fortigate.APIError
	if errors.As(err, &apiErr) {
		return output.NewError("api_error", apiErr.Message, apiErr.Detail)
	}

	return output.NewError("command_error", err.Error(), nil)
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, config.ErrNotConfigured) {
		return 2
	}
	var apiErr *fortigate.APIError
	if errors.As(err, &apiErr) {
		return 3
	}
	return 1
}

func writeStdout(cmd *cobra.Command, data []byte) error {
	if _, err := cmd.OutOrStdout().Write(data); err != nil {
		return output.NewError("write_error", err.Error(), nil)
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		if _, err := fmt.Fprintln(cmd.OutOrStdout()); err != nil {
			return output.NewError("write_error", err.Error(), nil)
		}
	}
	return nil
}

func setDefaultStreams(cmd *cobra.Command) {
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
}
