package alert

import _ "embed"

// generatedExamples is rendered by tools/specgen/specgen.py from the request
// examples SigNoz publishes on POST /api/v2/rules in the pinned OpenAPI spec.
//
//go:embed examples.generated.md
var generatedExamples string

// Examples is the MCP resource content for fusescope://alert/examples: the
// SigNoz-published examples between a hand-written routing preamble and the
// examples the spec does not carry: a Cost Meter budget and a v1 anomaly rule.
var Examples = examplesPreamble + generatedExamples + examplesAddendum

const examplesPreamble = `# FuseScope Alert Rule Examples

Every example uses schemaVersion v2alpha1 except the metric anomaly alert, which uses v1, the only schema anomaly_rule takes. Adapt each example to the target workspace: discover filter and groupBy keys with fusescope_get_field_keys instead of copying attribute names.

**Before using any example:** channel displayName values are illustrative. Reuse a fully paginated fusescope_list_notification_channels result only from the same still-current prepared operation; otherwise call it and replace every direct-routing value with an exact returned displayName. If none fits, ask the user or offer fusescope_create_notification_channel with settings the user provides; never create automatically. Channel creation defaults test to false. Confirmed v2 policy routing may remove tier channels and set notificationSettings.usePolicy=true.

`

const examplesAddendum = `
## Cost Meter budget alert (cumulative window, source "meter")

Not in the FuseScope OpenAPI spec yet. Fires when today's total log ingestion exceeds 10 GiB. The query targets Cost Meter (source "meter") with timeAggregation increase; the threshold uses matchType in_total over a cumulative daily window. For a currency budget, add a builder_formula converting bytes to cost (for example "(A / 1e9) * cost_per_gb") and set selectedQueryName to the formula.

` + "```" + `json
{
  "alert": "Daily log ingestion budget",
  "alertType": "METRIC_BASED_ALERT",
  "ruleType": "threshold_rule",
  "schemaVersion": "v2alpha1",
  "labels": {"severity": "critical"},
  "condition": {
    "compositeQuery": {
      "queryType": "builder",
      "panelType": "graph",
      "queries": [
        {
          "type": "builder_query",
          "spec": {
            "name": "A",
            "signal": "metrics",
            "source": "meter",
            "stepInterval": 3600,
            "aggregations": [
              {"metricName": "signoz.meter.log.size", "timeAggregation": "increase", "spaceAggregation": "sum"}
            ]
          }
        }
      ]
    },
    "selectedQueryName": "A",
    "thresholds": {
      "kind": "basic",
      "spec": [
        {"name": "critical", "target": 10737418240, "op": "above", "matchType": "in_total", "channels": ["my-channel"]}
      ]
    }
  },
  "evaluation": {
    "kind": "cumulative",
    "spec": {"schedule": {"type": "daily", "minute": 0, "hour": 0}, "frequency": "1m", "timezone": "UTC"}
  },
  "notificationSettings": {"renotify": {"enabled": false, "interval": "30m"}}
}
` + "```" + `

## Metric anomaly alert (v1)

Not in the FuseScope OpenAPI spec yet. Fires when the request rate of a service rises more than 3 scaled deviations above the same time on the 4 previous days. condition.target and condition.seasonality override the anomaly function's args; preferredChannels routes the alert.

` + "```" + `json
{
  "alert": "Request rate anomaly",
  "alertType": "METRIC_BASED_ALERT",
  "ruleType": "anomaly_rule",
  "schemaVersion": "v1",
  "evalWindow": "10m",
  "frequency": "1m",
  "labels": {"severity": "warning"},
  "annotations": {"summary": "Request rate z-score {{$value}} above {{$threshold}}"},
  "condition": {
    "compositeQuery": {
      "queryType": "builder",
      "panelType": "graph",
      "queries": [
        {
          "type": "builder_query",
          "spec": {
            "name": "A",
            "signal": "metrics",
            "stepInterval": 60,
            "aggregations": [
              {"metricName": "http_requests_total", "timeAggregation": "rate", "spaceAggregation": "sum"}
            ],
            "groupBy": [{"name": "service.name", "fieldContext": "attribute", "fieldDataType": "string"}],
            "functions": [{"name": "anomaly", "args": [{"name": "z_score_threshold", "value": 3}]}]
          }
        }
      ]
    },
    "selectedQueryName": "A",
    "target": 3,
    "op": "above",
    "matchType": "at_least_once",
    "algorithm": "standard",
    "seasonality": "daily"
  },
  "preferredChannels": ["my-channel"]
}
` + "```" + `

## Key Notes
1. Metrics signal uses the object aggregation shape ({metricName, timeAggregation, spaceAggregation}); logs and traces use the expression shape ({expression: "count()"}).
2. selectedQueryName references the query or formula that determines the alert.
3. Direct routing uses exact displayName values from a same-operation, still-current fusescope_list_notification_channels result, or calls the fully paginated tool when absent or stale. V2 needs a displayName on every tier; confirmed policy routing may omit them.
4. Set schemaVersion="v2alpha1" and include evaluation and notificationSettings for threshold_rule and promql_rule; FuseScope requires both and treats an omitted schemaVersion as v1. anomaly_rule takes schemaVersion "v1" with condition.target, op, matchType, evalWindow, frequency and preferredChannels instead.
`
