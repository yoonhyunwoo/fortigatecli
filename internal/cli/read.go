package cli

import (
	"fortigatecli/internal/fortigate"
	"fortigatecli/internal/output"

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

type shapeOptions struct {
	query      string
	selectors  []string
	flatten    bool
	flattenSep string
	columns    []string
}

func newShapeOptions() *shapeOptions {
	return &shapeOptions{
		flattenSep: ".",
	}
}

func bindShapeFlags(cmd *cobra.Command, opts *shapeOptions) {
	cmd.Flags().StringVar(&opts.query, "query", "", "local selector applied to the response payload")
	cmd.Flags().StringArrayVar(&opts.selectors, "select", nil, "repeatable local field projection")
	cmd.Flags().BoolVar(&opts.flatten, "flatten", false, "flatten nested objects for local output shaping")
	cmd.Flags().StringVar(&opts.flattenSep, "flatten-sep", ".", "separator for flattened keys")
	cmd.Flags().StringSliceVar(&opts.columns, "columns", nil, "ordered output columns for shaped rows")
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

func (o *shapeOptions) toOutputOptions() output.ShapeOptions {
	if o == nil {
		return output.ShapeOptions{}
	}

	return output.ShapeOptions{
		Query:      o.query,
		Select:     append([]string(nil), o.selectors...),
		Flatten:    o.flatten,
		FlattenSep: o.flattenSep,
		Columns:    append([]string(nil), o.columns...),
	}
}
