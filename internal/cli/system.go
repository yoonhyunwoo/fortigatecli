package cli

import (
	"fmt"

	"fortigatecli/internal/fortigate"
	"fortigatecli/internal/output"

	"github.com/spf13/cobra"
)

var systemReadAliases = []readAlias{
	{use: "admins", short: "List configured admin users", path: "system/admin", kind: "cmdb"},
	{use: "dns", short: "Fetch DNS configuration", path: "system/dns", kind: "cmdb"},
	{use: "ntp", short: "Fetch NTP configuration", path: "system/ntp", kind: "cmdb"},
	{use: "vdoms", short: "List configured VDOMs", path: "system/vdom", kind: "cmdb"},
}

func newSystemCommand(rootOpts *rootOptions) *cobra.Command {
	systemCmd := &cobra.Command{
		Use: "system",
	}

	systemCmd.AddCommand(
		newSystemStatusCommand(rootOpts),
		newSystemBackupCommand(rootOpts),
		newSystemHostnameCommand(rootOpts),
		newSystemFirmwareCommand(rootOpts),
		newSystemInterfaceCommand(rootOpts),
		newSystemHAPeersCommand(rootOpts),
	)
	for _, spec := range systemMonitorCompatibilitySpecs() {
		systemCmd.AddCommand(newSystemMonitorAliasCommand(rootOpts, spec))
	}
	for _, alias := range systemReadAliases {
		systemCmd.AddCommand(newReadAliasCommand(rootOpts, alias))
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

func newSystemHostnameCommand(rootOpts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hostname",
		Short: "Fetch the FortiGate hostname",
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

			hostname, err := systemHostnameValue(envelope)
			if err != nil {
				return err
			}
			return render(cmd, rootOpts.output, hostname)
		},
	}
	setDefaultStreams(cmd)
	return cmd
}

func newSystemFirmwareCommand(rootOpts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "firmware",
		Short: "Fetch firmware summary",
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

			return render(cmd, rootOpts.output, systemFirmwareValue(envelope))
		},
	}
	setDefaultStreams(cmd)
	return cmd
}

func newSystemInterfaceCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   "interface <name>",
		Args:  cobra.ExactArgs(1),
		Short: "Fetch a single system interface",
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

			apiOpts := readOpts.toAPIOptions()
			apiOpts.Filters = append([]string{fmt.Sprintf("name==%s", args[0])}, apiOpts.Filters...)

			envelope, err := client.GetMonitor(ctx, "system/interface", apiOpts)
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

func newSystemHAPeersCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   "ha-peers",
		Short: "Fetch HA peers summary",
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

			envelope, err := client.GetMonitor(ctx, "system/ha-status", readOpts.toAPIOptions())
			if err != nil {
				return err
			}

			return render(cmd, rootOpts.output, systemHAPeersValue(envelope))
		},
	}
	bindReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

func systemHostnameValue(envelope *fortigate.Envelope) (map[string]any, error) {
	results, ok := envelope.Results.(map[string]any)
	if !ok {
		return nil, output.NewError("output_error", "system status response did not contain object results", nil)
	}

	hostname, ok := results["hostname"]
	if !ok {
		return nil, output.NewError("output_error", "system status response did not contain hostname", nil)
	}

	return map[string]any{"hostname": hostname}, nil
}

func systemFirmwareValue(envelope *fortigate.Envelope) map[string]any {
	return map[string]any{
		"version":  envelope.Version,
		"build":    envelope.Build,
		"revision": envelope.Revision,
		"serial":   envelope.Serial,
	}
}

type systemHAPeersResponse struct {
	Role  string `json:"role,omitempty"`
	Peers any    `json:"peers,omitempty"`
}

func systemHAPeersValue(envelope *fortigate.Envelope) any {
	results, ok := envelope.Results.(map[string]any)
	if !ok {
		return envelope.Results
	}

	resp := systemHAPeersResponse{}
	if role, ok := firstString(results, "role", "ha_role", "cluster_role", "state"); ok {
		resp.Role = role
	}
	if peers, ok := firstValue(results, "peers", "peer", "peer_list", "members", "nodes"); ok {
		resp.Peers = peers
	}

	if resp.Role == "" && resp.Peers == nil {
		return envelope.Results
	}
	return resp
}

func firstString(values map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		str, ok := value.(string)
		if ok && str != "" {
			return str, true
		}
	}
	return "", false
}

func firstValue(values map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		value, ok := values[key]
		if ok {
			return value, true
		}
	}
	return nil, false
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
