package fortigate

type Envelope struct {
	HTTPMethod string `json:"http_method,omitempty"`
	Status     string `json:"status,omitempty"`
	HTTPStatus int    `json:"http_status,omitempty"`
	Path       string `json:"path,omitempty"`
	Name       string `json:"name,omitempty"`
	VDOM       string `json:"vdom,omitempty"`
	Serial     string `json:"serial,omitempty"`
	Version    string `json:"version,omitempty"`
	Build      int    `json:"build,omitempty"`
	Revision   string `json:"revision,omitempty"`
	Results    any    `json:"results,omitempty"`
	Error      int    `json:"error,omitempty"`
	Message    string `json:"message,omitempty"`
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
