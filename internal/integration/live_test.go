package integration

import (
	"context"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/ibagur/cli-hdx/internal/client"
)

func TestLiveMetadataLocationsSmoke(t *testing.T) {
	appIdentifier := os.Getenv("HAPI_APP_IDENTIFIER")
	if appIdentifier == "" {
		t.Skip("HAPI_APP_IDENTIFIER is not set; skipping live HAPI integration smoke test")
	}
	c := client.New(client.Config{
		BaseURL:       "https://hapi.humdata.org/api",
		APIVersion:    "v2",
		AppIdentifier: appIdentifier,
		Timeout:       30 * time.Second,
	})
	resp, err := c.Fetch(context.Background(), "metadata/location", url.Values{"name": {"Sudan"}}, client.Options{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) == 0 {
		t.Fatal("expected at least one Sudan location record")
	}
}
