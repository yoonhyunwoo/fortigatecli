package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

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
	limit      int
	withMeta   bool
	datasource bool
	watch      bool
	follow     bool
	interval   time.Duration
}

func newReadOptions() *readOptions {
	return &readOptions{
		start:    -1,
		count:    -1,
		limit:    -1,
		interval: 5 * time.Second,
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

func bindObservabilityReadFlags(cmd *cobra.Command, opts *readOptions) {
	bindReadFlags(cmd, opts)
	cmd.Flags().IntVar(&opts.limit, "limit", -1, "observability result limit")
	cmd.Flags().BoolVar(&opts.watch, "watch", false, "poll for changes and re-render on updates")
	cmd.Flags().BoolVar(&opts.follow, "follow", false, "alias for --watch")
	cmd.Flags().DurationVar(&opts.interval, "interval", 5*time.Second, "watch poll interval")
}

func (o *readOptions) toAPIOptions() fortigate.ReadOptions {
	count := o.count
	if count < 0 && o.limit >= 0 {
		count = o.limit
	}
	return fortigate.ReadOptions{
		Filters:    o.filters,
		Fields:     o.fields,
		Formats:    o.formats,
		Sort:       o.sort,
		Start:      o.start,
		Count:      count,
		WithMeta:   o.withMeta,
		Datasource: o.datasource,
	}
}

func (o *readOptions) watchEnabled() bool {
	return o.watch || o.follow
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
