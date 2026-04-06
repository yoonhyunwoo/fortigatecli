package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
)

type CLIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  any    `json:"detail,omitempty"`
}

func (e *CLIError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func Write(w io.Writer, format string, value any) error {
	switch strings.ToLower(format) {
	case "", "json":
		return writeJSON(w, value)
	case "table":
		return writeTable(w, value)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func WriteError(err error) {
	cliErr, ok := err.(*CLIError)
	if !ok {
		cliErr = &CLIError{
			Code:    "internal_error",
			Message: err.Error(),
		}
	}

	_ = writeJSON(os.Stderr, cliErr)
}

func NewError(code, message string, detail any) *CLIError {
	return &CLIError{
		Code:    code,
		Message: message,
		Detail:  detail,
	}
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeTable(w io.Writer, value any) error {
	switch v := value.(type) {
	case map[string]any:
		return writeMapTable(w, v)
	default:
		return writeJSON(w, value)
	}
}

func writeMapTable(w io.Writer, value map[string]any) error {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, key := range keys {
		if _, err := fmt.Fprintf(tw, "%s\t%v\n", key, value[key]); err != nil {
			return err
		}
	}
	return tw.Flush()
}
