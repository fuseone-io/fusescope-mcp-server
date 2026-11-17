package tools

import (
	"fmt"
	"slices"

	mcp "github.com/fuseone-io/fusescope-mcp-server/internal/mcpcontract"
	"github.com/google/jsonschema-go/jsonschema"
)

const supportedRuleTypeDescription = "The evaluation strategy: threshold_rule compares against thresholds, promql_rule evaluates a PromQL expression, " +
	"anomaly_rule fires when the anomaly score of a metrics query carrying the anomaly function passes condition.target, a z-score from 1 to 7. " +
	"Defaults to threshold_rule for builder and clickhouse_sql queries and is forced to promql_rule for promql queries. " +
	"anomaly_rule takes schemaVersion v1 only: condition.target, op and matchType, evalWindow, frequency and preferredChannels replace thresholds, evaluation and notificationSettings."

// v1Derived lists the fields the generated PostableRule requires that a
// schemaVersion v1 rule, the only kind anomaly_rule takes, leaves to FuseScope.
var v1Derived = map[string][]string{
	"":          {"evaluation", "notificationSettings"},
	"condition": {"thresholds"},
}

// withSupportedRuleTypes narrows the generated rule schema to the rule types
// FuseScope evaluates, and stops requiring the fields FuseScope derives for a
// v1 anomaly rule: the pinned spec describes anomaly_rule as v2alpha1 only.
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
		ruleType.Enum = []any{"threshold_rule", "promql_rule", "anomaly_rule"}
		for _, field := range []string{"algorithm", "seasonality"} {
			if _, ok := condition.Properties[field]; !ok {
				panic(fmt.Sprintf("tool %q: condition has no %s, which anomaly_rule needs", t.Name, field))
			}
		}
		for property, fields := range v1Derived {
			target := schema
			if property != "" {
				target = schema.Properties[property]
			}
			for _, field := range fields {
				index := slices.Index(target.Required, field)
				if index < 0 {
					panic(fmt.Sprintf("tool %q: %q no longer requires %s; drop it from the overlay", t.Name, property, field))
				}
				target.Required = slices.Delete(target.Required, index, index+1)
			}
		}
	}
}
