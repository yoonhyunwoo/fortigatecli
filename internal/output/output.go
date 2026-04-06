package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
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
	if rows, ok := extractTableRows(value); ok {
		return writeRowsTable(w, rows)
	}
	switch v := value.(type) {
	case map[string]any:
		return writeMapTable(w, v)
	default:
		return writeJSON(w, value)
	}
}

func extractTableRows(value any) ([]map[string]any, bool) {
	if value == nil {
		return nil, false
	}

	switch typed := value.(type) {
	case []map[string]any:
		return typed, true
	case []any:
		rows := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			row, ok := item.(map[string]any)
			if !ok {
				return nil, false
			}
			rows = append(rows, row)
		}
		return rows, true
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Pointer && !rv.IsNil() {
		return extractTableRows(rv.Elem().Interface())
	}
	if rv.Kind() != reflect.Struct {
		return nil, false
	}

	data, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, false
	}

	results, ok := payload["results"]
	if !ok {
		return nil, false
	}
	return extractTableRows(results)
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

func writeRowsTable(w io.Writer, rows []map[string]any) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, "no results")
		return err
	}

	columns := make([]string, 0)
	seen := make(map[string]struct{})
	for _, row := range rows {
		for key := range row {
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			columns = append(columns, key)
		}
	}
	sort.Strings(columns)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, strings.Join(columns, "\t")); err != nil {
		return err
	}
	for _, row := range rows {
		values := make([]string, 0, len(columns))
		for _, column := range columns {
			values = append(values, fmt.Sprintf("%v", row[column]))
		}
		if _, err := fmt.Fprintln(tw, strings.Join(values, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}
