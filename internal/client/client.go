package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const MaxLimit = 10000

type Config struct {
	BaseURL       string
	APIVersion    string
	AppIdentifier string
	HTTPClient    *http.Client
	Timeout       time.Duration
}

type Client struct {
	cfg Config
}

type Page struct {
	Limit  int
	Offset int
}

type Options struct {
	Limit    int
	Offset   int
	AllPages bool
}

type Response struct {
	Data    []map[string]any
	Partial bool
}

type Error struct {
	Code      string
	Message   string
	Retryable bool
	exitCode  int
}

func (e *Error) Error() string { return e.Message }
func (e *Error) ExitCode() int {
	if e.exitCode == 0 {
		return 1
	}
	return e.exitCode
}

func New(cfg Config) *Client {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Client{cfg: cfg}
}

func (c *Client) BuildURL(endpoint string, params url.Values, page Page, outputFormat string) (*url.URL, error) {
	base, err := url.Parse(strings.TrimRight(c.cfg.BaseURL, "/"))
	if err != nil {
		return nil, err
	}
	endpoint = strings.TrimPrefix(endpoint, "/")
	base.Path = path.Join(base.Path, c.cfg.APIVersion, endpoint)

	q := base.Query()
	for key, values := range params {
		for _, value := range values {
			q.Add(key, value)
		}
	}
	q.Set("app_identifier", c.cfg.AppIdentifier)
	q.Set("output_format", outputFormat)
	q.Set("limit", fmt.Sprintf("%d", page.Limit))
	q.Set("offset", fmt.Sprintf("%d", page.Offset))
	base.RawQuery = q.Encode()
	return base, nil
}

func (c *Client) Fetch(ctx context.Context, endpoint string, params url.Values, opts Options) (Response, error) {
	if opts.Limit == 0 {
		opts.Limit = MaxLimit
	}
	if opts.Limit > MaxLimit {
		opts.Limit = MaxLimit
	}
	if params == nil {
		params = url.Values{}
	}
	if !opts.AllPages {
		data, err := c.fetchPage(ctx, endpoint, params, Page{Limit: opts.Limit, Offset: opts.Offset})
		if err != nil {
			return Response{}, err
		}
		if len(data) == 0 {
			return Response{}, &Error{Code: "no_data", Message: "No data returned for query.", Retryable: false, exitCode: 4}
		}
		return Response{Data: data}, nil
	}

	all := []map[string]any{}
	offset := opts.Offset
	for {
		pageData, err := c.fetchPage(ctx, endpoint, params, Page{Limit: opts.Limit, Offset: offset})
		if err != nil {
			if len(all) > 0 {
				return Response{Data: all, Partial: true}, &Error{Code: "partial_data", Message: err.Error(), Retryable: true, exitCode: 5}
			}
			return Response{}, err
		}
		all = append(all, pageData...)
		if len(pageData) < opts.Limit {
			break
		}
		offset += opts.Limit
	}
	if len(all) == 0 {
		return Response{}, &Error{Code: "no_data", Message: "No data returned for query.", Retryable: false, exitCode: 4}
	}
	return Response{Data: all}, nil
}

func (c *Client) FetchCSV(ctx context.Context, endpoint string, params url.Values, opts Options) ([]byte, error) {
	if opts.Limit == 0 {
		opts.Limit = MaxLimit
	}
	u, err := c.BuildURL(endpoint, params, Page{Limit: opts.Limit, Offset: opts.Offset}, "csv")
	if err != nil {
		return nil, &Error{Code: "invalid_url", Message: err.Error(), Retryable: false, exitCode: 1}
	}
	body, status, err := c.do(ctx, u)
	if err != nil {
		return nil, err
	}
	if status < 200 || status > 299 {
		return nil, httpError(status, body)
	}
	return body, nil
}

func (c *Client) fetchPage(ctx context.Context, endpoint string, params url.Values, page Page) ([]map[string]any, error) {
	u, err := c.BuildURL(endpoint, params, page, "json")
	if err != nil {
		return nil, &Error{Code: "invalid_url", Message: err.Error(), Retryable: false, exitCode: 1}
	}
	body, status, err := c.do(ctx, u)
	if err != nil {
		return nil, err
	}
	if status < 200 || status > 299 {
		return nil, httpError(status, body)
	}

	var envelope struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		var bare []map[string]any
		if err2 := json.Unmarshal(body, &bare); err2 == nil {
			return bare, nil
		}
		return nil, &Error{Code: "invalid_json", Message: "HAPI returned malformed JSON.", Retryable: true, exitCode: 3}
	}
	if envelope.Data == nil {
		return nil, &Error{Code: "invalid_json", Message: "HAPI JSON response did not contain a data array.", Retryable: true, exitCode: 3}
	}
	return envelope.Data, nil
}

func (c *Client) do(ctx context.Context, u *url.URL) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, &Error{Code: "invalid_request", Message: err.Error(), Retryable: false, exitCode: 1}
	}
	req.Header.Set("Accept", "application/json, text/csv")
	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, &Error{Code: "network_error", Message: err.Error(), Retryable: true, exitCode: 3}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, &Error{Code: "network_error", Message: err.Error(), Retryable: true, exitCode: 3}
	}
	return body, resp.StatusCode, nil
}

func httpError(status int, body []byte) error {
	code := "hapi_error"
	exit := 3
	retry := true
	if status == http.StatusBadRequest || status == http.StatusUnprocessableEntity {
		code = "hapi_validation_error"
		exit = 2
		retry = false
	}
	msg := strings.TrimSpace(string(bytes.TrimSpace(body)))
	if msg == "" {
		msg = fmt.Sprintf("HAPI returned HTTP %d.", status)
	}
	return &Error{Code: code, Message: msg, Retryable: retry, exitCode: exit}
}

func ExitCode(err error) int {
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return 1
}
