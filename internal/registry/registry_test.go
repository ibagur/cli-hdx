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
