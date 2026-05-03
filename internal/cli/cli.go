package cli

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ibagur/cli-hdx/internal/client"
	"github.com/ibagur/cli-hdx/internal/config"
	"github.com/ibagur/cli-hdx/internal/output"
	"github.com/ibagur/cli-hdx/internal/registry"
	"github.com/ibagur/cli-hdx/internal/workflows"
)

type Options struct {
	Stdout     io.Writer
	Stderr     io.Writer
	HTTPClient *http.Client
	Env        map[string]string
	ConfigPath string
}

type state struct {
	opts       Options
	configPath string
	fieldsCSV  string
}

func NewRootCommand(opts Options) *cobra.Command {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	s := &state{opts: opts}
	root := &cobra.Command{
		Use:   "hapi",
		Short: "Agent-first CLI for HDX HAPI",
		Long: `Agent-first CLI for HDX HAPI.

stdout is data; stderr is diagnostics. JSON output is the stable machine-readable contract.

Exit codes:
  0  success
  1  CLI usage or configuration error
  2  HAPI validation or bad request
  3  network, timeout, malformed response, or provider unavailable
  4  no data returned
  5  partial data with one or more page failures`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(opts.Stdout)
	root.SetErr(opts.Stderr)

	p := root.PersistentFlags()
	p.String("format", config.DefaultFormat, "Output format: json, jsonl, csv, table")
	p.Int("limit", config.DefaultLimit, "Maximum rows per page")
	p.Int("offset", 0, "Rows to skip")
	p.Bool("all-pages", false, "Fetch all pages until HAPI returns a short page")
	p.StringVar(&s.fieldsCSV, "fields", "", "Comma-separated fields to include")
	p.String("output", "", "Write data output to a file")
	p.String("api-version", config.DefaultAPIVersion, "HAPI API version: v1 or v2")
	p.String("base-url", config.DefaultBaseURL, "HAPI API base URL")
	p.String("app-identifier", "", "HAPI app identifier")
	p.Bool("quiet", false, "Suppress diagnostics")
	p.Bool("debug", false, "Enable debug diagnostics")
	p.Float64("timeout", config.DefaultTimeout, "Request timeout in seconds")
	p.StringVar(&s.configPath, "config", firstNonEmpty(opts.ConfigPath, config.DefaultConfigPath()), "Config file path")

	root.AddCommand(s.authCommand())
	root.AddCommand(s.metadataCommand())
	root.AddCommand(s.getCommand())
	root.AddCommand(s.listEndpointsCommand())
	root.AddCommand(s.workflowCommand())
	return root
}

func (s *state) authCommand() *cobra.Command {
	var appName, email string
	cmd := &cobra.Command{Use: "auth", Short: "Authentication helpers"}
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Create an app identifier from app name and email",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(appName) == "" || strings.TrimSpace(email) == "" {
				return &config.Error{Code: "missing_auth_fields", Message: "Pass --app-name and --email."}
			}
			identifier := base64.StdEncoding.EncodeToString([]byte(appName + ":" + email))
			env := output.Envelope{
				OK:         true,
				Source:     "HDX/HAPI",
				APIVersion: "v2",
				Endpoint:   "encode_app_identifier",
				Query:      map[string][]string{"application": {appName}, "email": {email}},
				Count:      1,
				Data:       []map[string]any{{"app_identifier": identifier}},
				Meta:       output.Meta{Limit: 1, Offset: 0, Warnings: []string{"Generated locally as base64(app_name:email), matching HAPI documentation."}},
			}
			return output.WriteJSON(s.opts.Stdout, env)
		},
	}
	initCmd.Flags().StringVar(&appName, "app-name", "", "Application name")
	initCmd.Flags().StringVar(&email, "email", "", "Contact email")
	cmd.AddCommand(initCmd)
	return cmd
}

func (s *state) metadataCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "metadata", Short: "HAPI metadata commands"}
	cmd.AddCommand(s.metadataEndpointCommand("locations", "metadata.locations", map[string]string{"name": "name"}))
	cmd.AddCommand(s.metadataEndpointCommand("sectors", "metadata.sectors", map[string]string{"name": "name"}))
	cmd.AddCommand(s.metadataEndpointCommand("availability", "metadata.availability", map[string]string{
		"location-code": "location_code",
		"category":      "category",
		"subcategory":   "subcategory",
	}))
	cmd.AddCommand(s.metadataEndpointCommand("admin1", "metadata.admin1", map[string]string{"location-code": "location_code"}))
	cmd.AddCommand(s.metadataEndpointCommand("admin2", "metadata.admin2", map[string]string{
		"location-code": "location_code",
		"admin1-code":   "admin1_code",
	}))
	return cmd
}

func (s *state) metadataEndpointCommand(use, key string, flags map[string]string) *cobra.Command {
	values := map[string]*string{}
	cmd := &cobra.Command{
		Use:   use,
		Short: "Query " + use + " metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := s.resolveConfig(cmd, true)
			if err != nil {
				return err
			}
			params := url.Values{}
			for flag, param := range flags {
				if val := strings.TrimSpace(*values[flag]); val != "" {
					params.Set(param, val)
				}
			}
			return s.executeQuery(cmd.Context(), cfg, registry.MustPath(cfg.APIVersion, key), params)
		},
	}
	for flag := range flags {
		v := ""
		values[flag] = &v
		cmd.Flags().StringVar(values[flag], flag, "", "")
	}
	return cmd
}

func (s *state) getCommand() *cobra.Command {
	var paramList []string
	cmd := &cobra.Command{
		Use:   "get <endpoint>",
		Short: "Query any HAPI endpoint path",
		Example: `  hapi get metadata/sector --param name=Water
  hapi get coordination-context/conflict-events --param location_code=SDN
  hapi get food-security-nutrition-poverty/food-prices-market-monitor --param location_code=SDN
  hapi get climate/hazards-rainfall --param location_code=SDN`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := s.resolveConfig(cmd, true)
			if err != nil {
				return err
			}
			params, err := parseParams(paramList)
			if err != nil {
				return err
			}
			endpoint := registry.NormalizeEndpoint(cfg.APIVersion, args[0])
			return s.executeQuery(cmd.Context(), cfg, endpoint, params)
		},
	}
	cmd.Flags().StringArrayVar(&paramList, "param", nil, "Query parameter as key=value; repeatable")
	return cmd
}

func (s *state) listEndpointsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-endpoints",
		Short: "List known HAPI endpoint paths",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := s.resolveConfig(cmd, false)
			if err != nil {
				return err
			}
			return s.renderResult(cfg, workflows.Result{
				Endpoint: "registry/endpoints",
				Query:    url.Values{"api_version": {cfg.APIVersion}},
				Data:     endpointRows(registry.List(cfg.APIVersion)),
			}, nil)
		},
	}
	return cmd
}

func (s *state) workflowCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "workflow", Short: "Curated humanitarian workflows"}
	cmd.AddCommand(s.countryOverviewCommand())
	cmd.AddCommand(s.wash3WCommand())
	cmd.AddCommand(s.fundingCommand())
	cmd.AddCommand(s.foodSecurityCommand())
	cmd.AddCommand(s.displacementCommand())
	cmd.AddCommand(s.humanitarianNeedsCommand())
	cmd.AddCommand(s.refugeesCommand())
	cmd.AddCommand(s.populationCommand())
	cmd.AddCommand(s.conflictEventsCommand())
	return cmd
}

func (s *state) countryOverviewCommand() *cobra.Command {
	var country string
	var adminLevel int
	cmd := &cobra.Command{
		Use:   "country-overview",
		Short: "Resolve country and return HAPI data availability",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, svc, err := s.workflowService(cmd)
			if err != nil {
				return err
			}
			result, err := svc.CountryOverview(cmd.Context(), workflows.CountryInput{Country: country, AdminLevel: adminLevel})
			return s.renderResult(cfg, result, err)
		},
	}
	cmd.Flags().StringVar(&country, "country", "", "Country name")
	cmd.Flags().IntVar(&adminLevel, "admin-level", 0, "Administrative level")
	return cmd
}

func (s *state) wash3WCommand() *cobra.Command {
	var country, admin1Name string
	var adminLevel int
	cmd := &cobra.Command{
		Use:   "wash-3w",
		Short: "Resolve country, check availability, and query WASH operational presence",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, svc, err := s.workflowService(cmd)
			if err != nil {
				return err
			}
			result, err := svc.Wash3W(cmd.Context(), workflows.Wash3WInput{Country: country, Admin1Name: admin1Name, AdminLevel: adminLevel})
			return s.renderResult(cfg, result, err)
		},
	}
	cmd.Flags().StringVar(&country, "country", "", "Country name")
	cmd.Flags().StringVar(&admin1Name, "admin1-name", "", "Admin 1 name")
	cmd.Flags().IntVar(&adminLevel, "admin-level", 0, "Administrative level")
	return cmd
}

func (s *state) fundingCommand() *cobra.Command {
	var country string
	cmd := &cobra.Command{
		Use:   "funding",
		Short: "Resolve country, check availability, and query funding",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, svc, err := s.workflowService(cmd)
			if err != nil {
				return err
			}
			result, err := svc.Funding(cmd.Context(), workflows.CountryInput{Country: country})
			return s.renderResult(cfg, result, err)
		},
	}
	cmd.Flags().StringVar(&country, "country", "", "Country name")
	return cmd
}

func (s *state) foodSecurityCommand() *cobra.Command {
	var country, ipcPhase string
	var adminLevel int
	cmd := &cobra.Command{
		Use:   "food-security",
		Short: "Resolve country, check availability, and query food security",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, svc, err := s.workflowService(cmd)
			if err != nil {
				return err
			}
			result, err := svc.FoodSecurity(cmd.Context(), workflows.FoodSecurityInput{Country: country, IPCPhase: ipcPhase, AdminLevel: adminLevel})
			return s.renderResult(cfg, result, err)
		},
	}
	cmd.Flags().StringVar(&country, "country", "", "Country name")
	cmd.Flags().StringVar(&ipcPhase, "ipc-phase", "", "IPC phase filter, e.g. 3+")
	cmd.Flags().IntVar(&adminLevel, "admin-level", 0, "Administrative level")
	return cmd
}

func (s *state) displacementCommand() *cobra.Command {
	var country, typ string
	var adminLevel int
	cmd := &cobra.Command{
		Use:   "displacement",
		Short: "Resolve country, check availability, and query displacement",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, svc, err := s.workflowService(cmd)
			if err != nil {
				return err
			}
			result, err := svc.Displacement(cmd.Context(), workflows.DisplacementInput{Country: country, Type: typ, AdminLevel: adminLevel})
			return s.renderResult(cfg, result, err)
		},
	}
	cmd.Flags().StringVar(&country, "country", "", "Country name")
	cmd.Flags().StringVar(&typ, "type", "idps", "Displacement type")
	cmd.Flags().IntVar(&adminLevel, "admin-level", 0, "Administrative level")
	return cmd
}

func (s *state) humanitarianNeedsCommand() *cobra.Command {
	var country, sector, status string
	var adminLevel int
	cmd := &cobra.Command{
		Use:   "humanitarian-needs",
		Short: "Resolve country, check availability, and query humanitarian needs",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, svc, err := s.workflowService(cmd)
			if err != nil {
				return err
			}
			result, err := svc.HumanitarianNeeds(cmd.Context(), workflows.HumanitarianNeedsInput{Country: country, Sector: sector, Status: status, AdminLevel: adminLevel})
			return s.renderResult(cfg, result, err)
		},
	}
	cmd.Flags().StringVar(&country, "country", "", "Country name")
	cmd.Flags().StringVar(&sector, "sector", "", "Sector name")
	cmd.Flags().StringVar(&status, "status", "", "Population status")
	cmd.Flags().IntVar(&adminLevel, "admin-level", 0, "Administrative level")
	return cmd
}

func (s *state) refugeesCommand() *cobra.Command {
	var country string
	cmd := &cobra.Command{
		Use:   "refugees",
		Short: "Resolve asylum country, check availability, and query refugees and persons of concern",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, svc, err := s.workflowService(cmd)
			if err != nil {
				return err
			}
			result, err := svc.Refugees(cmd.Context(), workflows.CountryInput{Country: country})
			return s.renderResult(cfg, result, err)
		},
	}
	cmd.Flags().StringVar(&country, "country", "", "Country of asylum")
	return cmd
}

func (s *state) populationCommand() *cobra.Command {
	var country string
	var adminLevel int
	cmd := &cobra.Command{
		Use:   "population",
		Short: "Resolve country, check availability, and query baseline population",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, svc, err := s.workflowService(cmd)
			if err != nil {
				return err
			}
			result, err := svc.Population(cmd.Context(), workflows.CountryInput{Country: country, AdminLevel: adminLevel})
			return s.renderResult(cfg, result, err)
		},
	}
	cmd.Flags().StringVar(&country, "country", "", "Country name")
	cmd.Flags().IntVar(&adminLevel, "admin-level", 0, "Administrative level")
	return cmd
}

func (s *state) conflictEventsCommand() *cobra.Command {
	var country, eventType, startDate, endDate, admin1Name, admin2Name string
	var adminLevel int
	cmd := &cobra.Command{
		Use:   "conflict-events",
		Short: "Resolve country, check availability, and query ACLED-derived conflict events",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, svc, err := s.workflowService(cmd)
			if err != nil {
				return err
			}
			result, err := svc.ConflictEvents(cmd.Context(), workflows.ConflictEventsInput{
				Country:    country,
				EventType:  eventType,
				StartDate:  startDate,
				EndDate:    endDate,
				AdminLevel: adminLevel,
				Admin1Name: admin1Name,
				Admin2Name: admin2Name,
			})
			return s.renderResult(cfg, result, err)
		},
	}
	cmd.Flags().StringVar(&country, "country", "", "Country name")
	cmd.Flags().StringVar(&eventType, "event-type", "", "ACLED event type category")
	cmd.Flags().StringVar(&startDate, "start-date", "", "Reference period lower bound, e.g. 2026-01-01")
	cmd.Flags().StringVar(&endDate, "end-date", "", "Reference period upper bound, e.g. 2026-03-31")
	cmd.Flags().IntVar(&adminLevel, "admin-level", 0, "Administrative level")
	cmd.Flags().StringVar(&admin1Name, "admin1-name", "", "Admin 1 name")
	cmd.Flags().StringVar(&admin2Name, "admin2-name", "", "Admin 2 name")
	return cmd
}

func (s *state) workflowService(cmd *cobra.Command) (config.Config, *workflows.Service, error) {
	cfg, err := s.resolveConfig(cmd, true)
	if err != nil {
		return config.Config{}, nil, err
	}
	return cfg, workflows.New(s.hapiClient(cfg), workflows.Options{
		APIVersion: cfg.APIVersion,
		Limit:      cfg.Limit,
		Offset:     cfg.Offset,
		AllPages:   cfg.AllPages,
	}), nil
}

func (s *state) executeQuery(ctx context.Context, cfg config.Config, endpoint string, params url.Values) error {
	c := s.hapiClient(cfg)
	if cfg.Format == "csv" {
		raw, err := c.FetchCSV(ctx, endpoint, params, client.Options{Limit: cfg.Limit, Offset: cfg.Offset, AllPages: cfg.AllPages})
		if err != nil {
			return err
		}
		return s.writeBytes(cfg, raw)
	}
	resp, err := c.Fetch(ctx, endpoint, params, client.Options{Limit: cfg.Limit, Offset: cfg.Offset, AllPages: cfg.AllPages})
	result := workflows.Result{Endpoint: endpoint, Query: params}
	if err == nil {
		result.Data = resp.Data
	}
	return s.renderResult(cfg, result, err)
}

func (s *state) renderResult(cfg config.Config, result workflows.Result, err error) error {
	if err != nil {
		return err
	}
	rows := output.SelectFields(result.Data, cfg.Fields)
	switch cfg.Format {
	case "json":
		env := output.Envelope{
			OK:         true,
			Source:     "HDX/HAPI",
			APIVersion: cfg.APIVersion,
			Endpoint:   result.Endpoint,
			Query:      map[string][]string(result.Query),
			Count:      len(rows),
			Data:       rows,
			Meta:       output.Meta{Limit: cfg.Limit, Offset: cfg.Offset, AllPages: cfg.AllPages, Warnings: []string{}},
		}
		return s.write(cfg, func(w io.Writer) error { return output.WriteJSON(w, env) })
	case "jsonl":
		return s.write(cfg, func(w io.Writer) error { return output.WriteJSONL(w, rows) })
	case "csv":
		return s.write(cfg, func(w io.Writer) error { return output.WriteCSV(w, rows, cfg.Fields) })
	case "table":
		return s.write(cfg, func(w io.Writer) error { return output.WriteTable(w, rows, cfg.Fields) })
	default:
		return &config.Error{Code: "invalid_format", Message: "Use --format json, jsonl, csv, or table."}
	}
}

func (s *state) writeBytes(cfg config.Config, data []byte) error {
	return s.write(cfg, func(w io.Writer) error {
		_, err := w.Write(data)
		return err
	})
}

func (s *state) write(cfg config.Config, fn func(io.Writer) error) error {
	if cfg.Output == "" {
		return fn(s.opts.Stdout)
	}
	f, err := os.Create(cfg.Output)
	if err != nil {
		return err
	}
	defer f.Close()
	return fn(f)
}

func (s *state) resolveConfig(cmd *cobra.Command, requireApp bool) (config.Config, error) {
	flags := config.FlagValues{}
	p := cmd.Root().PersistentFlags()
	if p.Changed("base-url") {
		flags.BaseURL, _ = p.GetString("base-url")
	}
	if p.Changed("api-version") {
		flags.APIVersion, _ = p.GetString("api-version")
	}
	if p.Changed("app-identifier") {
		flags.AppIdentifier, _ = p.GetString("app-identifier")
	}
	if p.Changed("format") {
		flags.Format, _ = p.GetString("format")
	}
	if p.Changed("limit") {
		flags.Limit, _ = p.GetInt("limit")
	}
	if p.Changed("offset") {
		flags.Offset, _ = p.GetInt("offset")
	}
	if p.Changed("all-pages") {
		flags.AllPages, _ = p.GetBool("all-pages")
	}
	if p.Changed("output") {
		flags.Output, _ = p.GetString("output")
	}
	if p.Changed("quiet") {
		flags.Quiet, _ = p.GetBool("quiet")
	}
	if p.Changed("debug") {
		flags.Debug, _ = p.GetBool("debug")
	}
	if p.Changed("timeout") {
		flags.TimeoutSeconds, _ = p.GetFloat64("timeout")
	}
	if s.fieldsCSV != "" {
		flags.Fields = splitCSV(s.fieldsCSV)
	}
	cfg, err := config.Resolve(flags, s.opts.Env, s.configPath)
	if err != nil {
		return config.Config{}, err
	}
	if !requireApp {
		return cfg, nil
	}
	return cfg, cfg.Validate()
}

func (s *state) hapiClient(cfg config.Config) *client.Client {
	return client.New(client.Config{
		BaseURL:       cfg.BaseURL,
		APIVersion:    cfg.APIVersion,
		AppIdentifier: cfg.AppIdentifier,
		HTTPClient:    s.opts.HTTPClient,
		Timeout:       time.Duration(cfg.TimeoutSeconds * float64(time.Second)),
	})
}

func endpointRows(endpoints []registry.Endpoint) []map[string]any {
	rows := make([]map[string]any, 0, len(endpoints))
	for _, ep := range endpoints {
		rows = append(rows, map[string]any{
			"key":         ep.Key,
			"path":        ep.Path,
			"description": ep.Description,
		})
	}
	return rows
}

func parseParams(items []string) (url.Values, error) {
	params := url.Values{}
	for _, item := range items {
		key, value, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, &config.Error{Code: "invalid_param", Message: "--param must use key=value."}
		}
		params.Add(strings.TrimSpace(key), value)
	}
	return params, nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func ExitCode(err error) int {
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return 1
}

func ErrorEnvelope(err error) output.ErrorEnvelope {
	code := "error"
	retryable := false
	var clientErr *client.Error
	if errors.As(err, &clientErr) {
		code = clientErr.Code
		retryable = clientErr.Retryable
	}
	var cfgErr *config.Error
	if errors.As(err, &cfgErr) {
		code = cfgErr.Code
	}
	return output.ErrorEnvelope{
		OK: false,
		Error: output.ErrorBlock{
			Code:      code,
			Message:   fmt.Sprint(err),
			Retryable: retryable,
		},
	}
}
