package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fortigatecli/internal/config"
	"fortigatecli/internal/fortigate"

	"github.com/spf13/cobra"
)

func TestSystemHostnameCommand(t *testing.T) {
	output := executeSystemCommand(t, []string{"system", "hostname"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/api/v2/monitor/system/status" {
			t.Fatalf("path = %q", got)
		}
		_, _ = io.WriteString(w, `{"status":"success","http_status":200,"results":{"hostname":"fg-prod"}}`)
	}))

	var got map[string]any
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got["hostname"] != "fg-prod" {
		t.Fatalf("hostname = %#v, want fg-prod", got["hostname"])
	}
}

func TestSystemFirmwareCommand(t *testing.T) {
	output := executeSystemCommand(t, []string{"system", "firmware"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/api/v2/monitor/system/status" {
			t.Fatalf("path = %q", got)
		}
		_, _ = io.WriteString(w, `{"status":"success","http_status":200,"version":"v7.4.1","build":1234,"revision":"r123","serial":"FGT1234567890","results":{"hostname":"fg-prod"}}`)
	}))

	var got map[string]any
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got["version"] != "v7.4.1" {
		t.Fatalf("version = %#v, want v7.4.1", got["version"])
	}
	if got["build"] != float64(1234) {
		t.Fatalf("build = %#v, want 1234", got["build"])
	}
	if got["revision"] != "r123" {
		t.Fatalf("revision = %#v, want r123", got["revision"])
	}
	if got["serial"] != "FGT1234567890" {
		t.Fatalf("serial = %#v, want FGT1234567890", got["serial"])
	}
}

func TestSystemInterfaceCommand(t *testing.T) {
	output := executeSystemCommand(t, []string{"system", "interface", "port1", "--field", "name"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/api/v2/monitor/system/interface" {
			t.Fatalf("path = %q", got)
		}
		query := r.URL.Query()
		if got := query["filter"]; len(got) != 1 || got[0] != "name==port1" {
			t.Fatalf("filter = %#v", got)
		}
		if got := query.Get("fields"); got != "name" {
			t.Fatalf("fields = %q", got)
		}
		_, _ = io.WriteString(w, `{"status":"success","http_status":200,"path":"/api/v2/monitor/system/interface","results":[{"name":"port1","status":"up"}]}`)
	}))

	var got fortigate.Envelope
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.Path != "/api/v2/monitor/system/interface" {
		t.Fatalf("Path = %q, want /api/v2/monitor/system/interface", got.Path)
	}
	results, ok := got.Results.([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("Results = %#v, want single interface entry", got.Results)
	}
}

func TestSystemHAPeersCommand(t *testing.T) {
	output := executeSystemCommand(t, []string{"system", "ha-peers"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/api/v2/monitor/system/ha-status" {
			t.Fatalf("path = %q", got)
		}
		_, _ = io.WriteString(w, `{"status":"success","http_status":200,"results":{"role":"primary","peers":[{"serial":"FGT2","hostname":"fg-b"}]}}`)
	}))

	var got systemHAPeersResponse
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.Role != "primary" {
		t.Fatalf("Role = %q, want primary", got.Role)
	}
	peers, ok := got.Peers.([]any)
	if !ok || len(peers) != 1 {
		t.Fatalf("Peers = %#v, want one peer", got.Peers)
	}
}

func executeSystemCommand(t *testing.T, args []string, handler http.Handler) string {
	t.Helper()

	t.Setenv("HOME", t.TempDir())

	server := httptest.NewTLSServer(handler)
	defer server.Close()

	if err := config.Save(config.Config{
		Host:     server.URL,
		Token:    "secret-token",
		Insecure: true,
		Timeout:  5 * time.Second,
	}); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := newRootCommand()
	setCommandStreamsForTest(cmd, stdout, stderr)
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v, stderr = %s", err, stderr.String())
	}
	return stdout.String()
}

func setCommandStreamsForTest(cmd *cobra.Command, stdout *bytes.Buffer, stderr *bytes.Buffer) {
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	for _, child := range cmd.Commands() {
		setCommandStreamsForTest(child, stdout, stderr)
	}
}
