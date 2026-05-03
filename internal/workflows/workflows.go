package workflows

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/ibagur/cli-hdx/internal/client"
	"github.com/ibagur/cli-hdx/internal/registry"
)

type Queryer interface {
	Fetch(ctx context.Context, endpoint string, params url.Values, opts client.Options) (client.Response, error)
}

type Options struct {
	APIVersion string
	Limit      int
	Offset     int
	AllPages   bool
}

type Service struct {
	q    Queryer
	opts Options
}

type Result struct {
	Endpoint string
	Query    url.Values
	Data     []map[string]any
}

type CountryInput struct {
	Country    string
	AdminLevel int
}

type Wash3WInput struct {
	Country    string
	Admin1Name string
	AdminLevel int
}

type FoodSecurityInput struct {
	Country    string
	IPCPhase   string
	AdminLevel int
}

type DisplacementInput struct {
	Country    string
	Type       string
	AdminLevel int
}

type HumanitarianNeedsInput struct {
	Country    string
	Sector     string
	Status     string
	AdminLevel int
}

type ConflictEventsInput struct {
	Country    string
	EventType  string
	StartDate  string
	EndDate    string
	AdminLevel int
	Admin1Name string
	Admin2Name string
}

func New(q Queryer, opts Options) *Service {
	if opts.APIVersion == "" {
		opts.APIVersion = "v2"
	}
	if opts.Limit == 0 {
		opts.Limit = 1000
	}
	return &Service{q: q, opts: opts}
}

func (s *Service) CountryOverview(ctx context.Context, in CountryInput) (Result, error) {
	location, err := s.resolveLocation(ctx, in.Country)
	if err != nil {
		return Result{}, err
	}
	query := url.Values{"location_code": {location.Code}}
	if in.AdminLevel > 0 {
		query.Set("admin_level", strconv.Itoa(in.AdminLevel))
	}
	return s.fetch(ctx, "metadata.availability", query)
}

func (s *Service) Wash3W(ctx context.Context, in Wash3WInput) (Result, error) {
	location, err := s.resolveLocation(ctx, in.Country)
	if err != nil {
		return Result{}, err
	}
	if _, err := s.checkAvailability(ctx, location.Code, "Operational Presence"); err != nil {
		return Result{}, err
	}
	sector, err := s.resolveSector(ctx, "Water Sanitation Hygiene")
	if err != nil {
		return Result{}, err
	}
	query := url.Values{
		"location_code": {location.Code},
		"sector_code":   {sector.Code},
	}
	if in.Admin1Name != "" {
		query.Set("admin1_name", in.Admin1Name)
	}
	if in.AdminLevel > 0 {
		query.Set("admin_level", strconv.Itoa(in.AdminLevel))
	}
	return s.fetch(ctx, "operational_presence", query)
}

func (s *Service) Funding(ctx context.Context, in CountryInput) (Result, error) {
	location, err := s.resolveLocation(ctx, in.Country)
	if err != nil {
		return Result{}, err
	}
	if _, err := s.checkAvailability(ctx, location.Code, "Funding"); err != nil {
		return Result{}, err
	}
	return s.fetch(ctx, "funding", url.Values{"location_code": {location.Code}})
}

func (s *Service) FoodSecurity(ctx context.Context, in FoodSecurityInput) (Result, error) {
	location, err := s.resolveLocation(ctx, in.Country)
	if err != nil {
		return Result{}, err
	}
	if _, err := s.checkAvailability(ctx, location.Code, "Food Security"); err != nil {
		return Result{}, err
	}
	query := url.Values{"location_code": {location.Code}}
	if in.IPCPhase != "" {
		query.Set("ipc_phase", in.IPCPhase)
	}
	if in.AdminLevel > 0 {
		query.Set("admin_level", strconv.Itoa(in.AdminLevel))
	}
	return s.fetch(ctx, "food_security", query)
}

func (s *Service) Displacement(ctx context.Context, in DisplacementInput) (Result, error) {
	location, err := s.resolveLocation(ctx, in.Country)
	if err != nil {
		return Result{}, err
	}
	displacementType := strings.ToLower(strings.TrimSpace(in.Type))
	if displacementType == "" || displacementType == "idps" || displacementType == "idp" {
		if _, err := s.checkAvailability(ctx, location.Code, "IDPs"); err != nil {
			return Result{}, err
		}
		query := url.Values{"location_code": {location.Code}}
		if in.AdminLevel > 0 {
			query.Set("admin_level", strconv.Itoa(in.AdminLevel))
		}
		return s.fetch(ctx, "displacement.idps", query)
	}
	return Result{}, &client.Error{Code: "unsupported_workflow_type", Message: "Only --type idps is supported in v1.", Retryable: false}
}

func (s *Service) HumanitarianNeeds(ctx context.Context, in HumanitarianNeedsInput) (Result, error) {
	location, err := s.resolveLocation(ctx, in.Country)
	if err != nil {
		return Result{}, err
	}
	if _, err := s.checkAvailability(ctx, location.Code, "Humanitarian Needs"); err != nil {
		return Result{}, err
	}
	query := url.Values{"location_code": {location.Code}}
	if in.Sector != "" {
		query.Set("sector_name", in.Sector)
	}
	if in.Status != "" {
		query.Set("population_status", in.Status)
	}
	if in.AdminLevel > 0 {
		query.Set("admin_level", strconv.Itoa(in.AdminLevel))
	}
	return s.fetch(ctx, "humanitarian_needs", query)
}

func (s *Service) Refugees(ctx context.Context, in CountryInput) (Result, error) {
	location, err := s.resolveLocation(ctx, in.Country)
	if err != nil {
		return Result{}, err
	}
	if _, err := s.checkAvailability(ctx, location.Code, "Refugees & Persons of Concern"); err != nil {
		return Result{}, err
	}
	return s.fetch(ctx, "refugees_persons_of_concern", url.Values{"asylum_location_code": {location.Code}})
}

func (s *Service) Population(ctx context.Context, in CountryInput) (Result, error) {
	location, err := s.resolveLocation(ctx, in.Country)
	if err != nil {
		return Result{}, err
	}
	if _, err := s.checkAvailability(ctx, location.Code, "Baseline Population"); err != nil {
		return Result{}, err
	}
	query := url.Values{"location_code": {location.Code}}
	if in.AdminLevel > 0 {
		query.Set("admin_level", strconv.Itoa(in.AdminLevel))
	}
	return s.fetch(ctx, "baseline_population", query)
}

func (s *Service) ConflictEvents(ctx context.Context, in ConflictEventsInput) (Result, error) {
	location, err := s.resolveLocation(ctx, in.Country)
	if err != nil {
		return Result{}, err
	}
	if _, err := s.checkAvailability(ctx, location.Code, "conflict-events"); err != nil {
		return Result{}, err
	}
	query := url.Values{"location_code": {location.Code}}
	if in.EventType != "" {
		query.Set("event_type", in.EventType)
	}
	if in.StartDate != "" {
		query.Set("start_date", in.StartDate)
	}
	if in.EndDate != "" {
		query.Set("end_date", in.EndDate)
	}
	if in.AdminLevel > 0 {
		query.Set("admin_level", strconv.Itoa(in.AdminLevel))
	}
	if in.Admin1Name != "" {
		query.Set("admin1_name", in.Admin1Name)
	}
	if in.Admin2Name != "" {
		query.Set("admin2_name", in.Admin2Name)
	}
	return s.fetch(ctx, "conflict_events", query)
}

type location struct {
	Code string
	Name string
}

type sector struct {
	Code string
	Name string
}

func (s *Service) resolveLocation(ctx context.Context, country string) (location, error) {
	if strings.TrimSpace(country) == "" {
		return location{}, &client.Error{Code: "missing_country", Message: "Pass --country.", Retryable: false}
	}
	resp, err := s.q.Fetch(ctx, registry.MustPath(s.opts.APIVersion, "metadata.locations"), url.Values{"name": {country}}, s.clientOptions())
	if err != nil {
		return location{}, err
	}
	for _, row := range resp.Data {
		name := fmt.Sprint(row["name"])
		if strings.EqualFold(name, country) {
			return location{Code: fmt.Sprint(row["code"]), Name: name}, nil
		}
	}
	if len(resp.Data) == 0 {
		return location{}, &client.Error{Code: "location_not_found", Message: "No HAPI location matched the country.", Retryable: false}
	}
	return location{Code: fmt.Sprint(resp.Data[0]["code"]), Name: fmt.Sprint(resp.Data[0]["name"])}, nil
}

func (s *Service) resolveSector(ctx context.Context, name string) (sector, error) {
	resp, err := s.q.Fetch(ctx, registry.MustPath(s.opts.APIVersion, "metadata.sectors"), url.Values{"name": {name}}, s.clientOptions())
	if err != nil {
		return sector{}, err
	}
	for _, row := range resp.Data {
		rowName := fmt.Sprint(row["name"])
		if strings.EqualFold(rowName, name) {
			return sector{Code: fmt.Sprint(row["code"]), Name: rowName}, nil
		}
	}
	if len(resp.Data) == 0 {
		return sector{}, &client.Error{Code: "sector_not_found", Message: "No HAPI sector matched Water Sanitation Hygiene.", Retryable: false}
	}
	return sector{Code: fmt.Sprint(resp.Data[0]["code"]), Name: fmt.Sprint(resp.Data[0]["name"])}, nil
}

func (s *Service) checkAvailability(ctx context.Context, locationCode, subcategory string) (Result, error) {
	query := url.Values{"location_code": {locationCode}}
	if subcategory != "" {
		query.Set("subcategory", subcategory)
	}
	return s.fetch(ctx, "metadata.availability", query)
}

func (s *Service) fetch(ctx context.Context, key string, query url.Values) (Result, error) {
	endpoint := registry.MustPath(s.opts.APIVersion, key)
	resp, err := s.q.Fetch(ctx, endpoint, query, s.clientOptions())
	if err != nil {
		return Result{}, err
	}
	return Result{Endpoint: endpoint, Query: query, Data: resp.Data}, nil
}

func (s *Service) clientOptions() client.Options {
	return client.Options{
		Limit:    s.opts.Limit,
		Offset:   s.opts.Offset,
		AllPages: s.opts.AllPages,
	}
}
