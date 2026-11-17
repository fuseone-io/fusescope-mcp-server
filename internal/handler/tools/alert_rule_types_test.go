package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fuseone-io/fusescope-mcp-server/internal/config"
	mcp "github.com/fuseone-io/fusescope-mcp-server/internal/mcpcontract"
	logpkg "github.com/fuseone-io/fusescope-mcp-server/pkg/log"
	"github.com/fuseone-io/fusescope-mcp-server/pkg/util"
)

// FuseScope evaluates anomaly_rule on schemaVersion v1 only, so the alert write
// tools offer it with its condition fields.
func TestAlertWriteSchemasOfferV1AnomalyRules(t *testing.T) {
	h := &Handler{logger: logpkg.New("error")}
	s := newMCPTestServer()
	h.RegisterAlertsHandlers(s)
	tools := listTestTools(t, s)

	for _, name := range []string{"fusescope_create_alert", "fusescope_update_alert"} {
		t.Run(name, func(t *testing.T) {
			tool, ok := tools[name]
			require.True(t, ok, "%s is not registered", name)
			raw, err := json.Marshal(tool.Tool.InputSchema)
			require.NoError(t, err)
			var schema struct {
				Properties map[string]struct {
					Description string         `json:"description"`
					Enum        []string       `json:"enum"`
					Properties  map[string]any `json:"properties"`
				} `json:"properties"`
			}
			require.NoError(t, json.Unmarshal(raw, &schema))

			ruleType := schema.Properties["ruleType"]
			require.Equal(t, []string{"threshold_rule", "promql_rule", "anomaly_rule"}, ruleType.Enum)
			require.Contains(t, ruleType.Description, "anomaly_rule", "ruleType must teach when to pick anomaly_rule")
			require.Contains(t, ruleType.Description, "schemaVersion v1")
			condition := schema.Properties["condition"].Properties
			for _, field := range []string{"algorithm", "seasonality"} {
				require.Contains(t, condition, field, "condition must expose the anomaly field %q", field)
			}
		})
	}
}

// A v1 anomaly rule carries no evaluation, notificationSettings or thresholds:
// FuseScope derives them from the v1 fields, so the advertised schema must not
// require them and the payload reaches FuseScope as sent.
func TestCreateAlert_SendsV1AnomalyRuleAsSentWithoutSchemaNotice(t *testing.T) {
	const rule = `{
		"alert": "checkout traffic anomaly",
		"alertType": "METRIC_BASED_ALERT",
		"ruleType": "anomaly_rule",
		"schemaVersion": "v1",
		"evalWindow": "10m",
		"frequency": "1m",
		"condition": {
			"compositeQuery": {"queryType": "builder", "panelType": "graph", "queries": [{"type": "builder_query", "spec": {
				"name": "A", "signal": "metrics", "stepInterval": 60,
				"aggregations": [{"metricName": "http_server_requests", "timeAggregation": "rate", "spaceAggregation": "sum"}],
				"functions": [{"name": "anomaly", "args": [{"name": "z_score_threshold", "value": 3}]}]
			}}]},
			"selectedQueryName": "A",
			"target": 3,
			"op": "above",
			"matchType": "at_least_once",
			"algorithm": "standard",
			"seasonality": "daily"
		},
		"labels": {"severity": "warning"},
		"preferredChannels": ["On call"]
	}`

	var sent []byte
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/rules" {
			t.Errorf("unexpected API request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		sent, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"id":"019a0000-0000-7000-8000-000000000001"}}`))
	}))
	t.Cleanup(backend.Close)
	t.Setenv(config.FuseScopeURL, strings.Replace(backend.URL, "127.0.0.1", "localhost", 1))
	t.Setenv(config.FuseScopeAPIKey, "test-key")
	cfg, err := config.LoadConfig()
	require.NoError(t, err)
	h := NewHandler(logpkg.New("error"), cfg)
	s := newMCPTestServer()
	h.RegisterAlertsHandlers(s)
	ctx := util.SetAPIKey(util.SetSigNozURL(context.Background(), cfg.URL), cfg.APIKey)

	var args map[string]any
	require.NoError(t, json.Unmarshal([]byte(rule), &args))
	args["searchContext"] = "alert me when checkout traffic is anomalous"
	message, err := json.Marshal(map[string]any{"params": map[string]any{"name": "fusescope_create_alert", "arguments": args}})
	require.NoError(t, err)
	result := callTestToolFromJSONRPC(ctx, t, s, message)

	require.False(t, result.IsError, "%v", result.Content)
	require.JSONEq(t, rule, string(sent), "the rule must reach FuseScope as sent, without searchContext")
	for _, content := range result.Content {
		if text, ok := mcp.AsTextContent(content); ok {
			require.NotContains(t, text.Text, inputValidationNoticePrefix, "a valid v1 anomaly rule must match the advertised schema")
		}
	}
}
