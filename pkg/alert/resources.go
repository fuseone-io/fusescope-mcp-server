package alert

// Instructions is the MCP resource content for fusescope://alert/instructions.
const Instructions = `# FuseScope Alert Rule — Instructions

## Overview
An alert rule monitors a signal (metrics, logs, traces, or exceptions) and fires when a condition is met.
The alert is created via POST /api/v2/rules.

threshold_rule and promql_rule use schemaVersion **v2alpha1**: structured thresholds + evaluation + notificationSettings. Always set schemaVersion="v2alpha1" explicitly for them: FuseScope treats an omitted schemaVersion as the v1 schema and then requires the v1 condition fields instead. anomaly_rule takes schemaVersion **v1** only; see "Anomaly Alerts".

## Before Creating or Updating an Alert
1. Read fusescope://alert/examples for complete payloads unless already read for the same prepared operation.
2. Update is a full replacement. Reuse fusescope_get_alert only from the same still-current prepared operation; refresh if state may have changed, then preserve unchanged fields.
3. Use fusescope_get_field_keys to discover filter/groupBy attributes, reusing results from that operation.
4. NOTIFICATION ROUTING: Reuse a fully paginated fusescope_list_notification_channels result only from the same still-current prepared operation; otherwise call it, refreshing only if state may have changed. The list is config-free, defaults to 20 rows, and accepts at most 200 per page; follow its filtered total. The immutable routing identity is displayName, not machine name. For v2 direct routing, every threshold tier needs an exact returned displayName: FuseScope rejects a tier with no channels and a channel name that does not exist. Top-level preferredChannels plays no part in v2 routing (FuseScope stores it unused), so omit it; v1 anomaly rules route through it instead. If no direct channel fits, show the returned choices and ask; if none exists, offer fusescope_create_notification_channel with settings the user provides. Never guess or create automatically. Channel create and update default test to false; set test=true only when the user asks to send a test. Confirmed v2 policy routing sets notificationSettings.usePolicy=true and may omit tier channels.

## Quick Workflow: From User Intent to Payload
A repeatable mental model for going from a user request ("alert me when login p99 > 2s") to a valid payload:
1. **Signal -> alertType.** CPU, memory, latency histograms, request rate -> METRIC_BASED_ALERT. Log lines or log volume -> LOGS_BASED_ALERT. Span latency or span error rate -> TRACES_BASED_ALERT. Exception counts -> EXCEPTIONS_BASED_ALERT. FuseScope validates the value.
2. **Pick ruleType.** Default to threshold_rule. Use promql_rule only if the user provided a PromQL expression. Use anomaly_rule only for deviation from a metric's usual pattern (seasonal history) rather than a fixed value; see "Anomaly Alerts" for its v1 shape.
3. **Pick compositeQuery.queryType + matching envelope type.** See the "Query envelope type" table.
4. **Pick the aggregation shape.** Metrics -> object {metricName, timeAggregation, spaceAggregation}. Logs/traces -> {expression: "count()" | "p99(duration_nano)" | ...}.
5. **Write the filter.** See "Filter & Having Expressions" for the operator set. Prefer resource attributes (service.name, deployment.environment, k8s.*); the backend indexes them.
6. **Configure thresholds.** Every rule requires tiers with name, op, matchType, and target. Direct routing uses exact displayName values from the notification-routing preflight above on every tier; confirmed policy routing sets notificationSettings.usePolicy=true and may omit tier channels.
7. **Evaluation.** Set evaluation explicitly (FuseScope requires it for v2alpha1): kind=rolling with evalWindow=5m and frequency=1m unless the user asked for a different window.
8. **Notification.** Set notificationSettings.groupBy on high-cardinality queries to reduce noise.

The tool's input schema describes every top-level and condition field (accepted values, defaults, which schemaVersion reads each). This resource covers the fields the schema shows as plain objects (query envelopes, threshold tiers, evaluation specs) and the workflow around them.

## Composite Query Structure

condition.compositeQuery uses the v5 query format. Set panelType to "graph" for alerts; FuseScope stores an empty value when omitted.

### Query envelope type (queries[].type)
The envelope type must match compositeQuery.queryType:
| queryType | Accepted envelope types |
|-----------|-------------------------|
| builder | builder_query, builder_formula, builder_trace_operator |
| promql | promql (or builder_formula when composing with another promql query) |
| clickhouse_sql | clickhouse_sql (or builder_formula) |

### Builder query spec (builder_query)
- name: query identifier (A, B, C, …)
- signal: "metrics" | "logs" | "traces" (must match alertType)
- source (metrics only): "meter" to alert on Cost Meter usage/billing metrics (e.g. signoz.meter.log.size); omit otherwise. Works with either evaluation kind: cumulative for daily/monthly spend budgets, rolling for rate/over-time meter alerts.
- stepInterval: interval in seconds (60 for most alerts)
- aggregations: see "Aggregation shapes" below
- filter: {expression: "service.name = 'frontend' AND http.status_code >= 500"}
- groupBy: [{name, fieldContext: "resource" | "attribute", fieldDataType}]
- order and limit: omit both on alert queries. On a grouped query they cap evaluation to the top-N series (FuseScope builds a top-N CTE from them), so series beyond N are silently never evaluated; a "below" alert ordered descending can miss every breaching series. Set them only when top-N alerting is the explicit intent, using Query Builder v5 wire ordering (metrics: [{key: {name: "__result"}, direction: "desc"}]), never dashboard editor orderBy.
- functions: post-query transforms. An anomaly_rule query ends with the anomaly function; see "Anomaly Alerts".
- disabled: true when the query is used only as an input to a formula

### PromQL / ClickHouse query spec (envelope type=promql or clickhouse_sql)
- name: query identifier
- query: the PromQL or SQL string (required)
- legend: legend template
- disabled: false

### PromQL for OTel dotted metric names
Alerts with ruleType=promql_rule on OpenTelemetry metrics must use the Prometheus 3.x UTF-8 quoted-selector form for any name containing a dot. The forms that look natural but return no data in FuseScope:

- Underscored conversion: rate(payment_latency_ms_bucket[5m]). FuseScope does NOT rename dots to underscores.
- Bare dotted name: rate(payment_latency_ms.bucket[5m]). A dot is not a legal identifier char.

Correct form (quote the name inside braces):

  histogram_quantile(0.9, sum(rate({"payment_latency_ms.bucket"}[5m])) by (le))

The same rule applies to dotted label names in by(...) and in matchers:

  sum by (le, "service.name") (
    rate({"payment_latency_ms.bucket", "deployment.environment"="prod"}[5m])
  )

For the full guide (syntax, examples by metric type, anti-pattern table, pre-flight checklist for histogram alerts) read fusescope://promql/instructions. Always read it before writing a promql_rule query.

### Formula query spec (envelope type=builder_formula)
- name: formula identifier (F1, F2, …)
- expression: math expression referencing other query names (e.g. "(A / B) * 100"). Supports +, -, *, /, and functions like abs(), sqrt(), log(), exp()
- legend: legend template
- Set selectedQueryName to the formula name (e.g. "F1") so the alert triggers on the formula result

Omit limit and order on formula and input queries too: a limit ranks groups over the whole evaluation window and drops everything outside the top N, so a group with a short-lived spike can silently never be evaluated.

## Aggregation shapes

### Metrics signal — object shape
Use this shape when spec.signal = "metrics":

` + "```" + `json
"aggregations": [
  {"metricName": "k8s.pod.cpu_request_utilization", "timeAggregation": "avg", "spaceAggregation": "max"}
]
` + "```" + `

- metricName: the metric you're querying.
- timeAggregation: per-series time aggregation. Common: avg, max, min, sum, rate, increase, count, count_distinct, latest. Default by metric type: gauge -> avg, cumulative counter -> rate, delta counter -> rate.
- spaceAggregation: cross-series aggregation. Common: sum, avg, min, max, count. For histograms: p50, p75, p90, p95, p99.

### Logs / traces signal — expression shape
Use this shape when spec.signal = "logs" or "traces":

` + "```" + `json
"aggregations": [{"expression": "count()"}]
` + "```" + `

Common expressions: count(), count_distinct(user_id), avg(duration), sum(bytes), min(x), max(x), p50/p75/p90/p95/p99(duration_nano).

## Filter & Having Expressions

builder_query spec.filter.expression (pre-aggregation) and spec.having.expression (post-aggregation) use the same syntax. Prefer resource attributes in filters; they are the fastest path through the storage backend.

For logs/traces, HAVING aggregation expressions must be included in aggregations. For a percentile count guard, put the percentile first and count() second, then use count() > N in having.expression. Alerts evaluate the first aggregation.

### Operator reference
| Intent | Operator | Example |
|--------|----------|---------|
| Field exists | EXISTS | trace_id EXISTS |
| Field is absent | NOT EXISTS | k8s.pod.name NOT EXISTS |
| Exact match | = | service.name = 'frontend' |
| Not equal (field must exist) | EXISTS AND != | service.name EXISTS AND service.name != 'redis' |
| One of several values | IN | severity_text IN ('ERROR', 'WARN', 'FATAL') |
| Excluded from a set | NOT IN | deployment.environment NOT IN ('dev', 'staging') |
| Substring / pattern | LIKE | name LIKE '%payment%' |
| Case-insensitive pattern | ILIKE | body ILIKE '%timeout%' |
| Simple containment | CONTAINS | body CONTAINS 'timeout' |
| Regex | REGEXP | name REGEXP '^grpc\.' |
| Numeric comparison | >, >=, <, <= | http.status_code >= 500 |

### Data type guardrails
- **bool**: =, !=, EXISTS, NOT EXISTS
- **int / float**: =, !=, >, >=, <, <=, IN, EXISTS, NOT EXISTS
- **string**: =, !=, LIKE, ILIKE, CONTAINS, REGEXP, IN, NOT IN, EXISTS, NOT EXISTS

### Composition
Combine with AND / OR and parentheses for grouping:
- service.name = 'frontend' AND http.status_code >= 500
- (severity_text = 'ERROR' AND service.name = 'payments-api') OR service.name = 'billing-api'

Negative operators (!=, NOT IN, NOT LIKE) only match rows where the field is present. To also exclude rows that lack the field entirely, pair with EXISTS:
- k8s.namespace.name EXISTS AND k8s.namespace.name != 'kube-system'

Common compositeQuery.unit values: percent, percentunit, ms, s, ns, bytes, kbytes, mbytes, gbytes, reqps, ops, cps.

## Threshold Configuration (v2alpha1 — threshold_rule & promql_rule)

condition.thresholds defines one or more routing tiers. Each tier can route to different channels:

` + "```" + `json
"thresholds": {
  "kind": "basic",
  "spec": [
    {
      "name": "critical",
      "target": 1000,
      "op": "above",
      "matchType": "all_the_times",
      "channels": ["pagerduty-oncall", "slack-alerts"]
    },
    {
      "name": "warning",
      "target": 100,
      "op": "above",
      "matchType": "at_least_once",
      "channels": ["slack-alerts"]
    }
  ]
}
` + "```" + `

### Threshold fields
- **name**: tier name (critical, error, warning, or info). Acts as the routing label: alerts carry threshold.name equal to this value. Set labels.severity to match your highest tier.
- **target**: numeric threshold value (required).
- **targetUnit**: unit of the target (e.g. ms, percent, s, bytes). Auto-converted to compositeQuery.unit during evaluation.
- **recoveryTarget**: hysteresis value to avoid flapping (e.g. target=80%, recoveryTarget=75%). null uses the target itself as the recovery point.
- **matchType**: canonical at_least_once, all_the_times, on_average, in_total, last. Aliases accepted: avg (=on_average), sum (=in_total).
- **op**: canonical above, below, equal, not_equal, above_or_equal, below_or_equal, outside_bounds. Short forms accepted: eq, not_eq, above_or_eq, below_or_eq. Symbolic accepted: >, <, =, !=, >=, <=.
- **channels**: existing notification channel displayName values for this tier. Direct v2 routing requires at least one exact returned displayName from fusescope_list_notification_channels on every tier; top-level preferredChannels is not a fallback. Reuse only a same-operation, still-current list result; otherwise call the fully paginated tool, refreshing if state may have changed. With notificationSettings.usePolicy=true, tier channels may be omitted, but supplied displayName values are still validated.

### Choosing targetUnit
- Set targetUnit when the threshold value is in a different unit from the query series. Example: the series emits nanoseconds (compositeQuery.unit="ns") but you want to threshold at "5 seconds". Set target=5, targetUnit="s". FuseScope converts during evaluation.
- If compositeQuery.unit is empty and targetUnit is set, the validator propagates targetUnit onto compositeQuery.unit, so you can omit compositeQuery.unit on single-threshold rules.
- Formulas that compute ratios (e.g. (A/B) * 100) already emit percent. Set compositeQuery.unit="percent" and give the threshold a bare numeric target (no targetUnit).

### Choosing recoveryTarget
- **null / unset**: the alert recovers as soon as the series crosses back through target. Simplest but can flap when the value oscillates at the threshold boundary.
- **non-null**: creates a dead-band between target and recoveryTarget. Example: fire at target=80, recover at recoveryTarget=70. The series must drop below 70 before the alert clears. Recommended for signals that oscillate near the threshold.

## Evaluation (v2alpha1)

evaluation controls how the rule is evaluated:

` + "```" + `json
"evaluation": {
  "kind": "rolling",
  "spec": {"evalWindow": "5m", "frequency": "1m"}
}
` + "```" + `

- evalWindow: how long the condition must persist (e.g. 5m, 15m, 30m, 1h, 4h, 24h).
- frequency: how often to evaluate (e.g. 1m, 5m, 15m).
- Required by FuseScope for threshold/promql rules (schemaVersion v2alpha1); it rejects a rule without an evaluation block. Use a 5m window with 1m frequency unless asked otherwise.
- **Format**: Go duration strings. Both "5m" and "5m0s" are accepted; stick to one style per payload. Common values: 1m, 5m, 15m, 30m, 1h, 4h, 24h, 1d.
- **Sizing tips**: keep evalWindow ≥ frequency. Short windows (1-5m) catch spikes; long windows (30m-1h) smooth noise. For infrequent signals (hourly batch jobs) set frequency to 5-15m to reduce evaluation cost.

### Cumulative window (daily/monthly totals)

A general evaluation kind, independent of signal/source. Use it for any period-total alert (daily error budgets, monthly request counts, Cost Meter spend budgets, …). It accumulates from a fixed reset point instead of using a sliding window:

` + "```" + `json
"evaluation": {
  "kind": "cumulative",
  "spec": {"schedule": {"type": "daily", "minute": 0, "hour": 0}, "frequency": "1m", "timezone": "UTC"}
}
` + "```" + `

- kind "cumulative" sums since the schedule boundary (vs "rolling" sliding window); pair with threshold matchType "in_total".
- schedule.type: "daily" (shown; minute/hour set the reset, e.g. 00:00) or "monthly". timezone e.g. "UTC".

## Notification Settings (v2alpha1)

` + "```" + `json
"notificationSettings": {
  "groupBy": ["service.name", "deployment.environment"],
  "newGroupEvalDelay": "2m",
  "renotify": {
    "enabled": true,
    "interval": "30m",
    "alertStates": ["firing", "nodata"]
  },
  "usePolicy": false
}
` + "```" + `

Set groupBy on high-cardinality queries to reduce noise. usePolicy=false (the default) requires existing channels on every threshold tier; usePolicy=true routes by the organization's notification policies, so tier channels may be omitted.

## Labels & Routing

- labels.severity: MUST be one of critical, error, warning, info. When thresholds is used, threshold.name is the routing tier. Set labels.severity to the highest tier the rule carries.
- Additional labels like team, service, environment enable routing policies.

### Label sources available to routing policies
Routing policies evaluate expressions against three merged label sources:
1. **User static labels** from the labels object on the rule (severity, team, service, environment, …).
2. **Platform labels** auto-injected at fire time:
   - alertname: the rule's alert field
   - threshold.name: the tier that fired (critical | error | warning | info)
   - ruleSource, ruleId: rule metadata
3. **Dynamic labels** from groupBy fields in the query (service.name, k8s.pod.name, http.route, deployment.environment, topic, partition, …).

### Routing-policy matcher operators
Policy expressions use a reduced operator set (not identical to query filters):
- Comparison: =, !=
- Text matching: CONTAINS, REGEXP
- Set membership: IN, NOT IN
- Logical: AND, OR, parentheses

Example: deployment.environment = "production" AND threshold.name = "critical"

### V2 channel-routing modes
| notificationSettings.usePolicy | thresholds[].channels | Effective routing |
|--------------------------------|-----------------------|-------------------|
| false (default) | present on every tier | Send each tier to its listed channels |
| false | absent on any tier | Invalid; preferredChannels is not a v2 fallback |
| true | omit (any supplied names are still validated) | Match alert labels against the org-level routing policy; send to policy-matched channels |

## Anomaly Alerts (ruleType: anomaly_rule)
An anomaly rule scores the selected metrics query against the same time in its 4 previous seasons and fires when the score passes condition.target. The score is unitless: how many scaled deviations the value sits from the seasonal median. It uses schemaVersion **v1** only; FuseScope rejects a v2alpha1 anomaly rule.

Required shape:
- alertType: METRIC_BASED_ALERT; ruleType: anomaly_rule; schemaVersion: "v1".
- condition.compositeQuery: queryType builder with exactly one anomaly function in the request, on a metrics builder_query with one aggregation and no limit, having, source "meter", or timeShift or fillZero before it. The anomaly function is the query's last function, e.g. [{name: "anomaly", args: [{name: "z_score_threshold", value: 3}]}]. The query-time rules of fusescope://metrics-aggregation-guide "Anomaly Function" apply at evaluation.
- condition.selectedQueryName: the query holding the anomaly function.
- condition.target: the z-score, 1 to 7. Leave targetUnit empty; FuseScope ignores it.
- condition.op: above (score rises past target), below (drops past target; keep target positive), or outside_bounds (either way).
- condition.matchType: at_least_once, all_the_times, on_average, in_total or last, as for v1 threshold rules.
- condition.algorithm: "standard" (the default and only value).
- condition.seasonality: "hourly" (default), "daily" or "weekly". Pick the period the traffic repeats over: daily for business-hours patterns, weekly for weekday-vs-weekend.
- evalWindow and frequency (top level, Go durations; defaults 5m and 1m), and preferredChannels: the exact displayName values to notify. Omit thresholds, evaluation and notificationSettings: FuseScope builds them from these fields, renotifies every 4h, and does not use routing policies for v1 rules.

condition.target and condition.seasonality override the function's own z_score_threshold and seasonality, so the band and what fires never disagree.

Alerts carry the score in $value and the target in $threshold, both unitless. An alert scored against a flat baseline (no variation across the seasons) carries the annotation anomaly_baseline: flat; that annotation name is reserved, and FuseScope rejects a rule that sets it.

FuseScope answers 400 for a v2alpha1 schemaVersion, an algorithm other than standard, a target outside 1 to 7, the reserved annotation, a request without exactly one anomaly function, or a selectedQueryName that is not the anomaly query.

Choosing target: 3 is a balanced default; 4 or more catches only strong deviations; 2 catches small ones at the cost of noise. A point needs data in at least 2 of the 4 previous seasons to be scored, so weekly seasonality needs weeks of history.

See fusescope://alert/examples, "Metric anomaly alert (v1)", for a complete payload.

## Set These Explicitly
The payload is forwarded to FuseScope exactly as composed, and FuseScope fills in very little (version="v5" and notificationSettings.usePolicy=false). Everything else is on the author:
- schemaVersion: "v2alpha1" for threshold_rule and promql_rule, "v1" for anomaly_rule; FuseScope treats an omitted schemaVersion as v1 and then requires the v1 condition fields.
- evaluation and notificationSettings: FuseScope rejects a v2alpha1 rule missing either block. A v1 anomaly rule omits both.
- panelType: set "graph"; FuseScope stores an empty value when omitted.
- selectedQueryName: set it on every rule to the query or formula the rule alerts on, even when there is only one query.
- labels.severity and annotations: FuseScope adds no defaults; include them so routing tiers and notification text work.
- Omit server-populated fields (createdAt, updatedAt, createdBy, updatedBy) and the injected webUrl, also when composing an update from a fusescope_get_alert result.

## Further Reading
User-facing docs. Cite these back to the user when they want to understand a concept in depth:
- Alert rules: https://docs.fusescope.fuseone.io/alerts/alert-rules/
- Routing policies and planned downtime: https://docs.fusescope.fuseone.io/alerts/routing-and-downtime/
- Notification channels: https://docs.fusescope.fuseone.io/alerts/notification-channels/
`
