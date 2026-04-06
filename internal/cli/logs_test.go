package cli

import (
	"bytes"
	"context"
	"testing"
	"time"

	"fortigatecli/internal/fortigate"
)

func TestLogsCommandIncludesObservabilityCategories(t *testing.T) {
	cmd := newLogsCommand(&rootOptions{})

	got := make(map[string]bool)
	for _, child := range cmd.Commands() {
		got[child.Name()] = true
	}

	for _, name := range []string{"traffic", "event", "utm", "system", "session", "performance"} {
		if !got[name] {
			t.Fatalf("missing logs category %q", name)
		}
	}
}

func TestPerformanceCommandIncludesAliases(t *testing.T) {
	cmd := newLogsCategoryCommand(&rootOptions{}, "performance", "Performance observability reads")

	got := make(map[string]bool)
	for _, child := range cmd.Commands() {
		got[child.Name()] = true
	}

	for _, name := range []string{"cpu", "memory", "sessions", "session-rate", "npu-sessions"} {
		if !got[name] {
			t.Fatalf("missing performance alias %q", name)
		}
	}
}

func TestRunWatchedReadSuppressesUnchangedSnapshots(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	snapshots := [][]byte{
		[]byte(`{"value":1}`),
		[]byte(`{"value":1}`),
		[]byte(`{"value":2}`),
	}

	var renders int
	var calls int
	errCh := make(chan error, 1)
	go func() {
		errCh <- runWatchedRead(ctx, 2*time.Millisecond, func(context.Context) (*fortigate.Envelope, error) {
			if calls >= len(snapshots) {
				return &fortigate.Envelope{Results: map[string]any{"value": 2}}, nil
			}
			value := snapshots[calls]
			calls++
			return &fortigate.Envelope{Results: value}, nil
		}, func(*fortigate.Envelope) error {
			renders++
			if renders == 2 {
				cancel()
			}
			return nil
		})
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("runWatchedRead() error = %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("runWatchedRead() timed out")
	}

	if renders != 2 {
		t.Fatalf("render count = %d, want 2", renders)
	}
}

func TestRunWatchedReadStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runWatchedRead(ctx, time.Millisecond, func(context.Context) (*fortigate.Envelope, error) {
		return &fortigate.Envelope{Results: []map[string]any{}}, nil
	}, func(*fortigate.Envelope) error {
		return nil
	})
	if err != nil {
		t.Fatalf("runWatchedRead() error = %v", err)
	}
}

func TestRunReadRendersSingleEnvelope(t *testing.T) {
	cmd := newLogsAliasCommand(&rootOptions{output: "json"}, logsAlias{
		use:           "list",
		short:         "List sessions",
		path:          "session-top",
		kind:          "session",
		supportsWatch: true,
		supportsTable: true,
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetContext(context.Background())

	err := runRead(cmd, "json", false, time.Second, func(context.Context) (*fortigate.Envelope, error) {
		return &fortigate.Envelope{Status: "success", Results: []map[string]any{{"src": "1.1.1.1"}}}, nil
	})
	if err != nil {
		t.Fatalf("runRead() error = %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"status": "success"`)) {
		t.Fatalf("output = %q", buf.String())
	}
}
