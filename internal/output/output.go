package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
)

type Envelope struct {
	OK         bool                `json:"ok"`
	Source     string              `json:"source"`
	APIVersion string              `json:"api_version"`
	Endpoint   string              `json:"endpoint"`
	Query      map[string][]string `json:"query"`
	Count      int                 `json:"count"`
	Data       []map[string]any    `json:"data"`
	Meta       Meta                `json:"meta"`
}

type Meta struct {
	Limit    int      `json:"limit"`
	Offset   int      `json:"offset"`
	AllPages bool     `json:"all_pages"`
	Warnings []string `json:"warnings"`
}

type ErrorEnvelope struct {
	OK    bool       `json:"ok"`
	Error ErrorBlock `json:"error"`
}

type ErrorBlock struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func WriteJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(value)
}

func WriteJSONL(w io.Writer, rows []map[string]any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for _, row := range rows {
		if err := enc.Encode(row); err != nil {
			return err
		}
	}
	return nil
}

func WriteCSV(w io.Writer, rows []map[string]any, fields []string) error {
	if len(rows) == 0 {
		return nil
	}
	if len(fields) == 0 {
		fields = sortedFields(rows)
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(fields); err != nil {
		return err
	}
	for _, row := range rows {
		record := make([]string, len(fields))
		for i, field := range fields {
			record[i] = fmt.Sprint(row[field])
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func WriteTable(w io.Writer, rows []map[string]any, fields []string) error {
	if len(rows) == 0 {
		return nil
	}
	if len(fields) == 0 {
		fields = sortedFields(rows)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, strings.Join(fields, "\t")); err != nil {
		return err
	}
	for _, row := range rows {
		record := make([]string, len(fields))
		for i, field := range fields {
			record[i] = fmt.Sprint(row[field])
		}
		if _, err := fmt.Fprintln(tw, strings.Join(record, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func SelectFields(rows []map[string]any, fields []string) []map[string]any {
	if len(fields) == 0 {
		return rows
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		next := map[string]any{}
		for _, field := range fields {
			if value, ok := row[field]; ok {
				next[field] = value
			}
		}
		out = append(out, next)
	}
	return out
}

func sortedFields(rows []map[string]any) []string {
	seen := map[string]bool{}
	for _, row := range rows {
		for key := range row {
			seen[key] = true
		}
	}
	fields := make([]string, 0, len(seen))
	for key := range seen {
		fields = append(fields, key)
	}
	sort.Strings(fields)
	return fields
}
