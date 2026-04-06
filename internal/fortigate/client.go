package fortigate

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type Config struct {
	BaseURL  string
	Token    string
	VDOM     string
	Insecure bool
	Timeout  time.Duration
}

type ReadOptions struct {
	Filters    []string
	Fields     []string
	Formats    []string
	Sort       []string
	Start      int
	Count      int
	WithMeta   bool
	Datasource bool
}

type Client struct {
	baseURL    *url.URL
	token      string
	vdom       string
	httpClient *http.Client
}

func NewClient(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("token is required")
	}

	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: cfg.Insecure,
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	vdom := cfg.VDOM
	if vdom == "" {
		vdom = "root"
	}

	return &Client{
		baseURL: parsed,
		token:   cfg.Token,
		vdom:    vdom,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}, nil
}

func (c *Client) Test(ctx context.Context) (*Envelope, error) {
	return c.GetMonitor(ctx, "system/status", ReadOptions{})
}

func (c *Client) GetMonitor(ctx context.Context, resourcePath string, options ReadOptions) (*Envelope, error) {
	return c.get(ctx, "/api/v2/monitor/"+strings.TrimPrefix(resourcePath, "/"), options, addReadOptions)
}

func (c *Client) GetCMDB(ctx context.Context, resourcePath string, options ReadOptions) (*Envelope, error) {
	return c.get(ctx, "/api/v2/cmdb/"+strings.TrimPrefix(resourcePath, "/"), options, addReadOptions)
}

func (c *Client) GetLog(ctx context.Context, resourcePath string, options ReadOptions) (*Envelope, error) {
	return c.get(ctx, "/api/v2/log/"+strings.TrimPrefix(resourcePath, "/"), options, addLogReadOptions)
}

func (c *Client) GetSession(ctx context.Context, resourcePath string, options ReadOptions) (*Envelope, error) {
	return c.get(ctx, "/api/v2/monitor/firewall/"+strings.TrimPrefix(resourcePath, "/"), options, addReadOptions)
}

func (c *Client) GetPerformance(ctx context.Context, resourcePath string, options ReadOptions) (*Envelope, error) {
	return c.get(ctx, "/api/v2/monitor/"+strings.TrimPrefix(resourcePath, "/"), options, addReadOptions)
}

func (c *Client) RawGet(ctx context.Context, apiPath string, options ReadOptions) (*Envelope, error) {
	if strings.HasPrefix(apiPath, "http://") || strings.HasPrefix(apiPath, "https://") {
		return nil, &APIError{
			Operation: "raw_get",
			Message:   "raw path must be relative to the FortiGate host",
		}
	}
	normalized := apiPath
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	return c.get(ctx, normalized, options, addReadOptions)
}

func (c *Client) GetVPNIPsecStatus(ctx context.Context, options ReadOptions) (*Envelope, error) {
	return c.GetMonitor(ctx, "vpn/ipsec", options)
}

func (c *Client) ListVPNIPsecTunnels(ctx context.Context, options ReadOptions) (*Envelope, error) {
	return c.GetMonitor(ctx, "vpn/ipsec", options)
}

func (c *Client) GetVPNIPsecTunnel(ctx context.Context, tunnelName string, options ReadOptions) (*Envelope, error) {
	options.Filters = append([]string{fmt.Sprintf("name==%s", tunnelName)}, options.Filters...)
	return c.GetMonitor(ctx, "vpn/ipsec", options)
}

func (c *Client) GetSSLVPNSettings(ctx context.Context, options ReadOptions) (*Envelope, error) {
	return c.GetCMDB(ctx, "vpn.ssl/settings", options)
}

func (c *Client) ListSSLVPNSessions(ctx context.Context, options ReadOptions) (*Envelope, error) {
	return c.GetMonitor(ctx, "vpn/ssl", options)
}

func (c *Client) Backup(ctx context.Context) ([]byte, error) {
	u := *c.baseURL
	u.Path = "/api/v2/monitor/system/config/backup"
	query := u.Query()
	query.Set("scope", "global")
	query.Set("vdom", c.vdom)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build backup request: %w", err)
	}

	c.applyHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform backup request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read backup response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, c.decodeError("system_backup", resp.StatusCode, body)
	}

	return body, nil
}

func (c *Client) get(ctx context.Context, apiPath string, options ReadOptions, addQueryOptions func(url.Values, ReadOptions)) (*Envelope, error) {
	parsedPath, err := url.Parse(apiPath)
	if err != nil {
		return nil, fmt.Errorf("parse API path: %w", err)
	}

	u := *c.baseURL
	u.Path = path.Clean(strings.TrimSuffix(c.baseURL.Path, "/") + "/" + strings.TrimPrefix(parsedPath.Path, "/"))
	query := parsedPath.Query()
	query.Set("vdom", c.vdom)
	addQueryOptions(query, options)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	c.applyHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, c.decodeError(apiPath, resp.StatusCode, body)
	}

	var envelope Envelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if envelope.HTTPStatus == 0 {
		envelope.HTTPStatus = resp.StatusCode
	}
	if envelope.Path == "" {
		envelope.Path = parsedPath.Path
	}
	if envelope.VDOM == "" {
		envelope.VDOM = c.vdom
	}

	return &envelope, nil
}

func addReadOptions(query url.Values, options ReadOptions) {
	for _, filter := range options.Filters {
		if filter != "" {
			query.Add("filter", filter)
		}
	}
	if len(options.Fields) > 0 {
		query.Set("fields", strings.Join(options.Fields, ","))
	}
	for _, format := range options.Formats {
		if format != "" {
			query.Add("format", format)
		}
	}
	for _, sortValue := range options.Sort {
		if sortValue != "" {
			query.Add("sort", sortValue)
		}
	}
	if options.Start >= 0 {
		query.Set("start", fmt.Sprintf("%d", options.Start))
	}
	if options.Count >= 0 {
		query.Set("count", fmt.Sprintf("%d", options.Count))
	}
	if options.WithMeta {
		query.Set("with_meta", "true")
	}
	if options.Datasource {
		query.Set("datasource", "true")
	}
}

func addLogReadOptions(query url.Values, options ReadOptions) {
	for _, filter := range options.Filters {
		if filter != "" {
			query.Add("filter", filter)
		}
	}
	if len(options.Fields) > 0 {
		query.Set("fields", strings.Join(options.Fields, ","))
	}
	for _, format := range options.Formats {
		if format != "" {
			query.Add("format", format)
		}
	}
	for _, sortValue := range options.Sort {
		if sortValue != "" {
			query.Add("sort", sortValue)
		}
	}
	if options.Start >= 0 {
		query.Set("start", fmt.Sprintf("%d", options.Start))
	}
	if options.Count >= 0 {
		query.Set("rows", fmt.Sprintf("%d", options.Count))
	}
	if options.WithMeta {
		query.Set("with_meta", "true")
	}
	if options.Datasource {
		query.Set("datasource", "true")
	}
}

func (c *Client) applyHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
}

func (c *Client) decodeError(operation string, statusCode int, body []byte) error {
	var envelope Envelope
	if err := json.Unmarshal(body, &envelope); err == nil {
		message := envelope.Message
		if message == "" {
			message = envelope.Status
		}
		return &APIError{
			Operation: operation,
			Code:      statusCode,
			Message:   message,
			Detail:    envelope,
		}
	}

	return &APIError{
		Operation: operation,
		Code:      statusCode,
		Message:   strings.TrimSpace(string(body)),
	}
}
