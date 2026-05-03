package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	DefaultBaseURL    = "https://hapi.humdata.org/api"
	DefaultAPIVersion = "v2"
	DefaultFormat     = "json"
	DefaultLimit      = 1000
	DefaultTimeout    = 30.0
	MaxLimit          = 10000
)

type Config struct {
	BaseURL        string
	APIVersion     string
	AppIdentifier  string
	Format         string
	Limit          int
	Offset         int
	AllPages       bool
	Fields         []string
	Output         string
	TimeoutSeconds float64
	Quiet          bool
	Debug          bool
}

type FlagValues struct {
	BaseURL        string
	APIVersion     string
	AppIdentifier  string
	Format         string
	Limit          int
	Offset         int
	AllPages       bool
	Fields         []string
	Output         string
	TimeoutSeconds float64
	Quiet          bool
	Debug          bool
}

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }
func (e *Error) ExitCode() int { return 1 }

func DefaultConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "hapi", "config.toml")
}

func Resolve(flags FlagValues, env map[string]string, configPath string) (Config, error) {
	fileValues, err := parseConfigFile(configPath)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		BaseURL:        DefaultBaseURL,
		APIVersion:     DefaultAPIVersion,
		Format:         DefaultFormat,
		Limit:          DefaultLimit,
		Offset:         0,
		AppIdentifier:  "",
		TimeoutSeconds: DefaultTimeout,
	}

	cfg.BaseURL = first(flags.BaseURL, envValue(env, "HAPI_BASE_URL"), envValue(env, "HDX_BASE_URL"), fileValues["base_url"], cfg.BaseURL)
	cfg.BaseURL = normalizeBaseURL(cfg.BaseURL)
	cfg.APIVersion = first(flags.APIVersion, envValue(env, "HAPI_API_VERSION"), envValue(env, "HDX_API_VERSION"), fileValues["api_version"], cfg.APIVersion)
	cfg.AppIdentifier = first(flags.AppIdentifier, envValue(env, "HAPI_APP_IDENTIFIER"), envValue(env, "HDX_APP_IDENTIFIER"), fileValues["app_identifier"], cfg.AppIdentifier)
	cfg.Format = first(flags.Format, fileValues["format"], cfg.Format)
	cfg.Output = first(flags.Output, fileValues["output"], "")

	if flags.Limit != 0 {
		cfg.Limit = flags.Limit
	} else if v := envValue(env, "HAPI_LIMIT"); v != "" {
		cfg.Limit, err = parseInt("HAPI_LIMIT", v)
		if err != nil {
			return Config{}, err
		}
	} else if v := fileValues["limit"]; v != "" {
		cfg.Limit, err = parseInt("limit", v)
		if err != nil {
			return Config{}, err
		}
	}

	if flags.Offset != 0 {
		cfg.Offset = flags.Offset
	} else if v := fileValues["offset"]; v != "" {
		cfg.Offset, err = parseInt("offset", v)
		if err != nil {
			return Config{}, err
		}
	}

	if flags.TimeoutSeconds != 0 {
		cfg.TimeoutSeconds = flags.TimeoutSeconds
	} else if v := envValue(env, "HAPI_TIMEOUT"); v != "" {
		cfg.TimeoutSeconds, err = parseFloat("HAPI_TIMEOUT", v)
		if err != nil {
			return Config{}, err
		}
	} else if v := envValue(env, "HDX_TIMEOUT"); v != "" {
		cfg.TimeoutSeconds, err = parseFloat("HDX_TIMEOUT", v)
		if err != nil {
			return Config{}, err
		}
	} else if v := fileValues["timeout"]; v != "" {
		cfg.TimeoutSeconds, err = parseFloat("timeout", v)
		if err != nil {
			return Config{}, err
		}
	}

	cfg.AllPages = flags.AllPages || parseBool(fileValues["all_pages"])
	cfg.Fields = flags.Fields
	if len(cfg.Fields) == 0 && fileValues["fields"] != "" {
		cfg.Fields = splitCSV(fileValues["fields"])
	}
	cfg.Quiet = flags.Quiet || parseBool(fileValues["quiet"])
	cfg.Debug = flags.Debug || parseBool(fileValues["debug"])
	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.AppIdentifier) == "" {
		return &Error{Code: "missing_app_identifier", Message: "Set HAPI_APP_IDENTIFIER, HDX_APP_IDENTIFIER, or pass --app-identifier."}
	}
	if c.APIVersion != "v1" && c.APIVersion != "v2" {
		return &Error{Code: "invalid_api_version", Message: "Use --api-version v1 or v2."}
	}
	switch c.Format {
	case "json", "jsonl", "csv", "table":
	default:
		return &Error{Code: "invalid_format", Message: "Use --format json, jsonl, csv, or table."}
	}
	if c.Limit < 0 || c.Limit > MaxLimit {
		return &Error{Code: "invalid_limit", Message: fmt.Sprintf("--limit must be between 0 and %d.", MaxLimit)}
	}
	if c.Offset < 0 {
		return &Error{Code: "invalid_offset", Message: "--offset must be 0 or greater."}
	}
	if c.TimeoutSeconds <= 0 {
		return &Error{Code: "invalid_timeout", Message: "Timeout must be greater than 0 seconds."}
	}
	return nil
}

func ExitCode(err error) int {
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return 1
}

func parseConfigFile(path string) (map[string]string, error) {
	values := map[string]string{}
	if path == "" {
		return values, nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return values, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return nil, &Error{Code: "invalid_config", Message: "Config lines must use key = value syntax."}
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"`)
		values[key] = val
	}
	return values, scanner.Err()
}

func first(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func envValue(env map[string]string, key string) string {
	if env != nil {
		return env[key]
	}
	return os.Getenv(key)
}

func parseInt(name, value string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, &Error{Code: "invalid_config", Message: fmt.Sprintf("%s must be an integer.", name)}
	}
	return n, nil
}

func parseFloat(name, value string) (float64, error) {
	n, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, &Error{Code: "invalid_config", Message: fmt.Sprintf("%s must be a number.", name)}
	}
	return n, nil
}

func parseBool(value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	return v == "1" || v == "true" || v == "yes"
}

func normalizeBaseURL(value string) string {
	v := strings.TrimRight(strings.TrimSpace(value), "/")
	if strings.HasSuffix(v, "/api") {
		return v
	}
	return v + "/api"
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
