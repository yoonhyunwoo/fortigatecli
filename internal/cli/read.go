package cli

import (
	"fortigatecli/internal/fortigate"
	"fortigatecli/internal/output"

	"github.com/spf13/cobra"
)

type readAlias struct {
	use   string
	short string
	path  string
	kind  string
}

type readOptions struct {
	filters    []string
	fields     []string
	formats    []string
	sort       []string
	start      int
	count      int
	withMeta   bool
	datasource bool
}

func newReadOptions() *readOptions {
	return &readOptions{
		start: -1,
		count: -1,
	}
}

func bindReadFlags(cmd *cobra.Command, opts *readOptions) {
	cmd.Flags().StringArrayVar(&opts.filters, "filter", nil, "repeatable API filter")
	cmd.Flags().StringArrayVar(&opts.fields, "field", nil, "repeatable field selector")
	cmd.Flags().StringArrayVar(&opts.formats, "format", nil, "repeatable API format selector")
	cmd.Flags().StringArrayVar(&opts.sort, "sort", nil, "repeatable sort directive")
	cmd.Flags().IntVar(&opts.start, "start", -1, "result offset")
	cmd.Flags().IntVar(&opts.count, "count", -1, "result count limit")
	cmd.Flags().BoolVar(&opts.withMeta, "with-meta", false, "request metadata in the response")
	cmd.Flags().BoolVar(&opts.datasource, "datasource", false, "request FortiGate datasource expansion")
}

func (o *readOptions) toAPIOptions() fortigate.ReadOptions {
	return fortigate.ReadOptions{
		Filters:    o.filters,
		Fields:     o.fields,
		Formats:    o.formats,
		Sort:       o.sort,
		Start:      o.start,
		Count:      o.count,
		WithMeta:   o.withMeta,
		Datasource: o.datasource,
	}
}

func newReadAliasCommand(rootOpts *rootOptions, alias readAlias) *cobra.Command {
	readOpts := newReadOptions()
	cmd := &cobra.Command{
		Use:   alias.use,
		Short: alias.short,
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

			var envelope any
			switch alias.kind {
			case "cmdb":
				envelope, err = client.GetCMDB(ctx, alias.path, readOpts.toAPIOptions())
			default:
				envelope, err = client.GetMonitor(ctx, alias.path, readOpts.toAPIOptions())
			}
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
