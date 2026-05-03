package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func testClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func jsonResponse(status int, value any) *http.Response {
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(value)
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(&buf),
		Header:     http.Header{"Content-Type": {"application/json"}},
	}
}

func TestBuildURLAddsStandardParameters(t *testing.T) {
	c := New(Config{
		BaseURL:       "https://example.test/api",
		APIVersion:    "v2",
		AppIdentifier: "abc",
		HTTPClient:    http.DefaultClient,
		Timeout:       time.Second,
	})
	u, err := c.BuildURL("metadata/location", url.Values{"name": {"Sudan"}}, Page{Limit: 100, Offset: 20}, "json")
	if err != nil {
		t.Fatal(err)
	}
	if u.String() != "https://example.test/api/v2/metadata/location?app_identifier=abc&limit=100&name=Sudan&offset=20&output_format=json" {
		t.Fatalf("url = %s", u.String())
	}
}

func TestFetchSinglePageSuccess(t *testing.T) {
	var seenPath string
	var seenQuery url.Values
	httpClient := testClient(func(r *http.Request) (*http.Response, error) {
		seenPath = r.URL.Path
		seenQuery = r.URL.Query()
		return jsonResponse(http.StatusOK, map[string]any{
			"data": []map[string]any{{"code": "SDN", "name": "Sudan"}},
		}), nil
	})

	c := New(Config{BaseURL: "https://example.test/api", APIVersion: "v2", AppIdentifier: "id", HTTPClient: httpClient, Timeout: time.Second})
	resp, err := c.Fetch(context.Background(), "metadata/location", url.Values{"name": {"Sudan"}}, Options{Limit: 100, Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	if seenPath != "/api/v2/metadata/location" {
		t.Fatalf("path = %q", seenPath)
	}
	if seenQuery.Get("app_identifier") != "id" || seenQuery.Get("output_format") != "json" {
		t.Fatalf("query = %s", seenQuery.Encode())
	}
	if len(resp.Data) != 1 || resp.Data[0]["code"] != "SDN" {
		t.Fatalf("data = %#v", resp.Data)
	}
}

func TestFetchAllPagesStopsWhenPageShorterThanLimit(t *testing.T) {
	calls := 0
	httpClient := testClient(func(r *http.Request) (*http.Response, error) {
		calls++
		offset := r.URL.Query().Get("offset")
		data := []map[string]any{{"offset": offset}, {"offset": offset}}
		if offset == "2" {
			data = data[:1]
		}
		return jsonResponse(http.StatusOK, map[string]any{"data": data}), nil
	})

	c := New(Config{BaseURL: "https://example.test/api", APIVersion: "v2", AppIdentifier: "id", HTTPClient: httpClient, Timeout: time.Second})
	resp, err := c.Fetch(context.Background(), "metadata/location", nil, Options{Limit: 2, AllPages: true})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d", calls)
	}
	if len(resp.Data) != 3 {
		t.Fatalf("len(data) = %d", len(resp.Data))
	}
}

func TestFetchEmptyDataReturnsNoDataError(t *testing.T) {
	httpClient := testClient(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, map[string]any{"data": []map[string]any{}}), nil
	})

	c := New(Config{BaseURL: "https://example.test/api", APIVersion: "v2", AppIdentifier: "id", HTTPClient: httpClient, Timeout: time.Second})
	_, err := c.Fetch(context.Background(), "metadata/location", nil, Options{Limit: 100})
	if err == nil {
		t.Fatal("Fetch succeeded with empty data")
	}
	if got := ExitCode(err); got != 4 {
		t.Fatalf("ExitCode = %d", got)
	}
}

func TestFetchNon200MapsBadRequestAndNetworkErrors(t *testing.T) {
	httpClient := testClient(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(`{"detail":"bad"}`)),
		}, nil
	})

	c := New(Config{BaseURL: "https://example.test/api", APIVersion: "v2", AppIdentifier: "id", HTTPClient: httpClient, Timeout: time.Second})
	_, err := c.Fetch(context.Background(), "metadata/location", nil, Options{Limit: 100})
	if err == nil {
		t.Fatal("Fetch succeeded with 400")
	}
	if got := ExitCode(err); got != 2 {
		t.Fatalf("ExitCode = %d", got)
	}
}

func TestFetchMalformedJSONMapsNetworkError(t *testing.T) {
	httpClient := testClient(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{`)),
		}, nil
	})

	c := New(Config{BaseURL: "https://example.test/api", APIVersion: "v2", AppIdentifier: "id", HTTPClient: httpClient, Timeout: time.Second})
	_, err := c.Fetch(context.Background(), "metadata/location", nil, Options{Limit: 100})
	if err == nil {
		t.Fatal("Fetch succeeded with malformed JSON")
	}
	if got := ExitCode(err); got != 3 {
		t.Fatalf("ExitCode = %d", got)
	}
}
