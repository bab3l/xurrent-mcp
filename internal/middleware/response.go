package middleware

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ResponseShaper controls how tool results are formatted for the agent.
type ResponseShaper struct {
	MaxChars int    // Maximum characters before truncation (0 = no limit).
	Compact  bool   // Strip decorative whitespace from JSON.
}

// NewResponseShaper reads configuration from environment:
//
//	XURRENT_TOOL_MAX_RESPONSE_CHARS — default 120000
//	XURRENT_TOOL_JSON_COMPACT — "0" or "false" disables compact mode; default enabled
func NewResponseShaper() *ResponseShaper {
	rs := &ResponseShaper{
		MaxChars: 120000,
		Compact:  true,
	}
	if v := strings.TrimSpace(getEnvStr("XURRENT_TOOL_MAX_RESPONSE_CHARS", "120000")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rs.MaxChars = n
		}
	}
	// Compact is enabled by default; disable if explicitly set to "0" or "false".
	if v := strings.TrimSpace(strings.ToLower(os.Getenv("XURRENT_TOOL_JSON_COMPACT"))); v == "0" || v == "false" {
		rs.Compact = false
	}
	return rs
}

// ShapeJSON marshals v to JSON applying compact mode and truncation.
// Returns the string and a bool indicating whether truncation occurred.
func (rs *ResponseShaper) ShapeJSON(v interface{}) (string, bool, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if rs.Compact {
		enc.SetEscapeHTML(false)
	}
	if err := enc.Encode(v); err != nil {
		return "", false, fmt.Errorf("shape: marshal: %w", err)
	}
	s := buf.String()
	// Strip trailing newline added by Encode.
	s = strings.TrimRight(s, "\n")
	if rs.MaxChars > 0 && len(s) > rs.MaxChars {
		truncated := s[:rs.MaxChars]
		truncated += fmt.Sprintf("\n...truncated (%d total chars, %d shown)", len(s), rs.MaxChars)
		return truncated, true, nil
	}
	return s, false, nil
}

// ShapeCSV converts rows ([]map[string]interface{}) to CSV format with truncation.
func (rs *ResponseShaper) ShapeCSV(rows []map[string]interface{}, columns []string) (string, bool, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if len(columns) > 0 {
		_ = w.Write(columns)
	}
	for _, row := range rows {
		rec := make([]string, len(columns))
		for i, col := range columns {
			if v, ok := row[col]; ok {
				rec[i] = fmt.Sprint(v)
			}
		}
		_ = w.Write(rec)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", false, fmt.Errorf("shape: csv: %w", err)
	}
	s := buf.String()
	rowCount := len(rows)
	if rs.MaxChars > 0 && len(s) > rs.MaxChars {
		cut := rs.MaxChars
		if idx := strings.LastIndex(s[:cut], "\n"); idx > 0 {
			cut = idx
		}
		truncated := s[:cut]
		truncated += fmt.Sprintf("\n...truncated (%d rows total, %d chars total)", rowCount, len(s))
		return truncated, true, nil
	}
	return s, false, nil
}

// TruncateText truncates a plain text string with a note.
func (rs *ResponseShaper) TruncateText(s string) (string, bool) {
	if rs.MaxChars > 0 && len(s) > rs.MaxChars {
		return s[:rs.MaxChars] + fmt.Sprintf("\n...truncated (%d total chars)", len(s)), true
	}
	return s, false
}

func getEnvStr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
