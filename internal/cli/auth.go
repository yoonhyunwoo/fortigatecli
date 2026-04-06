package cli

import (
	"strings"
	"time"

	"fortigatecli/internal/config"
	"fortigatecli/internal/output"

	"github.com/spf13/cobra"
)

func newAuthCommand(rootOpts *rootOptions) *cobra.Command {
	authCmd := &cobra.Command{
		Use: "auth",
	}

	authCmd.AddCommand(
		newAuthInitCommand(),
		newAuthShowCommand(rootOpts),
		newAuthTestCommand(rootOpts),
	)
	return authCmd
}

func newAuthInitCommand() *cobra.Command {
	var cfg config.Config
	var timeout string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Save FortiGate access settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !strings.HasPrefix(cfg.Host, "https://") && !strings.HasPrefix(cfg.Host, "http://") {
				return output.NewError("validation_error", "host must include http:// or https://", nil)
			}

			if timeout != "" {
				parsed, err := time.ParseDuration(timeout)
				if err != nil {
					return output.NewError("validation_error", "invalid timeout duration", err.Error())
				}
				cfg.Timeout = parsed
			}

			if cfg.VDOM == "" {
				cfg.VDOM = "root"
			}
			if cfg.Timeout == 0 {
				cfg.Timeout = 10 * time.Second
			}

			if err := config.Save(cfg); err != nil {
				return output.NewError("config_error", err.Error(), nil)
			}

			savedPath, _ := config.Path()
			return render(cmd, "json", map[string]any{
				"saved": true,
				"path":  savedPath,
				"host":  cfg.Host,
				"vdom":  cfg.VDOM,
			})
		},
	}

	cmd.Flags().StringVar(&cfg.Host, "host", "", "FortiGate base URL")
	cmd.Flags().StringVar(&cfg.Token, "token", "", "FortiGate API token")
	cmd.Flags().StringVar(&cfg.VDOM, "vdom", "root", "default VDOM")
	cmd.Flags().BoolVar(&cfg.Insecure, "insecure", true, "skip TLS certificate verification")
	cmd.Flags().StringVar(&timeout, "timeout", "10s", "HTTP timeout")
	_ = cmd.MarkFlagRequired("host")
	_ = cmd.MarkFlagRequired("token")
	setDefaultStreams(cmd)

	return cmd
}

func newAuthShowCommand(rootOpts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show current auth configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadRuntimeConfig(rootOpts.vdom)
			if err != nil {
				return err
			}

			maskedToken := ""
			if len(cfg.Token) > 8 {
				maskedToken = cfg.Token[:4] + strings.Repeat("*", len(cfg.Token)-8) + cfg.Token[len(cfg.Token)-4:]
			} else if cfg.Token != "" {
				maskedToken = "********"
			}

			return render(cmd, rootOpts.output, map[string]any{
				"host":     cfg.Host,
				"token":    maskedToken,
				"vdom":     cfg.VDOM,
				"insecure": cfg.Insecure,
				"timeout":  cfg.Timeout.String(),
			})
		},
	}
	setDefaultStreams(cmd)
	return cmd
}

func newAuthTestCommand(rootOpts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Verify current auth settings",
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

			return render(cmd, rootOpts.output, map[string]any{
				"ok":          true,
				"host":        cfg.Host,
				"vdom":        cfg.VDOM,
				"http_status": envelope.HTTPStatus,
				"status":      envelope.Status,
				"version":     envelope.Version,
				"serial":      envelope.Serial,
			})
		},
	}
	setDefaultStreams(cmd)
	return cmd
}
