package config

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_LegacySigNozEnvAliases(t *testing.T) {
	vars := []struct {
		newKey, oldKey string
		newValue       string
		oldValue       string
		read           func(*Config) string
	}{
		{"FUSESCOPE_URL", "SIGNOZ_URL", "https://new.example.com", "https://old.example.com",
			func(c *Config) string { return c.URL }},
		{"FUSESCOPE_API_KEY", "SIGNOZ_API_KEY", "new-key", "old-key",
			func(c *Config) string { return c.APIKey }},
		{"FUSESCOPE_CUSTOM_HEADERS", "SIGNOZ_CUSTOM_HEADERS", "X-Which:new", "X-Which:old",
			func(c *Config) string {
				if value, ok := c.CustomHeaders["X-Which"]; ok {
					return "X-Which:" + value
				}
				return ""
			}},
		{"FUSESCOPE_INSTANCE_URL_ALLOWLIST", "SIGNOZ_INSTANCE_URL_ALLOWLIST", "new.example.com", "old.example.com",
			func(c *Config) string {
				switch {
				case !c.InstanceURLAllowlist.Configured():
					return ""
				case c.InstanceURLAllowlist.AllowsHost("new.example.com"):
					return "new.example.com"
				case c.InstanceURLAllowlist.AllowsHost("old.example.com"):
					return "old.example.com"
				}
				return "unexpected"
			}},
		{"FUSESCOPE_DOCS_REFRESH_INTERVAL", "SIGNOZ_DOCS_REFRESH_INTERVAL", "2h", "3h",
			func(c *Config) string { return c.DocsRefreshInterval.String() }},
		{"FUSESCOPE_DOCS_FULL_REFRESH_INTERVAL", "SIGNOZ_DOCS_FULL_REFRESH_INTERVAL", "48h", "72h",
			func(c *Config) string { return c.DocsFullRefreshInterval.String() }},
	}

	cases := []struct {
		name         string
		setNew       bool
		setOld       bool
		wantNew      bool
		wantOld      bool
		wantWarnings int
	}{
		{name: "only new", setNew: true, wantNew: true},
		{name: "only old", setOld: true, wantOld: true, wantWarnings: 1},
		{name: "both set, new wins", setNew: true, setOld: true, wantNew: true, wantWarnings: 1},
		{name: "neither set"},
	}

	defaults := map[string]string{
		"FUSESCOPE_DOCS_REFRESH_INTERVAL":      defaultDocsRefreshInterval.String(),
		"FUSESCOPE_DOCS_FULL_REFRESH_INTERVAL": defaultDocsFullRefreshPeriod.String(),
	}

	for _, v := range vars {
		for _, tc := range cases {
			t.Run(v.newKey+"/"+tc.name, func(t *testing.T) {
				for _, other := range vars {
					t.Setenv(other.newKey, "")
					t.Setenv(other.oldKey, "")
				}
				if tc.setNew {
					t.Setenv(v.newKey, v.newValue)
				}
				if tc.setOld {
					t.Setenv(v.oldKey, v.oldValue)
				}

				var logs bytes.Buffer
				log.SetOutput(&logs)
				t.Cleanup(func() { log.SetOutput(os.Stderr) })

				cfg, err := LoadConfig()
				require.NoError(t, err)

				want := defaults[v.newKey]
				if tc.wantNew {
					want = v.newValue
				}
				if tc.wantOld {
					want = v.oldValue
				}
				if d, err := time.ParseDuration(want); err == nil {
					want = d.String()
				}
				assert.Equal(t, want, v.read(cfg))
				assert.Equal(t, tc.wantWarnings, strings.Count(logs.String(), v.oldKey+" is deprecated"), logs.String())
			})
		}
	}
}
