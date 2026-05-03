package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePrecedenceFlagEnvFileDefault(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.toml")
	err := os.WriteFile(configPath, []byte(`
base_url = "https://file.example/api"
api_version = "v1"
app_identifier = "from-file"
limit = 500
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	flags := FlagValues{
		BaseURL:       "https://flag.example/api",
		AppIdentifier: "from-flag",
	}
	env := map[string]string{
		"HAPI_BASE_URL":       "https://env.example/api",
		"HAPI_API_VERSION":    "v2",
		"HAPI_APP_IDENTIFIER": "from-env",
	}

	cfg, err := Resolve(flags, env, configPath)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if cfg.BaseURL != "https://flag.example/api" {
		t.Fatalf("BaseURL = %q", cfg.BaseURL)
	}
	if cfg.APIVersion != "v2" {
		t.Fatalf("APIVersion = %q", cfg.APIVersion)
	}
	if cfg.AppIdentifier != "from-flag" {
		t.Fatalf("AppIdentifier = %q", cfg.AppIdentifier)
	}
	if cfg.Limit != 500 {
		t.Fatalf("Limit = %d", cfg.Limit)
	}
}

func TestResolveDefaults(t *testing.T) {
	cfg, err := Resolve(FlagValues{}, map[string]string{}, filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if cfg.BaseURL != DefaultBaseURL {
		t.Fatalf("BaseURL = %q", cfg.BaseURL)
	}
	if cfg.APIVersion != DefaultAPIVersion {
		t.Fatalf("APIVersion = %q", cfg.APIVersion)
	}
	if cfg.Format != "json" {
		t.Fatalf("Format = %q", cfg.Format)
	}
	if cfg.Limit != DefaultLimit {
		t.Fatalf("Limit = %d", cfg.Limit)
	}
}

func TestResolveSupportsHDXEnvironmentAliases(t *testing.T) {
	cfg, err := Resolve(FlagValues{}, map[string]string{
		"HDX_APP_IDENTIFIER": "from-hdx",
		"HDX_BASE_URL":       "https://hapi.humdata.org",
		"HDX_TIMEOUT":        "45.5",
	}, filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if cfg.AppIdentifier != "from-hdx" {
		t.Fatalf("AppIdentifier = %q", cfg.AppIdentifier)
	}
	if cfg.BaseURL != "https://hapi.humdata.org/api" {
		t.Fatalf("BaseURL = %q", cfg.BaseURL)
	}
	if cfg.TimeoutSeconds != 45.5 {
		t.Fatalf("TimeoutSeconds = %v", cfg.TimeoutSeconds)
	}
}

func TestResolvePrefersHAPIEnvironmentOverHDXAlias(t *testing.T) {
	cfg, err := Resolve(FlagValues{}, map[string]string{
		"HAPI_APP_IDENTIFIER": "from-hapi",
		"HDX_APP_IDENTIFIER":  "from-hdx",
		"HAPI_BASE_URL":       "https://example.org/api",
		"HDX_BASE_URL":        "https://hapi.humdata.org",
	}, filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if cfg.AppIdentifier != "from-hapi" {
		t.Fatalf("AppIdentifier = %q", cfg.AppIdentifier)
	}
	if cfg.BaseURL != "https://example.org/api" {
		t.Fatalf("BaseURL = %q", cfg.BaseURL)
	}
}

func TestValidateRequiresAppIdentifier(t *testing.T) {
	cfg := Config{BaseURL: DefaultBaseURL, APIVersion: "v2", Format: "json", Limit: 100}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate succeeded without app identifier")
	}
	if got := ExitCode(err); got != 1 {
		t.Fatalf("ExitCode = %d", got)
	}
}

func TestValidateCapsLimit(t *testing.T) {
	cfg := Config{BaseURL: DefaultBaseURL, APIVersion: "v2", Format: "json", Limit: 10001, AppIdentifier: "id"}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate succeeded with too-large limit")
	}
}
