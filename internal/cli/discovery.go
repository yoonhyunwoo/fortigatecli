package cli

import (
	"fortigatecli/internal/fortigate"
	"fortigatecli/internal/output"

	"github.com/spf13/cobra"
)

func newDiscoveryCommand(rootOpts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discovery",
		Short: "Inspect schema and field capabilities for CMDB and monitor resources",
	}

	cmd.AddCommand(
		newDiscoverySchemaCommand(rootOpts),
		newDiscoveryFieldsCommand(rootOpts),
		newDiscoveryCapabilitiesCommand(rootOpts),
	)

	return cmd
}

func newDiscoverySchemaCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:     "schema <cmdb|monitor> <path>",
		Short:   "Query the API schema endpoint for a resource",
		Args:    cobra.ExactArgs(2),
		Example: "fortigatecli discovery schema cmdb firewall/address --with-meta",
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := parseDiscoveryTarget(args[0])
			if err != nil {
				return err
			}
			return runDiscoverySchema(rootOpts, cmd, target, args[1], readOpts)
		},
	}
	bindDiscoverySchemaFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

func newDiscoveryFieldsCommand(rootOpts *rootOptions) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:     "fields <cmdb|monitor> <path>",
		Short:   "Sample a resource and infer available object fields",
		Args:    cobra.ExactArgs(2),
		Example: "fortigatecli discovery fields monitor system/interface --filter name==port1 --count 5",
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := parseDiscoveryTarget(args[0])
			if err != nil {
				return err
			}
			return runDiscoveryFields(rootOpts, cmd, target, args[1], readOpts)
		},
	}
	bindDiscoveryFieldFlags(cmd, readOpts)
	setDefaultStreams(cmd)
	return cmd
}

func newDiscoveryCapabilitiesCommand(rootOpts *rootOptions) *cobra.Command {
	var probe bool
	cmd := &cobra.Command{
		Use:     "capabilities <cmdb|monitor> <path>",
		Short:   "Show the safe discovery contract for a resource",
		Args:    cobra.ExactArgs(2),
		Example: "fortigatecli discovery capabilities cmdb firewall/address --probe",
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := parseDiscoveryTarget(args[0])
			if err != nil {
				return err
			}
			return runDiscoveryCapabilities(rootOpts, cmd, target, args[1], probe)
		},
	}
	bindDiscoveryCapabilitiesFlags(cmd, &probe)
	setDefaultStreams(cmd)
	return cmd
}

func runDiscoverySchema(rootOpts *rootOptions, cmd *cobra.Command, target fortigate.DiscoveryTarget, resourcePath string, readOpts *readOptions) error {
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

	report, err := client.GetDiscoverySchema(ctx, target, resourcePath, readOpts.toDiscoverySchemaOptions())
	if err != nil {
		return err
	}

	return render(cmd, rootOpts.output, report)
}

func runDiscoveryFields(rootOpts *rootOptions, cmd *cobra.Command, target fortigate.DiscoveryTarget, resourcePath string, readOpts *readOptions) error {
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

	report, err := client.DiscoverFields(ctx, target, resourcePath, readOpts.toDiscoveryFieldOptions())
	if err != nil {
		return err
	}

	return render(cmd, rootOpts.output, report)
}

func runDiscoveryCapabilities(rootOpts *rootOptions, cmd *cobra.Command, target fortigate.DiscoveryTarget, resourcePath string, probe bool) error {
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

	report, err := client.GetDiscoveryCapabilities(ctx, target, resourcePath, fortigate.DiscoveryCapabilityOptions{
		Probe: probe,
	})
	if err != nil {
		return err
	}

	return render(cmd, rootOpts.output, report)
}
