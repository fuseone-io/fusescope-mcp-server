package docs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanonicalDocURL(t *testing.T) {
	cases := []struct {
		raw  string
		want string
		ok   bool
	}{
		{"https://docs.fusescope.fuseone.io/install/helm/", "https://docs.fusescope.fuseone.io/install/helm/", true},
		{"https://docs.fusescope.fuseone.io/install/helm#values", "https://docs.fusescope.fuseone.io/install/helm/", true},
		{"/install/helm/", "https://docs.fusescope.fuseone.io/install/helm/", true},
		{"https://docs.fusescope.fuseone.io/sitemap.md", "https://docs.fusescope.fuseone.io/sitemap.md", true},
		// Links to the former GitHub Pages address resolve to the current site.
		{"https://fuseone-io.github.io/fusescope-docs/install/helm/", "https://docs.fusescope.fuseone.io/install/helm/", true},
		{"/fusescope-docs/install/helm/", "https://docs.fusescope.fuseone.io/install/helm/", true},
		{"https://fuseone-io.github.io/fusescope-charts/", "", false},
		{"http://docs.fusescope.fuseone.io/install/helm/", "", false},
		{"https://example.com/install/helm/", "", false},
		{"install/helm/", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := CanonicalDocURL(tc.raw)
		require.Equal(t, tc.ok, ok, tc.raw)
		require.Equal(t, tc.want, got, tc.raw)
	}
}
