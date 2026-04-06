package cli

import (
	"context"
	"fortigatecli/internal/fortigate"
	"fortigatecli/internal/output"

	"github.com/spf13/cobra"
)

func newVPNCommand(rootOpts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vpn",
		Short: "Read VPN status and configuration",
	}

	cmd.AddCommand(
		newVPNIPsecCommand(rootOpts),
		newVPNSSLCommand(rootOpts),
		newVPNTunnelsCommand(rootOpts),
		newVPNSessionsCommand(rootOpts),
		newVPNSettingsCommand(rootOpts),
	)
	setDefaultStreams(cmd)
	return cmd
}

func newVPNIPsecCommand(rootOpts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ipsec",
		Short: "Read IPsec status and tunnel details",
	}
	cmd.AddCommand(
		newVPNIPsecStatusCommand(rootOpts),
		newVPNIPsecTunnelsCommand(rootOpts),
		newVPNIPsecTunnelCommand(rootOpts),
	)
	setDefaultStreams(cmd)
	return cmd
}

func newVPNSSLCommand(rootOpts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssl",
		Short: "Read SSL-VPN settings and sessions",
	}
	cmd.AddCommand(
		newVPNSSLSettingsCommand(rootOpts),
		newVPNSSLSessionsCommand(rootOpts),
	)
	setDefaultStreams(cmd)
	return cmd
}

func newVPNIPsecStatusCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Fetch IPsec runtime status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPNMonitor(rootOpts, cmd, readOpts, func(client vpnClient, opts vpnReadOptions) (any, error) {
				return client.GetVPNIPsecStatus(cmd.Context(), opts)
			})
		},
	}
	bindReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

func newVPNIPsecTunnelsCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   "tunnels",
		Short: "List IPsec tunnels",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPNMonitor(rootOpts, cmd, readOpts, func(client vpnClient, opts vpnReadOptions) (any, error) {
				return client.ListVPNIPsecTunnels(cmd.Context(), opts)
			})
		},
	}
	bindReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

func newVPNIPsecTunnelCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   "tunnel <name>",
		Short: "Fetch monitor-based status detail for an IPsec tunnel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPNMonitor(rootOpts, cmd, readOpts, func(client vpnClient, opts vpnReadOptions) (any, error) {
				return client.GetVPNIPsecTunnel(cmd.Context(), args[0], opts)
			})
		},
	}
	bindReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

func newVPNSSLSettingsCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   "settings",
		Short: "Fetch SSL-VPN settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPNMonitor(rootOpts, cmd, readOpts, func(client vpnClient, opts vpnReadOptions) (any, error) {
				return client.GetSSLVPNSettings(cmd.Context(), opts)
			})
		},
	}
	bindReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

func newVPNSSLSessionsCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   "sessions",
		Short: "List active SSL-VPN sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPNMonitor(rootOpts, cmd, readOpts, func(client vpnClient, opts vpnReadOptions) (any, error) {
				return client.ListSSLVPNSessions(cmd.Context(), opts)
			})
		},
	}
	bindReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

func newVPNTunnelsCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   "tunnels",
		Short: "Shortcut for vpn ipsec tunnels",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPNMonitor(rootOpts, cmd, readOpts, func(client vpnClient, opts vpnReadOptions) (any, error) {
				return client.ListVPNIPsecTunnels(cmd.Context(), opts)
			})
		},
	}
	bindReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

func newVPNSessionsCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   "sessions",
		Short: "Shortcut for vpn ssl sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPNMonitor(rootOpts, cmd, readOpts, func(client vpnClient, opts vpnReadOptions) (any, error) {
				return client.ListSSLVPNSessions(cmd.Context(), opts)
			})
		},
	}
	bindReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

func newVPNSettingsCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   "settings",
		Short: "Shortcut for vpn ssl settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPNMonitor(rootOpts, cmd, readOpts, func(client vpnClient, opts vpnReadOptions) (any, error) {
				return client.GetSSLVPNSettings(cmd.Context(), opts)
			})
		},
	}
	bindReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

type vpnReadOptions = fortigate.ReadOptions

type vpnClient interface {
	GetVPNIPsecStatus(ctx context.Context, options vpnReadOptions) (*fortigate.Envelope, error)
	ListVPNIPsecTunnels(ctx context.Context, options vpnReadOptions) (*fortigate.Envelope, error)
	GetVPNIPsecTunnel(ctx context.Context, tunnelName string, options vpnReadOptions) (*fortigate.Envelope, error)
	GetSSLVPNSettings(ctx context.Context, options vpnReadOptions) (*fortigate.Envelope, error)
	ListSSLVPNSessions(ctx context.Context, options vpnReadOptions) (*fortigate.Envelope, error)
}

func runVPNMonitor(rootOpts *rootOptions, cmd *cobra.Command, readOpts *readOptions, run func(vpnClient, vpnReadOptions) (any, error)) error {
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
	cmd.SetContext(ctx)

	envelope, err := run(client, readOpts.toAPIOptions())
	if err != nil {
		return err
	}

	return render(cmd, rootOpts.output, envelope)
}
