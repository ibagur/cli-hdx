package registry

import "testing"

func TestV2EndpointPathsMatchVerifiedOpenAPI(t *testing.T) {
	tests := map[string]string{
		"metadata.locations":          "metadata/location",
		"metadata.availability":       "metadata/data-availability",
		"metadata.sectors":            "metadata/sector",
		"metadata.admin1":             "metadata/admin1",
		"metadata.admin2":             "metadata/admin2",
		"operational_presence":        "coordination-context/operational-presence",
		"funding":                     "coordination-context/funding",
		"food_security":               "food-security-nutrition-poverty/food-security",
		"displacement.idps":           "affected-people/idps",
		"humanitarian_needs":          "affected-people/humanitarian-needs",
		"baseline_population":         "geography-infrastructure/baseline-population",
		"refugees_persons_of_concern": "affected-people/refugees-persons-of-concern",
		"returnees":                   "affected-people/returnees",
		"conflict_events":             "coordination-context/conflict-events",
		"national_risk":               "coordination-context/national-risk",
		"food_prices_market_monitor":  "food-security-nutrition-poverty/food-prices-market-monitor",
		"poverty_rate":                "food-security-nutrition-poverty/poverty-rate",
		"hazards_rainfall":            "climate/hazards-rainfall",
	}

	for key, want := range tests {
		got, ok := Lookup("v2", key)
		if !ok {
			t.Fatalf("Lookup(%q) missing", key)
		}
		if got.Path != want {
			t.Fatalf("Lookup(%q).Path = %q, want %q", key, got.Path, want)
		}
	}
}

func TestListReturnsEndpointsSortedByKey(t *testing.T) {
	got := List("v2")
	if len(got) == 0 {
		t.Fatal("List returned no endpoints")
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].Key > got[i].Key {
			t.Fatalf("List not sorted at %d: %q > %q", i, got[i-1].Key, got[i].Key)
		}
	}
	if got[0].Key == "" || got[0].Path == "" || got[0].Description == "" {
		t.Fatalf("endpoint missing list fields: %#v", got[0])
	}
}

func TestNormalizeRawEndpoint(t *testing.T) {
	got := NormalizeEndpoint("v2", "/api/v2/metadata/sector")
	if got != "metadata/sector" {
		t.Fatalf("NormalizeEndpoint returned %q", got)
	}
	got = NormalizeEndpoint("v2", "metadata/sector")
	if got != "metadata/sector" {
		t.Fatalf("NormalizeEndpoint returned %q", got)
	}
}
