package config

import (
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

// TestRestServerIgnoresBareEnvVars pins the ignored:"true" tags: envconfig runs with an empty
// prefix, so an untagged field would bind from its bare uppercased name. VERSION is commonly
// exported in container images, and the subpaths are route-bearing.
func TestRestServerIgnoresBareEnvVars(t *testing.T) {
	for _, name := range []string{"TITLE", "FSPTITLE", "FSPSUBPATH", "DATITLE", "DAPSUBPATH", "VERSION"} {
		t.Setenv(name, "clobbered")
	}

	cfg := RestServer{
		Title:      "title",
		FSPTitle:   "fspTitle",
		FSPSubpath: "/fsp",
		DATitle:    "daTitle",
		DAPSubpath: "/da",
		Version:    "1.0.0",
	}

	require.NoError(t, envconfig.Process("", &cfg))

	require.Equal(t, "title", cfg.Title)
	require.Equal(t, "fspTitle", cfg.FSPTitle)
	require.Equal(t, "/fsp", cfg.FSPSubpath)
	require.Equal(t, "daTitle", cfg.DATitle)
	require.Equal(t, "/da", cfg.DAPSubpath)
	require.Equal(t, "1.0.0", cfg.Version)
}

// TestRestServerReadsTaggedEnvVars guards the other direction — the tags above must not
// silence the REST_* overrides that are meant to work.
func TestRestServerReadsTaggedEnvVars(t *testing.T) {
	t.Setenv("REST_ADDR", ":9999")
	t.Setenv("REST_API_KEY_NAME", "X-Key")
	t.Setenv("REST_API_KEYS", "a,b")
	t.Setenv("REST_CORS_ORIGIN", "https://example.com")

	var cfg RestServer
	require.NoError(t, envconfig.Process("", &cfg))

	require.Equal(t, ":9999", cfg.Addr)
	require.Equal(t, "X-Key", cfg.APIKeyName)
	require.Equal(t, []string{"a", "b"}, cfg.APIKeys)
	require.Equal(t, "https://example.com", cfg.CORSOrigin)
}
