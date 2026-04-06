package cli

import (
	"fmt"

	"fortigatecli/internal/fortigate"
	"fortigatecli/internal/output"

	"github.com/spf13/cobra"
)

type monitorReadCapability uint16

const (
	monitorCapabilityFilter monitorReadCapability = 1 << iota
	monitorCapabilityField
	monitorCapabilityFormat
	monitorCapabilitySort
	monitorCapabilityStart
	monitorCapabilityCount
	monitorCapabilityWithMeta
	monitorCapabilityDatasource
)

const monitorCapabilityAll = monitorCapabilityFilter |
	monitorCapabilityField |
	monitorCapabilityFormat |
	monitorCapabilitySort |
	monitorCapabilityStart |
	monitorCapabilityCount |
	monitorCapabilityWithMeta |
	monitorCapabilityDatasource

type monitorEndpointSpec struct {
	use          string
	short        string
	path         string
	kind         string
	capabilities monitorReadCapability
	defaultQuery fortigate.ReadOptions
}

var monitorEndpointSpecs = []monitorEndpointSpec{
	{
		use:          "status",
		short:        "Fetch system status",
		path:         "system/status",
		kind:         "status",
		capabilities: monitorCapabilityField | monitorCapabilityFormat | monitorCapabilityWithMeta | monitorCapabilityDatasource,
	},
	{
		use:          "interfaces",
		short:        "List system interfaces",
		path:         "system/interface",
		kind:         "list",
		capabilities: monitorCapabilityAll,
	},
	{
		use:          "ha-status",
		short:        "Fetch HA status",
		path:         "system/ha-status",
		kind:         "status",
		capabilities: monitorCapabilityField | monitorCapabilityFormat | monitorCapabilityWithMeta,
	},
	{
		use:          "license",
		short:        "Fetch license status",
		path:         "license/status",
		kind:         "status",
		capabilities: monitorCapabilityField | monitorCapabilityFormat | monitorCapabilityWithMeta,
	},
}

func monitorEndpointSpecByUse(use string) (monitorEndpointSpec, bool) {
	for _, spec := range monitorEndpointSpecs {
		if spec.use == use {
			return spec, true
		}
	}
	return monitorEndpointSpec{}, false
}

func systemMonitorCompatibilitySpecs() []monitorEndpointSpec {
	specs := make([]monitorEndpointSpec, len(monitorEndpointSpecs))
	copy(specs, monitorEndpointSpecs)
	return specs
}

func runMonitor(rootOpts *rootOptions, cmd *cobra.Command, resourcePath string, options fortigate.ReadOptions) error {
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

	envelope, err := client.GetMonitor(ctx, resourcePath, options)
	if err != nil {
		return err
	}

	return render(cmd, rootOpts.output, envelope)
}

func monitorOptionsOrError(opts *readOptions, spec *monitorEndpointSpec) (fortigate.ReadOptions, error) {
	options, err := opts.toMonitorAPIOptions(spec)
	if err != nil {
		return fortigate.ReadOptions{}, output.NewError("validation_error", err.Error(), nil)
	}
	return options, nil
}

func newMonitorAliasCommand(rootOpts *rootOptions, spec monitorEndpointSpec) *cobra.Command {
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

func newMonitorPathCommand(rootOpts *rootOptions, use string, short string) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s <path>", use),
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options, err := monitorOptionsOrError(readOpts, nil)
			if err != nil {
				return err
			}
			return runMonitor(rootOpts, cmd, args[0], options)
		},
	}
	bindMonitorReadFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}
