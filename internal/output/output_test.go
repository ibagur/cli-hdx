package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestWriteJSONEnvelope(t *testing.T) {
	var buf bytes.Buffer
	env := Envelope{
		OK:         true,
		Source:     "HDX/HAPI",
		APIVersion: "v2",
		Endpoint:   "metadata/location",
		Query:      map[string][]string{"name": {"Sudan"}},
		Count:      1,
		Data:       []map[string]any{{"code": "SDN", "name": "Sudan"}},
		Meta:       Meta{Limit: 1000, Offset: 0, Warnings: []string{}},
	}
	if err := WriteJSON(&buf, env); err != nil {
		t.Fatal(err)
	}
	var got Envelope
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if !got.OK || got.Count != 1 || got.Source != "HDX/HAPI" {
		t.Fatalf("envelope = %#v", got)
	}
}

func TestSelectFieldsKeepsRequestedOrder(t *testing.T) {
	rows := []map[string]any{{"name": "Sudan", "code": "SDN", "extra": true}}
	got := SelectFields(rows, []string{"code", "name"})
	if len(got) != 1 {
		t.Fatalf("rows = %#v", got)
	}
	if _, ok := got[0]["extra"]; ok {
		t.Fatalf("unexpected extra field: %#v", got[0])
	}
}

func TestWriteJSONL(t *testing.T) {
	var buf bytes.Buffer
	err := WriteJSONL(&buf, []map[string]any{{"a": 1}, {"a": 2}})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("line count = %d", len(lines))
	}
}

func TestWriteTable(t *testing.T) {
	var buf bytes.Buffer
	err := WriteTable(&buf, []map[string]any{{"code": "SDN", "name": "Sudan"}}, []string{"code", "name"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "code") || !strings.Contains(buf.String(), "SDN") {
		t.Fatalf("table = %q", buf.String())
	}
}
