package cli

import (
	"fortigatecli/internal/fortigate"

	"github.com/spf13/cobra"
)

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
