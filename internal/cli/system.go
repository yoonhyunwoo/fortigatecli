package cli

import (
	"context"
	"fmt"
	"strings"

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
	systemCmd := &cobra.Command{Use: "system"}
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

type backupRunner interface {
	BackupWithOptions(context.Context, fortigate.BackupOptions) ([]byte, error)
	BackupPlan(fortigate.BackupOptions) (*fortigate.BackupPlan, error)
}

type backupCommandOptions struct {
	scope      string
	outputPath string
	force      bool
	dryRun     bool
}

type backupDryRunReport struct {
	URL    string `json:"url"`
	Scope  string `json:"scope"`
	VDOM   string `json:"vdom,omitempty"`
	Output string `json:"output"`
}

func newSystemBackupCommand(rootOpts *rootOptions) *cobra.Command {
	backupOpts := &backupCommandOptions{}
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Print the system config backup to stdout",
		Long: strings.Join([]string{
			"Print the system config backup to stdout.",
			"",
			"This command is stdout-only. Use `system backup export` to save",
			"a backup to a file. Backup scope must be explicit: `global` omits",
			"`vdom`, and `vdom` requires `--vdom`.",
		}, "\n"),
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
			options, err := backupOpts.toAPIOptions(cmd, true)
			if err != nil {
				return err
			}
			data, err := client.BackupWithOptions(ctx, options)
			if err != nil {
				return err
			}
			return writeStdout(cmd, data)
		},
	}
	cmd.Flags().StringVar(&backupOpts.scope, "scope", string(fortigate.BackupScopeGlobal), "backup scope: global or vdom")
	cmd.AddCommand(newSystemBackupExportCommand(rootOpts))
	setDefaultStreams(cmd)
	return cmd
}

func newSystemBackupExportCommand(rootOpts *rootOptions) *cobra.Command {
	backupOpts := &backupCommandOptions{}
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export the system config backup to a file",
		Long: strings.Join([]string{
			"Export the system config backup to a file.",
			"",
			"Use `--output PATH` to write a file. Existing files are not",
			"overwritten unless `--force` is set. `--output -` is the only",
			"way to route export output to stdout. `--dry-run` prints the",
			"request URL and destination without creating a file.",
		}, "\n"),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadRuntimeConfig(rootOpts.vdom)
			if err != nil {
				return err
			}

			client, err := newClient(cfg)
			if err != nil {
				return output.NewError("client_error", err.Error(), nil)
			}

			options, err := backupOpts.toAPIOptions(cmd, false)
			if err != nil {
				return err
			}

			return runBackupExport(cmd, client, rootOpts.output, options)
		},
	}
	cmd.Flags().StringVar(&backupOpts.scope, "scope", string(fortigate.BackupScopeGlobal), "backup scope: global or vdom")
	cmd.Flags().StringVar(&backupOpts.outputPath, "output", "", "write export output to PATH, or '-' for stdout")
	cmd.Flags().BoolVar(&backupOpts.force, "force", false, "overwrite an existing output file")
	cmd.Flags().BoolVar(&backupOpts.dryRun, "dry-run", false, "print the backup request and destination without writing a file")
	setDefaultStreams(cmd)
	return cmd
}

func runBackupExport(cmd *cobra.Command, client backupRunner, format string, options fortigate.BackupOptions) error {
	plan, err := client.BackupPlan(options)
	if err != nil {
		return err
	}

	if options.DryRun {
		return render(cmd, format, backupDryRunReport{
			URL:    plan.URL,
			Scope:  string(plan.Scope),
			VDOM:   plan.VDOM,
			Output: options.OutputPath,
		})
	}

	ctx, cancel := commandContext()
	defer cancel()

	data, err := client.BackupWithOptions(ctx, options)
	if err != nil {
		return err
	}

	if options.Stdout {
		return writeStdout(cmd, data)
	}

	if err := output.WriteFileAtomic(options.OutputPath, data, options.Overwrite); err != nil {
		return output.NewError("file_error", err.Error(), nil)
	}

	return nil
}

func (o *backupCommandOptions) toAPIOptions(cmd *cobra.Command, allowImplicitStdout bool) (fortigate.BackupOptions, error) {
	scope, err := parseBackupScope(o.scope)
	if err != nil {
		return fortigate.BackupOptions{}, err
	}
	vdomFlag := cmd.Flag("vdom")
	if scope == fortigate.BackupScopeGlobal && vdomFlag != nil && vdomFlag.Changed {
		return fortigate.BackupOptions{}, output.NewError("validation_error", "--vdom can only be used with --scope vdom", nil)
	}

	options := fortigate.BackupOptions{
		Scope:      scope,
		OutputPath: o.outputPath,
		Overwrite:  o.force,
		DryRun:     o.dryRun,
	}

	if scope == fortigate.BackupScopeVDOM {
		if vdomFlag != nil {
			options.VDOM = vdomFlag.Value.String()
		}
	}

	switch {
	case allowImplicitStdout:
		options.Stdout = true
	case o.outputPath == "":
		return fortigate.BackupOptions{}, output.NewError("validation_error", "--output is required", nil)
	case o.outputPath == "-":
		options.Stdout = true
	}

	return options, nil
}

func parseBackupScope(raw string) (fortigate.BackupScope, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(fortigate.BackupScopeGlobal):
		return fortigate.BackupScopeGlobal, nil
	case string(fortigate.BackupScopeVDOM):
		return fortigate.BackupScopeVDOM, nil
	default:
		return "", output.NewError("validation_error", fmt.Sprintf("unsupported backup scope: %s", raw), nil)
	}
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
	shapeOpts := newShapeOptions()
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
			return renderRead(cmd, rootOpts.output, envelope, shapeOpts)
		},
	}
	bindReadFlags(cmd, readOpts)
	bindShapeFlags(cmd, shapeOpts)
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
