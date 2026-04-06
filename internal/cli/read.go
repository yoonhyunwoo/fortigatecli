package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	eqFilters  []string
	neFilters  []string
	contains   []string
	prefix     []string
	start      int
	count      int
	page       int
	pageSize   int
	all        bool
	limit      int
	withMeta   bool
	datasource bool
	allVDOMs   bool
	watch      bool
	follow     bool
	interval   time.Duration
}

type shapeOptions struct {
	query      string
	selectors  []string
	flatten    bool
	flattenSep string
	columns    []string
}

func newReadOptions() *readOptions {
	return &readOptions{
		start:    -1,
		count:    -1,
		page:     -1,
		pageSize: -1,
		limit:    -1,
		interval: 5 * time.Second,
	}
}

func newShapeOptions() *shapeOptions {
	return &shapeOptions{flattenSep: "."}
}

func bindReadFlags(cmd *cobra.Command, opts *readOptions) {
	cmd.Flags().StringArrayVar(&opts.filters, "filter", nil, "repeatable API filter")
	cmd.Flags().StringArrayVar(&opts.fields, "field", nil, "repeatable field selector")
	cmd.Flags().StringArrayVar(&opts.formats, "format", nil, "repeatable API format selector")
	cmd.Flags().StringArrayVar(&opts.sort, "sort", nil, "repeatable sort directive")
	cmd.Flags().IntVar(&opts.start, "start", -1, "result offset")
	cmd.Flags().IntVar(&opts.count, "count", -1, "result count limit")
	cmd.Flags().IntVar(&opts.page, "page", -1, "1-based page number")
	cmd.Flags().IntVar(&opts.pageSize, "page-size", -1, "page size for page-based reads")
	cmd.Flags().BoolVar(&opts.all, "all", false, "follow server-provided next-page metadata until exhausted")
	cmd.Flags().BoolVar(&opts.withMeta, "with-meta", false, "request metadata in the response")
	cmd.Flags().BoolVar(&opts.datasource, "datasource", false, "request FortiGate datasource expansion")
	cmd.Flags().BoolVar(&opts.allVDOMs, "all-vdoms", false, "read across all configured VDOMs")
}

func bindMonitorReadFlags(cmd *cobra.Command, opts *readOptions) {
	bindReadFlags(cmd, opts)
	cmd.Flags().StringArrayVar(&opts.eqFilters, "eq", nil, "repeatable equality filter in the form field=value")
	cmd.Flags().StringArrayVar(&opts.neFilters, "ne", nil, "repeatable inequality filter in the form field=value")
	cmd.Flags().StringArrayVar(&opts.contains, "contains", nil, "repeatable contains filter in the form field=value")
	cmd.Flags().StringArrayVar(&opts.prefix, "prefix", nil, "repeatable prefix filter in the form field=value")
}

func bindObservabilityReadFlags(cmd *cobra.Command, opts *readOptions) {
	bindReadFlags(cmd, opts)
	cmd.Flags().IntVar(&opts.limit, "limit", -1, "observability result limit")
	cmd.Flags().BoolVar(&opts.watch, "watch", false, "poll for changes and re-render on updates")
	cmd.Flags().BoolVar(&opts.follow, "follow", false, "alias for --watch")
	cmd.Flags().DurationVar(&opts.interval, "interval", 5*time.Second, "watch poll interval")
}

func bindDiscoverySchemaFlags(cmd *cobra.Command, opts *readOptions) {
	cmd.Flags().BoolVar(&opts.withMeta, "with-meta", false, "request metadata in the response")
}

func bindDiscoveryFieldFlags(cmd *cobra.Command, opts *readOptions) {
	cmd.Flags().StringArrayVar(&opts.filters, "filter", nil, "repeatable API filter")
	cmd.Flags().IntVar(&opts.count, "count", -1, "result count limit")
	cmd.Flags().BoolVar(&opts.withMeta, "with-meta", false, "request metadata in the response")
	cmd.Flags().BoolVar(&opts.datasource, "datasource", false, "request FortiGate datasource expansion")
}

func bindDiscoveryCapabilitiesFlags(cmd *cobra.Command, probe *bool) {
	cmd.Flags().BoolVar(probe, "probe", false, "probe the target resource for schema endpoint support")
}

func bindShapeFlags(cmd *cobra.Command, opts *shapeOptions) {
	cmd.Flags().StringVar(&opts.query, "query", "", "local selector applied to the response payload")
	cmd.Flags().StringArrayVar(&opts.selectors, "select", nil, "repeatable local field projection")
	cmd.Flags().BoolVar(&opts.flatten, "flatten", false, "flatten nested objects for local output shaping")
	cmd.Flags().StringVar(&opts.flattenSep, "flatten-sep", ".", "separator for flattened keys")
	cmd.Flags().StringSliceVar(&opts.columns, "columns", nil, "ordered output columns for shaped rows")
}

func (o *readOptions) toAPIOptions() fortigate.ReadOptions {
	count := o.count
	if count < 0 && o.limit >= 0 {
		count = o.limit
	}
	return fortigate.ReadOptions{
		Filters: append([]string{}, o.filters...),
		Fields:  append([]string{}, o.fields...),
		Formats: append([]string{}, o.formats...),
		Sort:    append([]string{}, o.sort...),
		Start:   o.start,
		Count:   count,
		Page: fortigate.PageOptions{
			Start:    o.start,
			Count:    count,
			Page:     o.page,
			PageSize: o.pageSize,
		},
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

func newReadAliasCommand(rootOpts *rootOptions, alias readAlias) *cobra.Command {
	readOpts := newReadOptions()
	shapeOpts := newShapeOptions()
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
			return renderRead(cmd, rootOpts.output, envelope, shapeOpts)
		},
	}
	bindReadFlags(cmd, readOpts)
	bindShapeFlags(cmd, shapeOpts)
	setDefaultStreams(cmd)
	return cmd
}

func (o *readOptions) toMonitorAPIOptions(spec *monitorEndpointSpec) (fortigate.ReadOptions, error) {
	if spec != nil {
		if err := validateMonitorReadOptions(*spec, o); err != nil {
			return fortigate.ReadOptions{}, err
		}
	}
	options := o.toAPIOptions()
	options = mergeReadOptions(spec, options)
	filters, err := o.monitorShortcutFilters()
	if err != nil {
		return fortigate.ReadOptions{}, err
	}
	options.Filters = append(options.Filters, filters...)
	return options, nil
}

func mergeReadOptions(spec *monitorEndpointSpec, options fortigate.ReadOptions) fortigate.ReadOptions {
	if spec == nil {
		return options
	}
	merged := fortigate.ReadOptions{Start: -1, Count: -1}
	merged.Filters = append([]string{}, spec.defaultQuery.Filters...)
	merged.Fields = append([]string{}, spec.defaultQuery.Fields...)
	merged.Formats = append([]string{}, spec.defaultQuery.Formats...)
	merged.Sort = append([]string{}, spec.defaultQuery.Sort...)
	if spec.defaultQuery.Start >= 0 {
		merged.Start = spec.defaultQuery.Start
	}
	if spec.defaultQuery.Count >= 0 {
		merged.Count = spec.defaultQuery.Count
	}
	merged.WithMeta = spec.defaultQuery.WithMeta
	merged.Datasource = spec.defaultQuery.Datasource
	merged.Filters = append(append([]string{}, merged.Filters...), options.Filters...)
	if len(options.Fields) > 0 {
		merged.Fields = append([]string{}, options.Fields...)
	}
	if len(options.Formats) > 0 {
		merged.Formats = append([]string{}, options.Formats...)
	}
	merged.Sort = append(append([]string{}, merged.Sort...), options.Sort...)
	if options.Start >= 0 {
		merged.Start = options.Start
	}
	if options.Count >= 0 {
		merged.Count = options.Count
	}
	merged.WithMeta = merged.WithMeta || options.WithMeta
	merged.Datasource = merged.Datasource || options.Datasource
	return merged
}

func validateMonitorReadOptions(spec monitorEndpointSpec, opts *readOptions) error {
	if usesFilterOptions(opts) && !supportsCapability(spec.capabilities, monitorCapabilityFilter) {
		return fmt.Errorf("monitor alias %q does not support filter options", spec.use)
	}
	if len(opts.fields) > 0 && !supportsCapability(spec.capabilities, monitorCapabilityField) {
		return fmt.Errorf("monitor alias %q does not support --field", spec.use)
	}
	if len(opts.formats) > 0 && !supportsCapability(spec.capabilities, monitorCapabilityFormat) {
		return fmt.Errorf("monitor alias %q does not support --format", spec.use)
	}
	if len(opts.sort) > 0 && !supportsCapability(spec.capabilities, monitorCapabilitySort) {
		return fmt.Errorf("monitor alias %q does not support --sort", spec.use)
	}
	if opts.start >= 0 && !supportsCapability(spec.capabilities, monitorCapabilityStart) {
		return fmt.Errorf("monitor alias %q does not support --start", spec.use)
	}
	if opts.count >= 0 && !supportsCapability(spec.capabilities, monitorCapabilityCount) {
		return fmt.Errorf("monitor alias %q does not support --count", spec.use)
	}
	if opts.withMeta && !supportsCapability(spec.capabilities, monitorCapabilityWithMeta) {
		return fmt.Errorf("monitor alias %q does not support --with-meta", spec.use)
	}
	if opts.datasource && !supportsCapability(spec.capabilities, monitorCapabilityDatasource) {
		return fmt.Errorf("monitor alias %q does not support --datasource", spec.use)
	}
	return nil
}

func usesFilterOptions(opts *readOptions) bool {
	return len(opts.filters) > 0 || len(opts.eqFilters) > 0 || len(opts.neFilters) > 0 || len(opts.contains) > 0 || len(opts.prefix) > 0
}

func supportsCapability(capabilities monitorReadCapability, capability monitorReadCapability) bool {
	return capabilities&capability != 0
}

func (o *readOptions) monitorShortcutFilters() ([]string, error) {
	shortcuts := []struct {
		values   []string
		operator string
		name     string
	}{
		{values: o.eqFilters, operator: "==", name: "--eq"},
		{values: o.neFilters, operator: "!=", name: "--ne"},
		{values: o.contains, operator: "=@", name: "--contains"},
		{values: o.prefix, operator: "=@", name: "--prefix"},
	}
	var filters []string
	for _, shortcut := range shortcuts {
		for _, raw := range shortcut.values {
			filter, err := translateShortcut(shortcut.name, shortcut.operator, raw)
			if err != nil {
				return nil, err
			}
			if shortcut.name == "--prefix" {
				filter += "*"
			}
			filters = append(filters, filter)
		}
	}
	return filters, nil
}

func translateShortcut(flag string, operator string, raw string) (string, error) {
	field, value, ok := strings.Cut(raw, "=")
	if !ok || strings.TrimSpace(field) == "" || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s expects field=value", flag)
	}
	return strings.TrimSpace(field) + operator + strings.TrimSpace(value), nil
}

func (o *readOptions) watchEnabled() bool { return o.watch || o.follow }

func parseDiscoveryTarget(raw string) (fortigate.DiscoveryTarget, error) {
	switch raw {
	case string(fortigate.DiscoveryTargetCMDB):
		return fortigate.DiscoveryTargetCMDB, nil
	case string(fortigate.DiscoveryTargetMonitor):
		return fortigate.DiscoveryTargetMonitor, nil
	default:
		return "", fmt.Errorf("unsupported discovery target %q: must be cmdb or monitor", raw)
	}
}

func (o *readOptions) toDiscoverySchemaOptions() fortigate.DiscoverySchemaOptions {
	return fortigate.DiscoverySchemaOptions{WithMeta: o.withMeta}
}

func (o *readOptions) toDiscoveryFieldOptions() fortigate.DiscoveryFieldOptions {
	return fortigate.DiscoveryFieldOptions{
		Filters:    o.filters,
		Count:      o.count,
		WithMeta:   o.withMeta,
		Datasource: o.datasource,
	}
}

type envelopeReader func(context.Context) (*fortigate.Envelope, error)

func readCommandContext(cmd *cobra.Command, watch bool) (context.Context, context.CancelFunc) {
	base := cmd.Context()
	if base == nil {
		base = context.Background()
	}
	if watch {
		return context.WithCancel(base)
	}
	return context.WithTimeout(base, 30*time.Second)
}

func runRead(cmd *cobra.Command, format string, watch bool, interval time.Duration, reader envelopeReader) error {
	ctx, cancel := readCommandContext(cmd, watch)
	defer cancel()
	if !watch {
		envelope, err := reader(ctx)
		if err != nil {
			return err
		}
		return render(cmd, format, envelope)
	}
	return runWatchedRead(ctx, interval, reader, func(envelope *fortigate.Envelope) error {
		return render(cmd, format, envelope)
	})
}

func runWatchedRead(ctx context.Context, interval time.Duration, reader envelopeReader, renderFn func(*fortigate.Envelope) error) error {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	envelope, err := reader(ctx)
	if err != nil {
		return err
	}
	current, err := canonicalResults(envelope.Results)
	if err != nil {
		return err
	}
	if err := renderFn(envelope); err != nil {
		return err
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil && err != context.Canceled {
				return err
			}
			return nil
		case <-ticker.C:
			envelope, err := reader(ctx)
			if err != nil {
				return err
			}
			next, err := canonicalResults(envelope.Results)
			if err != nil {
				return err
			}
			if bytes.Equal(current, next) {
				continue
			}
			current = next
			if err := renderFn(envelope); err != nil {
				return err
			}
		}
	}
}

func canonicalResults(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, output.NewError("encode_error", err.Error(), nil)
	}
	return data, nil
}
