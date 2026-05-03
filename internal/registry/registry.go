package registry

import "strings"

type Endpoint struct {
	Key         string
	Path        string
	Description string
}

var endpoints = map[string]map[string]Endpoint{
	"v2": {
		"metadata.locations":          {Key: "metadata.locations", Path: "metadata/location", Description: "Country and country-like location metadata"},
		"metadata.availability":       {Key: "metadata.availability", Path: "metadata/data-availability", Description: "HAPI data availability by location and subcategory"},
		"metadata.sectors":            {Key: "metadata.sectors", Path: "metadata/sector", Description: "Sector support table"},
		"metadata.admin1":             {Key: "metadata.admin1", Path: "metadata/admin1", Description: "Admin 1 metadata"},
		"metadata.admin2":             {Key: "metadata.admin2", Path: "metadata/admin2", Description: "Admin 2 metadata"},
		"operational_presence":        {Key: "operational_presence", Path: "coordination-context/operational-presence", Description: "3W operational presence"},
		"funding":                     {Key: "funding", Path: "coordination-context/funding", Description: "FTS requirements and funding"},
		"food_security":               {Key: "food_security", Path: "food-security-nutrition-poverty/food-security", Description: "IPC food security"},
		"displacement.idps":           {Key: "displacement.idps", Path: "affected-people/idps", Description: "IDP displacement figures"},
		"humanitarian_needs":          {Key: "humanitarian_needs", Path: "affected-people/humanitarian-needs", Description: "Humanitarian needs"},
		"baseline_population":         {Key: "baseline_population", Path: "geography-infrastructure/baseline-population", Description: "Baseline population"},
		"refugees_persons_of_concern": {Key: "refugees_persons_of_concern", Path: "affected-people/refugees-persons-of-concern", Description: "UNHCR refugees and persons of concern"},
	},
	"v1": {
		"metadata.locations":          {Key: "metadata.locations", Path: "metadata/location", Description: "Country and country-like location metadata"},
		"metadata.availability":       {Key: "metadata.availability", Path: "metadata/data-availability", Description: "HAPI data availability by location and subcategory"},
		"metadata.sectors":            {Key: "metadata.sectors", Path: "metadata/sector", Description: "Sector support table"},
		"metadata.admin1":             {Key: "metadata.admin1", Path: "metadata/admin1", Description: "Admin 1 metadata"},
		"metadata.admin2":             {Key: "metadata.admin2", Path: "metadata/admin2", Description: "Admin 2 metadata"},
		"operational_presence":        {Key: "operational_presence", Path: "coordination-context/operational-presence", Description: "3W operational presence"},
		"funding":                     {Key: "funding", Path: "coordination-context/funding", Description: "FTS requirements and funding"},
		"food_security":               {Key: "food_security", Path: "food/food-security", Description: "Legacy v1 food security"},
		"displacement.idps":           {Key: "displacement.idps", Path: "affected-people/idps", Description: "IDP displacement figures"},
		"humanitarian_needs":          {Key: "humanitarian_needs", Path: "affected-people/humanitarian-needs", Description: "Humanitarian needs"},
		"baseline_population":         {Key: "baseline_population", Path: "population-social/population", Description: "Legacy v1 population"},
		"refugees_persons_of_concern": {Key: "refugees_persons_of_concern", Path: "affected-people/refugees", Description: "Legacy v1 refugee data"},
	},
}

func Lookup(version, key string) (Endpoint, bool) {
	version = strings.ToLower(strings.TrimSpace(version))
	byVersion, ok := endpoints[version]
	if !ok {
		return Endpoint{}, false
	}
	ep, ok := byVersion[key]
	return ep, ok
}

func MustPath(version, key string) string {
	ep, ok := Lookup(version, key)
	if !ok {
		return key
	}
	return ep.Path
}

func NormalizeEndpoint(version, endpoint string) string {
	e := strings.TrimSpace(endpoint)
	e = strings.TrimPrefix(e, "/")
	prefix := "api/" + strings.TrimPrefix(version, "/") + "/"
	e = strings.TrimPrefix(e, prefix)
	return strings.TrimPrefix(e, "/")
}
