package fortigate

type PageRange struct {
	Start int `json:"start,omitempty"`
	End   int `json:"end,omitempty"`
}

type EnvelopeMeta struct {
	Count int        `json:"count,omitempty"`
	Range *PageRange `json:"range,omitempty"`
	Next  string     `json:"next,omitempty"`
}

type Envelope struct {
	HTTPMethod string        `json:"http_method,omitempty"`
	Status     string        `json:"status,omitempty"`
	HTTPStatus int           `json:"http_status,omitempty"`
	Path       string        `json:"path,omitempty"`
	Name       string        `json:"name,omitempty"`
	VDOM       string        `json:"vdom,omitempty"`
	Serial     string        `json:"serial,omitempty"`
	Version    string        `json:"version,omitempty"`
	Build      int           `json:"build,omitempty"`
	Revision   string        `json:"revision,omitempty"`
	Results    any           `json:"results,omitempty"`
	Meta       *EnvelopeMeta `json:"meta,omitempty"`
	Count      int           `json:"count,omitempty"`
	Range      *PageRange    `json:"range,omitempty"`
	Next       string        `json:"next,omitempty"`
	Error      int           `json:"error,omitempty"`
	Message    string        `json:"message,omitempty"`
}

func (e *Envelope) Paging() EnvelopeMeta {
	if e == nil {
		return EnvelopeMeta{}
	}

	meta := EnvelopeMeta{
		Count: e.Count,
		Range: e.Range,
		Next:  e.Next,
	}
	if e.Meta == nil {
		return meta
	}
	if meta.Count == 0 {
		meta.Count = e.Meta.Count
	}
	if meta.Range == nil {
		meta.Range = e.Meta.Range
	}
	if meta.Next == "" {
		meta.Next = e.Meta.Next
	}
	return meta
}

type VDOMResult struct {
	VDOM     string    `json:"vdom"`
	Envelope *Envelope `json:"envelope,omitempty"`
	Error    string    `json:"error,omitempty"`
}

type MultiVDOMEnvelope struct {
	Path    string       `json:"path"`
	Mode    string       `json:"mode"`
	Results []VDOMResult `json:"results"`
}

type APIError struct {
	Operation string `json:"operation"`
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Detail    any    `json:"detail,omitempty"`
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return "fortigate api error"
}

type DiscoveryTarget string

const (
	DiscoveryTargetCMDB    DiscoveryTarget = "cmdb"
	DiscoveryTargetMonitor DiscoveryTarget = "monitor"
)

type DiscoverySchemaOptions struct {
	WithMeta bool
}

type DiscoveryFieldOptions struct {
	Filters    []string
	Count      int
	WithMeta   bool
	Datasource bool
}

type DiscoveryCapabilityOptions struct {
	Probe bool
}

type SchemaReport struct {
	Target   DiscoveryTarget `json:"target"`
	Path     string          `json:"path"`
	Endpoint string          `json:"endpoint"`
	Source   string          `json:"source"`
	Schema   any             `json:"schema,omitempty"`
	Error    string          `json:"error,omitempty"`
}

type FieldReport struct {
	Target        DiscoveryTarget     `json:"target"`
	Path          string              `json:"path"`
	SampleCount   int                 `json:"sample_count"`
	Fields        []string            `json:"fields"`
	InferredTypes map[string][]string `json:"inferred_types,omitempty"`
	Source        string              `json:"source"`
}

type DiscoveryProbeResult struct {
	SchemaSupported bool   `json:"schema_supported"`
	Error           string `json:"error,omitempty"`
}

type CapabilityReport struct {
	Target                   DiscoveryTarget       `json:"target"`
	Path                     string                `json:"path"`
	SupportsSchema           bool                  `json:"supports_schema"`
	SupportsFieldExploration bool                  `json:"supports_field_exploration"`
	SupportedQueryFlags      map[string][]string   `json:"supported_query_flags"`
	ProbeResult              *DiscoveryProbeResult `json:"probe_result,omitempty"`
}
