package tools

import (
	"encoding/json"
	"strings"
	"testing"

	logpkg "github.com/fuseone-io/fusescope-mcp-server/pkg/log"
)

// FuseScope's community build rejects anomaly_rule, so the alert write tools
// must not teach its strategy or its condition fields.
func TestAlertWriteSchemasOmitAnomalyRule(t *testing.T) {
	h := &Handler{logger: logpkg.New("error")}
	s := newMCPTestServer()
	h.RegisterAlertsHandlers(s)
	tools := listTestTools(t, s)

	for _, name := range []string{"fusescope_create_alert", "fusescope_update_alert"} {
		t.Run(name, func(t *testing.T) {
			tool, ok := tools[name]
			if !ok {
				t.Fatalf("%s is not registered", name)
			}
			raw, err := json.Marshal(tool.Tool.InputSchema)
			if err != nil {
				t.Fatal(err)
			}
			var schema struct {
				Properties map[string]struct {
					Description string         `json:"description"`
					Enum        []string       `json:"enum"`
					Properties  map[string]any `json:"properties"`
				} `json:"properties"`
			}
			if err := json.Unmarshal(raw, &schema); err != nil {
				t.Fatal(err)
			}

			ruleType := schema.Properties["ruleType"]
			if strings.Contains(ruleType.Description, "anomaly") {
				t.Errorf("ruleType description teaches anomaly_rule: %q", ruleType.Description)
			}
			if strings.Join(ruleType.Enum, ",") != "threshold_rule,promql_rule" {
				t.Errorf("ruleType enum = %v, want [threshold_rule promql_rule]", ruleType.Enum)
			}
			condition := schema.Properties["condition"].Properties
			if len(condition) == 0 {
				t.Fatal("condition has no properties")
			}
			for _, field := range []string{"algorithm", "seasonality"} {
				if _, present := condition[field]; present {
					t.Errorf("condition still exposes anomaly-only field %q", field)
				}
			}
		})
	}
}
