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
	"sort"
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
	return c.get(ctx, "/api/v2/monitor/"+strings.TrimPrefix(resourcePath, "/"), options)
}

func (c *Client) GetCMDB(ctx context.Context, resourcePath string, options ReadOptions) (*Envelope, error) {
	return c.get(ctx, "/api/v2/cmdb/"+strings.TrimPrefix(resourcePath, "/"), options)
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
	return c.get(ctx, normalized, options)
}

func (c *Client) GetDiscoverySchema(ctx context.Context, target DiscoveryTarget, resourcePath string, options DiscoverySchemaOptions) (*SchemaReport, error) {
	apiPath, err := discoveryAPIPath(target, resourcePath)
	if err != nil {
		return nil, err
	}

	schemaPath := apiPath + "?action=schema"
	envelope, err := c.get(ctx, schemaPath, ReadOptions{WithMeta: options.WithMeta})
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && isUnsupportedSchemaError(apiErr) {
			return &SchemaReport{
				Target:   target,
				Path:     strings.TrimPrefix(resourcePath, "/"),
				Endpoint: schemaPath,
				Source:   "unsupported",
				Error:    apiErr.Message,
				Schema:   apiErr.Detail,
			}, nil
		}
		return nil, err
	}

	return &SchemaReport{
		Target:   target,
		Path:     strings.TrimPrefix(resourcePath, "/"),
		Endpoint: schemaPath,
		Source:   "api",
		Schema:   envelope.Results,
	}, nil
}

func (c *Client) DiscoverFields(ctx context.Context, target DiscoveryTarget, resourcePath string, options DiscoveryFieldOptions) (*FieldReport, error) {
	apiOptions := ReadOptions{
		Filters:    options.Filters,
		Count:      options.Count,
		WithMeta:   options.WithMeta,
		Datasource: options.Datasource,
	}

	var (
		envelope *Envelope
		err      error
	)
	switch target {
	case DiscoveryTargetCMDB:
		envelope, err = c.GetCMDB(ctx, resourcePath, apiOptions)
	case DiscoveryTargetMonitor:
		envelope, err = c.GetMonitor(ctx, resourcePath, apiOptions)
	default:
		return nil, &APIError{
			Operation: "discover_fields",
			Message:   fmt.Sprintf("unsupported discovery target %q", target),
		}
	}
	if err != nil {
		return nil, err
	}

	fields, inferredTypes, sampleCount := inferFields(envelope.Results)
	return &FieldReport{
		Target:        target,
		Path:          strings.TrimPrefix(resourcePath, "/"),
		SampleCount:   sampleCount,
		Fields:        fields,
		InferredTypes: inferredTypes,
		Source:        "sample",
	}, nil
}

func (c *Client) GetDiscoveryCapabilities(ctx context.Context, target DiscoveryTarget, resourcePath string, options DiscoveryCapabilityOptions) (*CapabilityReport, error) {
	report := &CapabilityReport{
		Target:                   target,
		Path:                     strings.TrimPrefix(resourcePath, "/"),
		SupportsSchema:           true,
		SupportsFieldExploration: true,
		SupportedQueryFlags: map[string][]string{
			"schema":       []string{"with-meta"},
			"fields":       []string{"filter", "count", "with-meta", "datasource"},
			"capabilities": []string{"probe"},
		},
	}

	if !options.Probe {
		return report, nil
	}

	schemaReport, err := c.GetDiscoverySchema(ctx, target, resourcePath, DiscoverySchemaOptions{})
	if err != nil {
		return nil, err
	}
	report.ProbeResult = &DiscoveryProbeResult{
		SchemaSupported: schemaReport.Source == "api",
		Error:           schemaReport.Error,
	}
	return report, nil
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

func (c *Client) get(ctx context.Context, apiPath string, options ReadOptions) (*Envelope, error) {
	parsedPath, err := url.Parse(apiPath)
	if err != nil {
		return nil, fmt.Errorf("parse API path: %w", err)
	}

	u := *c.baseURL
	u.Path = path.Clean(strings.TrimSuffix(c.baseURL.Path, "/") + "/" + strings.TrimPrefix(parsedPath.Path, "/"))
	query := parsedPath.Query()
	query.Set("vdom", c.vdom)
	addReadOptions(query, options)
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

func discoveryAPIPath(target DiscoveryTarget, resourcePath string) (string, error) {
	normalizedPath := strings.TrimPrefix(resourcePath, "/")
	switch target {
	case DiscoveryTargetCMDB:
		return "/api/v2/cmdb/" + normalizedPath, nil
	case DiscoveryTargetMonitor:
		return "/api/v2/monitor/" + normalizedPath, nil
	default:
		return "", &APIError{
			Operation: "discovery_path",
			Message:   fmt.Sprintf("unsupported discovery target %q", target),
		}
	}
}

func isUnsupportedSchemaError(err *APIError) bool {
	if err == nil {
		return false
	}
	if err.Code == http.StatusNotFound || err.Code == http.StatusMethodNotAllowed {
		return true
	}
	if err.Code == http.StatusBadRequest {
		message := strings.ToLower(err.Message)
		return strings.Contains(message, "invalid") || strings.Contains(message, "not found") || strings.Contains(message, "schema")
	}
	return false
}

func inferFields(results any) ([]string, map[string][]string, int) {
	fieldTypes := map[string]map[string]struct{}{}
	sampleCount := 0

	addObjectFields := func(values map[string]any) {
		sampleCount++
		for key, value := range values {
			typesForField, ok := fieldTypes[key]
			if !ok {
				typesForField = map[string]struct{}{}
				fieldTypes[key] = typesForField
			}
			typesForField[valueKind(value)] = struct{}{}
		}
	}

	switch typed := results.(type) {
	case map[string]any:
		addObjectFields(typed)
	case []any:
		for _, item := range typed {
			if object, ok := item.(map[string]any); ok {
				addObjectFields(object)
			}
		}
	}

	fields := make([]string, 0, len(fieldTypes))
	for field := range fieldTypes {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	inferredTypes := make(map[string][]string, len(fieldTypes))
	for _, field := range fields {
		typesForField := make([]string, 0, len(fieldTypes[field]))
		for kind := range fieldTypes[field] {
			typesForField = append(typesForField, kind)
		}
		sort.Strings(typesForField)
		inferredTypes[field] = typesForField
	}

	return fields, inferredTypes, sampleCount
}

func valueKind(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case bool:
		return "bool"
	case string:
		return "string"
	case float64:
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return fmt.Sprintf("%T", value)
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
