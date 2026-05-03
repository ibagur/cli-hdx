package workflows

import (
	"context"
	"net/url"
	"testing"

	"github.com/ibagur/cli-hdx/internal/client"
)

type call struct {
	endpoint string
	params   url.Values
}

type fakeQueryer struct {
	calls     []call
	responses map[string][]map[string]any
}

func (f *fakeQueryer) Fetch(ctx context.Context, endpoint string, params url.Values, opts client.Options) (client.Response, error) {
	f.calls = append(f.calls, call{endpoint: endpoint, params: params})
	data := f.responses[endpoint]
	return client.Response{Data: data}, nil
}

func TestWash3WResolvesLocationAvailabilitySectorThenQueriesOperationalPresence(t *testing.T) {
	q := &fakeQueryer{responses: map[string][]map[string]any{
		"metadata/location":                         {{"code": "NGA", "name": "Nigeria"}},
		"metadata/data-availability":                {{"subcategory": "Operational Presence", "location_code": "NGA"}},
		"metadata/sector":                           {{"code": "WSH", "name": "Water Sanitation Hygiene"}},
		"coordination-context/operational-presence": {{"org_name": "Example Org", "sector_code": "WSH"}},
	}}
	svc := New(q, Options{APIVersion: "v2", Limit: 1000})

	got, err := svc.Wash3W(context.Background(), Wash3WInput{Country: "Nigeria", Admin1Name: "Yobe"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Data) != 1 || got.Data[0]["org_name"] != "Example Org" {
		t.Fatalf("data = %#v", got.Data)
	}
	wantEndpoints := []string{
		"metadata/location",
		"metadata/data-availability",
		"metadata/sector",
		"coordination-context/operational-presence",
	}
	for i, want := range wantEndpoints {
		if q.calls[i].endpoint != want {
			t.Fatalf("call %d endpoint = %q, want %q", i, q.calls[i].endpoint, want)
		}
	}
	query := q.calls[3].params
	if query.Get("location_code") != "NGA" || query.Get("admin1_name") != "Yobe" || query.Get("sector_code") != "WSH" {
		t.Fatalf("operational presence query = %s", query.Encode())
	}
}

func TestFundingPreservesReturnedRecords(t *testing.T) {
	q := &fakeQueryer{responses: map[string][]map[string]any{
		"metadata/location":          {{"code": "SSD", "name": "South Sudan"}},
		"metadata/data-availability": {{"subcategory": "Funding", "location_code": "SSD"}},
		"coordination-context/funding": {
			{"requirements_usd": 100.0, "funding_usd": 25.0, "funding_pct": 25.0},
		},
	}}
	svc := New(q, Options{APIVersion: "v2", Limit: 1000})

	got, err := svc.Funding(context.Background(), CountryInput{Country: "South Sudan"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Data[0]["funding_pct"] != 25.0 {
		t.Fatalf("funding record was changed: %#v", got.Data[0])
	}
	if q.calls[2].endpoint != "coordination-context/funding" {
		t.Fatalf("endpoint = %q", q.calls[2].endpoint)
	}
}

func TestFoodSecuritySupportsIPCPhaseAndAdminLevel(t *testing.T) {
	q := &fakeQueryer{responses: map[string][]map[string]any{
		"metadata/location":                             {{"code": "MOZ", "name": "Mozambique"}},
		"metadata/data-availability":                    {{"subcategory": "Food Security", "location_code": "MOZ"}},
		"food-security-nutrition-poverty/food-security": {{"ipc_phase": "3+"}},
	}}
	svc := New(q, Options{APIVersion: "v2", Limit: 1000})

	_, err := svc.FoodSecurity(context.Background(), FoodSecurityInput{Country: "Mozambique", IPCPhase: "3+", AdminLevel: 1})
	if err != nil {
		t.Fatal(err)
	}
	query := q.calls[2].params
	if query.Get("ipc_phase") != "3+" || query.Get("admin_level") != "1" || query.Get("location_code") != "MOZ" {
		t.Fatalf("food security query = %s", query.Encode())
	}
}
