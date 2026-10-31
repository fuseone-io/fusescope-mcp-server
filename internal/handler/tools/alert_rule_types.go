package tools

import (
	"fmt"

	mcp "github.com/fuseone-io/fusescope-mcp-server/internal/mcpcontract"
	"github.com/google/jsonschema-go/jsonschema"
)

const supportedRuleTypeDescription = "The evaluation strategy: threshold_rule compares against thresholds, promql_rule evaluates a PromQL expression. " +
	"Defaults to threshold_rule for builder and clickhouse_sql queries and is forced to promql_rule for promql queries."

// withSupportedRuleTypes narrows the generated rule schema to the rule types
// FuseScope evaluates: the generated PostableRule still describes anomaly_rule
// and its condition.algorithm and condition.seasonality fields.
func withSupportedRuleTypes() mcp.ToolOption {
	return func(t *mcp.Tool) {
		schema, ok := t.InputSchema.(*jsonschema.Schema)
		if !ok {
			panic(fmt.Sprintf("tool %q: rule-type overlay needs an inferred input schema, got %T", t.Name, t.InputSchema))
		}
		ruleType, condition := schema.Properties["ruleType"], schema.Properties["condition"]
		if ruleType == nil || condition == nil {
			panic(fmt.Sprintf("tool %q: input schema has no ruleType or condition", t.Name))
		}
		ruleType.Description = supportedRuleTypeDescription
		ruleType.Enum = []any{"threshold_rule", "promql_rule"}
		for _, field := range []string{"algorithm", "seasonality"} {
			if _, ok := condition.Properties[field]; !ok {
				panic(fmt.Sprintf("tool %q: condition has no %s; drop it from the overlay", t.Name, field))
			}
			delete(condition.Properties, field)
		}
	}
}
