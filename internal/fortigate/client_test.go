package fortigate

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestGetMonitorAddsAuthorizationAndVDOM(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Fatalf("Authorization header = %q", got)
		}
		if got := r.URL.Query().Get("vdom"); got != "root" {
			t.Fatalf("vdom query = %q", got)
		}
		_, _ = io.WriteString(w, `{"status":"success","http_status":200,"results":{"hostname":"fg"}}`)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:  server.URL,
		Token:    "secret-token",
		VDOM:     "root",
		Insecure: true,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	envelope, err := client.GetMonitor(context.Background(), "system/status", ReadOptions{})
	if err != nil {
		t.Fatalf("GetMonitor() error = %v", err)
	}

	if envelope.Status != "success" {
		t.Fatalf("Status = %q, want success", envelope.Status)
	}
}

func TestBackupReturnsBody(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "config-version=FGT\n")
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:  server.URL,
		Token:    "secret-token",
		VDOM:     "root",
		Insecure: true,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	data, err := client.Backup(context.Background())
	if err != nil {
		t.Fatalf("Backup() error = %v", err)
	}

	if string(data) != "config-version=FGT\n" {
		t.Fatalf("Backup() = %q", string(data))
	}
}

func TestGetReturnsAPIError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"status":"error","http_status":401,"message":"invalid token"}`)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:  server.URL,
		Token:    "secret-token",
		VDOM:     "root",
		Insecure: true,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.GetCMDB(context.Background(), "firewall/address", ReadOptions{})
	if err == nil {
		t.Fatal("GetCMDB() error = nil, want error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.Code != http.StatusUnauthorized {
		t.Fatalf("APIError.Code = %d", apiErr.Code)
	}
	if apiErr.Message != "invalid token" {
		t.Fatalf("APIError.Message = %q", apiErr.Message)
	}
}

func TestClientSupportsSelfSignedTLS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"status":"success","http_status":200}`)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:  server.URL,
		Token:    "secret-token",
		VDOM:     "root",
		Insecure: true,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	transport := client.httpClient.Transport.(*http.Transport)
	if transport.TLSClientConfig == nil || !transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("expected InsecureSkipVerify=true")
	}

	_, err = client.GetMonitor(context.Background(), "system/status", ReadOptions{})
	if err != nil {
		t.Fatalf("GetMonitor() error = %v", err)
	}
}

func TestRawGetRejectsAbsoluteURL(t *testing.T) {
	client, err := NewClient(Config{
		BaseURL:  "https://fortigate.example.com",
		Token:    "secret-token",
		VDOM:     "root",
		Insecure: true,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.RawGet(context.Background(), "https://other.example.com/api/v2/monitor/system/status", ReadOptions{})
	if err == nil {
		t.Fatal("RawGet() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "relative") {
		t.Fatalf("RawGet() error = %v", err)
	}
}

func TestGetMonitorAddsReadOptions(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if got := query["filter"]; len(got) != 2 || got[0] != "name==port1" || got[1] != "status==up" {
			t.Fatalf("filter query = %#v", got)
		}
		if got := query.Get("fields"); got != "name,ip" {
			t.Fatalf("fields query = %q", got)
		}
		if got := query["format"]; len(got) != 2 || got[0] != "name" || got[1] != "status" {
			t.Fatalf("format query = %#v", got)
		}
		if got := query["sort"]; len(got) != 1 || got[0] != "name" {
			t.Fatalf("sort query = %#v", got)
		}
		if got := query.Get("start"); got != "5" {
			t.Fatalf("start query = %q", got)
		}
		if got := query.Get("count"); got != "10" {
			t.Fatalf("count query = %q", got)
		}
		if got := query.Get("with_meta"); got != "true" {
			t.Fatalf("with_meta query = %q", got)
		}
		if got := query.Get("datasource"); got != "true" {
			t.Fatalf("datasource query = %q", got)
		}
		_, _ = io.WriteString(w, `{"status":"success","http_status":200}`)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:  server.URL,
		Token:    "secret-token",
		VDOM:     "root",
		Insecure: true,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.GetMonitor(context.Background(), "system/interface", ReadOptions{
		Filters:    []string{"name==port1", "status==up"},
		Fields:     []string{"name", "ip"},
		Formats:    []string{"name", "status"},
		Sort:       []string{"name"},
		Start:      5,
		Count:      10,
		WithMeta:   true,
		Datasource: true,
	})
	if err != nil {
		t.Fatalf("GetMonitor() error = %v", err)
	}
}

func TestRawGetMergesExistingQueryAndReadOptions(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if got := query.Get("foo"); got != "bar" {
			t.Fatalf("foo query = %q", got)
		}
		if got := query.Get("fields"); got != "name" {
			t.Fatalf("fields query = %q", got)
		}
		if got := query.Get("count"); got != "1" {
			t.Fatalf("count query = %q", got)
		}
		if got := query.Get("vdom"); got != "root" {
			t.Fatalf("vdom query = %q", got)
		}
		_, _ = io.WriteString(w, `{"status":"success","http_status":200}`)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:  server.URL,
		Token:    "secret-token",
		VDOM:     "root",
		Insecure: true,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.RawGet(context.Background(), "/api/v2/cmdb/firewall/address?foo=bar", ReadOptions{
		Fields: []string{"name"},
		Count:  1,
	})
	if err != nil {
		t.Fatalf("RawGet() error = %v", err)
	}
}

func TestClientKeepsVerificationEnabledWhenConfigured(t *testing.T) {
	client, err := NewClient(Config{
		BaseURL:  "https://fortigate.example.com",
		Token:    "secret-token",
		VDOM:     "root",
		Insecure: false,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	transport := client.httpClient.Transport.(*http.Transport)
	if transport.TLSClientConfig == nil {
		t.Fatal("expected TLSClientConfig")
	}
	if transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("expected InsecureSkipVerify=false")
	}
}

func TestGetDiscoverySchemaBuildsSchemaRequest(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/api/v2/cmdb/firewall/address" {
			t.Fatalf("path = %q", got)
		}
		if got := r.URL.Query().Get("action"); got != "schema" {
			t.Fatalf("action query = %q", got)
		}
		if got := r.URL.Query().Get("with_meta"); got != "true" {
			t.Fatalf("with_meta query = %q", got)
		}
		_, _ = io.WriteString(w, `{"status":"success","http_status":200,"results":{"mkey":"name"}}`)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:  server.URL,
		Token:    "secret-token",
		VDOM:     "root",
		Insecure: true,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	report, err := client.GetDiscoverySchema(context.Background(), DiscoveryTargetCMDB, "firewall/address", DiscoverySchemaOptions{
		WithMeta: true,
	})
	if err != nil {
		t.Fatalf("GetDiscoverySchema() error = %v", err)
	}
	if report.Source != "api" {
		t.Fatalf("Source = %q", report.Source)
	}
}

func TestGetDiscoverySchemaReturnsUnsupportedReport(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"status":"error","http_status":404,"message":"schema not found"}`)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:  server.URL,
		Token:    "secret-token",
		VDOM:     "root",
		Insecure: true,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	report, err := client.GetDiscoverySchema(context.Background(), DiscoveryTargetMonitor, "system/status", DiscoverySchemaOptions{})
	if err != nil {
		t.Fatalf("GetDiscoverySchema() error = %v", err)
	}
	if report.Source != "unsupported" {
		t.Fatalf("Source = %q", report.Source)
	}
	if report.Error == "" {
		t.Fatal("expected unsupported error message")
	}
}

func TestDiscoverFieldsUsesAllowedReadOptions(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if got := r.URL.Path; got != "/api/v2/monitor/system/interface" {
			t.Fatalf("path = %q", got)
		}
		if got := query["filter"]; len(got) != 1 || got[0] != "name==port1" {
			t.Fatalf("filter query = %#v", got)
		}
		if got := query.Get("count"); got != "2" {
			t.Fatalf("count query = %q", got)
		}
		if got := query.Get("with_meta"); got != "true" {
			t.Fatalf("with_meta query = %q", got)
		}
		if got := query.Get("datasource"); got != "true" {
			t.Fatalf("datasource query = %q", got)
		}
		if got := query.Get("fields"); got != "" {
			t.Fatalf("fields query = %q, want empty", got)
		}
		_, _ = io.WriteString(w, `{"status":"success","http_status":200,"results":[{"name":"port1","status":"up"},{"name":"port2","status":true,"mtu":1500}]}`)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:  server.URL,
		Token:    "secret-token",
		VDOM:     "root",
		Insecure: true,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	report, err := client.DiscoverFields(context.Background(), DiscoveryTargetMonitor, "system/interface", DiscoveryFieldOptions{
		Filters:    []string{"name==port1"},
		Count:      2,
		WithMeta:   true,
		Datasource: true,
	})
	if err != nil {
		t.Fatalf("DiscoverFields() error = %v", err)
	}
	if report.SampleCount != 2 {
		t.Fatalf("SampleCount = %d", report.SampleCount)
	}
	wantFields := []string{"mtu", "name", "status"}
	if !slices.Equal(report.Fields, wantFields) {
		t.Fatalf("Fields = %#v, want %#v", report.Fields, wantFields)
	}
	if got := report.InferredTypes["status"]; !slices.Equal(got, []string{"bool", "string"}) {
		t.Fatalf("status types = %#v", got)
	}
}

func TestGetDiscoveryCapabilitiesProbeReflectsSchemaSupport(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = io.WriteString(w, `{"status":"error","http_status":405,"message":"invalid action"}`)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:  server.URL,
		Token:    "secret-token",
		VDOM:     "root",
		Insecure: true,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	report, err := client.GetDiscoveryCapabilities(context.Background(), DiscoveryTargetCMDB, "firewall/address", DiscoveryCapabilityOptions{
		Probe: true,
	})
	if err != nil {
		t.Fatalf("GetDiscoveryCapabilities() error = %v", err)
	}
	if report.ProbeResult == nil {
		t.Fatal("ProbeResult = nil")
	}
	if report.ProbeResult.SchemaSupported {
		t.Fatal("SchemaSupported = true, want false")
	}
}
