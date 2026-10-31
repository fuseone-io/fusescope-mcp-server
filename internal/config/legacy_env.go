package config

import (
	"log"
	"os"
)

// legacyEnvAliases maps each renamed variable to its former SIGNOZ_* name,
// still read when the new one is unset. Remove once deployments have moved.
var legacyEnvAliases = []struct{ current, legacy string }{
	{FuseScopeURL, "SIGNOZ_URL"},
	{FuseScopeAPIKey, "SIGNOZ_API_KEY"},
	{FuseScopeCustomHeaders, "SIGNOZ_CUSTOM_HEADERS"},
	{InstanceURLAllowlistEnv, "SIGNOZ_INSTANCE_URL_ALLOWLIST"},
	{DocsRefreshIntervalEnv, "SIGNOZ_DOCS_REFRESH_INTERVAL"},
	{DocsFullRefreshIntervalEnv, "SIGNOZ_DOCS_FULL_REFRESH_INTERVAL"},
}

// lookupEnv reads key, falling back to its legacy alias when key is unset.
func lookupEnv(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	for _, alias := range legacyEnvAliases {
		if alias.current == key {
			return os.Getenv(alias.legacy)
		}
	}
	return ""
}

func warnLegacyEnv() {
	for _, alias := range legacyEnvAliases {
		if os.Getenv(alias.legacy) == "" {
			continue
		}
		if os.Getenv(alias.current) != "" {
			log.Printf("WARN: %s is deprecated and ignored because %s is set; remove it", alias.legacy, alias.current)
			continue
		}
		log.Printf("WARN: %s is deprecated; rename it to %s", alias.legacy, alias.current)
	}
}
