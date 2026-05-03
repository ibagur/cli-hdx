package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestGetCommandWritesJSONEnvelopeToStdout(t *testing.T) {
	var requested string
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requested = r.URL.String()
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"code":"WSH","name":"Water Sanitation Hygiene"}]}`)),
			Header:     http.Header{"Content-Type": {"application/json"}},
		}, nil
	})}
	var stdout, stderr bytes.Buffer
	root := NewRootCommand(Options{
		Stdout:     &stdout,
		Stderr:     &stderr,
		HTTPClient: httpClient,
		Env:        map[string]string{"HAPI_APP_IDENTIFIER": "id"},
		ConfigPath: "/no/such/file",
	})
	root.SetArgs([]string{"get", "metadata/sector", "--param", "name=Water"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v; stderr=%s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(requested, "/api/v2/metadata/sector") || !strings.Contains(requested, "app_identifier=id") {
		t.Fatalf("requested = %s", requested)
	}
	var env map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("invalid stdout json: %v\n%s", err, stdout.String())
	}
	if env["ok"] != true || env["endpoint"] != "metadata/sector" {
		t.Fatalf("envelope = %#v", env)
	}
}

func TestAuthInitDoesNotRequireAppIdentifier(t *testing.T) {
	var stdout bytes.Buffer
	root := NewRootCommand(Options{
		Stdout:     &stdout,
		Stderr:     io.Discard,
		Env:        map[string]string{},
		ConfigPath: "/no/such/file",
	})
	root.SetArgs([]string{"auth", "init", "--app-name", "test app", "--email", "agent@example.org"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "app_identifier") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestListEndpointsDoesNotRequireAppIdentifier(t *testing.T) {
	var stdout bytes.Buffer
	root := NewRootCommand(Options{
		Stdout:     &stdout,
		Stderr:     io.Discard,
		Env:        map[string]string{},
		ConfigPath: "/no/such/file",
	})
	root.SetArgs([]string{"list-endpoints"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var env map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("invalid stdout json: %v\n%s", err, stdout.String())
	}
	if env["ok"] != true || env["endpoint"] != "registry/endpoints" {
		t.Fatalf("envelope = %#v", env)
	}
	data, ok := env["data"].([]any)
	if !ok || len(data) == 0 {
		t.Fatalf("data = %#v", env["data"])
	}
	if !strings.Contains(stdout.String(), "coordination-context/conflict-events") {
		t.Fatalf("stdout missing conflict events endpoint: %s", stdout.String())
	}
}

func TestWorkflowConflictEventsWiresFlags(t *testing.T) {
	var requested []string
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requested = append(requested, r.URL.String())
		body := `{"data":[{"code":"SDN","name":"Sudan"}]}`
		if strings.Contains(r.URL.Path, "data-availability") {
			body = `{"data":[{"subcategory":"Conflict Events","location_code":"SDN"}]}`
		}
		if strings.Contains(r.URL.Path, "conflict-events") {
			body = `{"data":[{"events":12,"fatalities":34}]}`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": {"application/json"}},
		}, nil
	})}
	var stdout bytes.Buffer
	root := NewRootCommand(Options{
		Stdout:     &stdout,
		Stderr:     io.Discard,
		HTTPClient: httpClient,
		Env:        map[string]string{"HAPI_APP_IDENTIFIER": "id"},
		ConfigPath: "/no/such/file",
	})
	root.SetArgs([]string{
		"workflow", "conflict-events",
		"--country", "Sudan",
		"--event-type", "battles",
		"--start-date", "2026-01-01",
		"--end-date", "2026-03-31",
		"--admin-level", "1",
		"--admin1-name", "Khartoum",
		"--admin2-name", "Omdurman",
	})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(requested) != 3 {
		t.Fatalf("requests = %#v", requested)
	}
	got := requested[2]
	for _, want := range []string{
		"/api/v2/coordination-context/conflict-events",
		"location_code=SDN",
		"event_type=battles",
		"start_date=2026-01-01",
		"end_date=2026-03-31",
		"admin_level=1",
		"admin1_name=Khartoum",
		"admin2_name=Omdurman",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("request %q missing %q", got, want)
		}
	}
	if strings.Contains(got, "has_hrp") || strings.Contains(got, "in_gho") {
		t.Fatalf("request should not include HRP/GHO filters: %s", got)
	}
}

func TestMissingAppIdentifierReturnsUsageExit(t *testing.T) {
	root := NewRootCommand(Options{
		Stdout:     io.Discard,
		Stderr:     io.Discard,
		Env:        map[string]string{},
		ConfigPath: "/no/such/file",
	})
	root.SetArgs([]string{"metadata", "locations", "--name", "Sudan"})
	err := root.Execute()
	if err == nil {
		t.Fatal("Execute succeeded without app identifier")
	}
	if ExitCode(err) != 1 {
		t.Fatalf("ExitCode = %d", ExitCode(err))
	}
}
