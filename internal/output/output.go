package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode"
)

type CLIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  any    `json:"detail,omitempty"`
}

type ShapeOptions struct {
	Query      string
	Select     []string
	Flatten    bool
	FlattenSep string
	Columns    []string
}

type shapeResult struct {
	value   any
	rows    []map[string]any
	columns []string
}

type selectorToken struct {
	field    string
	index    *int
	wildcard bool
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

func WriteShaped(w io.Writer, format string, value any, opts ShapeOptions) error {
	if !opts.enabled() {
		return Write(w, format, value)
	}

	result, err := shapeValue(value, opts)
	if err != nil {
		return err
	}

	switch strings.ToLower(format) {
	case "", "json":
		return writeJSON(w, result.value)
	case "table":
		return writeRowsTable(w, result.rows, result.columns)
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

func (o ShapeOptions) enabled() bool {
	return o.Query != "" || len(o.Select) > 0 || o.Flatten || len(o.Columns) > 0
}

func shapeValue(value any, opts ShapeOptions) (shapeResult, error) {
	target := value
	if opts.Query != "" {
		var err error
		target, err = applySelector(value, opts.Query)
		if err != nil {
			return shapeResult{}, err
		}
	} else {
		target = defaultShapeTarget(value)
	}

	columns := []string(nil)
	if len(opts.Select) > 0 {
		var err error
		target, columns, err = projectValue(target, opts.Select)
		if err != nil {
			return shapeResult{}, err
		}
	}

	if opts.Flatten {
		target = flattenValue(target, defaultFlattenSep(opts.FlattenSep))
	}

	rows, discoveredColumns := normalizeRows(target)
	if len(columns) == 0 {
		columns = discoveredColumns
	}
	if len(opts.Columns) > 0 {
		columns = append([]string(nil), opts.Columns...)
	}

	return shapeResult{
		value:   target,
		rows:    rows,
		columns: columns,
	}, nil
}

func defaultShapeTarget(value any) any {
	envelope, ok := value.(map[string]any)
	if !ok {
		return value
	}

	results, ok := envelope["results"]
	if !ok {
		return value
	}
	return results
}

func projectValue(value any, selectors []string) (any, []string, error) {
	columns := selectorLabels(selectors)
	switch typed := value.(type) {
	case []any:
		rows := make([]any, 0, len(typed))
		for _, item := range typed {
			row, err := projectRow(item, selectors, columns)
			if err != nil {
				return nil, nil, err
			}
			rows = append(rows, row)
		}
		return rows, columns, nil
	default:
		row, err := projectRow(value, selectors, columns)
		if err != nil {
			return nil, nil, err
		}
		return row, columns, nil
	}
}

func projectRow(value any, selectors, columns []string) (map[string]any, error) {
	row := make(map[string]any, len(selectors))
	for index, selector := range selectors {
		selected, err := applySelector(value, selector)
		if err != nil {
			return nil, err
		}
		row[columns[index]] = selected
	}
	return row, nil
}

func selectorLabels(selectors []string) []string {
	labels := make([]string, 0, len(selectors))
	for _, selector := range selectors {
		labels = append(labels, selectorLabel(selector))
	}
	return labels
}

func selectorLabel(selector string) string {
	trimmed := strings.TrimSpace(selector)
	trimmed = strings.TrimPrefix(trimmed, ".")
	if trimmed == "" {
		return "value"
	}
	return trimmed
}

func defaultFlattenSep(sep string) string {
	if sep == "" {
		return "."
	}
	return sep
}

func flattenValue(value any, sep string) any {
	switch typed := value.(type) {
	case []any:
		rows := make([]any, 0, len(typed))
		for _, item := range typed {
			rows = append(rows, flattenValue(item, sep))
		}
		return rows
	case map[string]any:
		flat := map[string]any{}
		flattenInto(flat, "", typed, sep)
		return flat
	default:
		return value
	}
}

func flattenInto(dst map[string]any, prefix string, value any, sep string) {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			flattenInto(dst, joinPath(prefix, key, sep), typed[key], sep)
		}
	case []any:
		for index, item := range typed {
			flattenInto(dst, joinPath(prefix, strconv.Itoa(index), sep), item, sep)
		}
	default:
		dst[prefix] = typed
	}
}

func joinPath(prefix, segment, sep string) string {
	if prefix == "" {
		return segment
	}
	if segment == "" {
		return prefix
	}
	return prefix + sep + segment
}

func normalizeRows(value any) ([]map[string]any, []string) {
	switch typed := value.(type) {
	case []any:
		rows := make([]map[string]any, 0, len(typed))
		columns := []string{}
		seen := map[string]struct{}{}
		for _, item := range typed {
			row := normalizeRow(item)
			rows = append(rows, row)
			for _, key := range orderedKeys(row) {
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				columns = append(columns, key)
			}
		}
		return rows, columns
	default:
		row := normalizeRow(value)
		return []map[string]any{row}, orderedKeys(row)
	}
}

func normalizeRow(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		row := make(map[string]any, len(typed))
		for key, item := range typed {
			row[key] = item
		}
		return row
	default:
		return map[string]any{"value": typed}
	}
}

func orderedKeys(value map[string]any) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func applySelector(value any, selector string) (any, error) {
	tokens, err := parseSelector(selector)
	if err != nil {
		return nil, err
	}

	current := []any{value}
	wildcardSeen := false
	for _, token := range tokens {
		next := make([]any, 0)
		for _, item := range current {
			values, expanded := selectToken(item, token)
			if expanded {
				wildcardSeen = true
			}
			next = append(next, values...)
		}
		current = next
	}

	if len(current) == 0 {
		if wildcardSeen {
			return []any{}, nil
		}
		return nil, nil
	}
	if wildcardSeen || len(current) > 1 {
		return current, nil
	}
	return current[0], nil
}

func selectToken(value any, token selectorToken) ([]any, bool) {
	if token.wildcard {
		items, ok := value.([]any)
		if !ok {
			return nil, true
		}
		return append([]any(nil), items...), true
	}

	if token.index != nil {
		items, ok := value.([]any)
		if !ok {
			return []any{nil}, false
		}
		index := *token.index
		if index < 0 || index >= len(items) {
			return []any{nil}, false
		}
		return []any{items[index]}, false
	}

	object, ok := value.(map[string]any)
	if !ok {
		return []any{nil}, false
	}
	return []any{object[token.field]}, false
}

func parseSelector(selector string) ([]selectorToken, error) {
	input := strings.TrimSpace(selector)
	if input == "" || input == "." {
		return nil, nil
	}

	index := 0
	tokens := make([]selectorToken, 0)
	if input[0] != '.' && input[0] != '[' {
		field, next := readIdentifier(input, 0)
		if field == "" {
			return nil, fmt.Errorf("invalid selector: %s", selector)
		}
		tokens = append(tokens, selectorToken{field: field})
		index = next
	}

	for index < len(input) {
		switch input[index] {
		case '.':
			index++
			field, next := readIdentifier(input, index)
			if field == "" {
				return nil, fmt.Errorf("invalid selector: %s", selector)
			}
			tokens = append(tokens, selectorToken{field: field})
			index = next
		case '[':
			end := strings.IndexByte(input[index:], ']')
			if end < 0 {
				return nil, fmt.Errorf("invalid selector: %s", selector)
			}
			end += index
			content := input[index+1 : end]
			switch content {
			case "", "*":
				tokens = append(tokens, selectorToken{wildcard: true})
			default:
				parsed, err := strconv.Atoi(content)
				if err != nil {
					return nil, fmt.Errorf("invalid selector: %s", selector)
				}
				tokenIndex := parsed
				tokens = append(tokens, selectorToken{index: &tokenIndex})
			}
			index = end + 1
		default:
			return nil, fmt.Errorf("invalid selector: %s", selector)
		}
	}

	return tokens, nil
}

func readIdentifier(input string, start int) (string, int) {
	index := start
	for index < len(input) {
		r := rune(input[index])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			index++
			continue
		}
		break
	}
	return input[start:index], index
}

func writeTable(w io.Writer, value any) error {
	switch v := value.(type) {
	case map[string]any:
		return writeMapTable(w, v)
	default:
		return writeJSON(w, value)
	}
}

func writeRowsTable(w io.Writer, rows []map[string]any, columns []string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if len(columns) > 0 {
		if _, err := fmt.Fprintln(tw, strings.Join(columns, "\t")); err != nil {
			return err
		}
	}
	for _, row := range rows {
		values := make([]string, 0, len(columns))
		for _, column := range columns {
			values = append(values, stringifyCell(row[column]))
		}
		if _, err := fmt.Fprintln(tw, strings.Join(values, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
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

func stringifyCell(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		data, err := json.Marshal(typed)
		if err == nil {
			return string(data)
		}
		return fmt.Sprint(typed)
	}
}
