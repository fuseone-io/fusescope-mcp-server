package querybuilder

import _ "embed"

// generatedExamples is rendered by tools/specgen/specgen.py from the request
// examples SigNoz publishes on POST /api/v5/query_range in the pinned OpenAPI
// spec.
//
//go:embed examples.generated.md
var generatedExamples string

// Examples is the body of fusescope://query-builder/examples: complete
// query_range requests for fusescope_execute_builder_query across builder,
// formula, PromQL, and ClickHouse SQL queries and the raw, scalar, and
// time_series request types.
var Examples = examplesPreamble + generatedExamples

const examplesPreamble = `# Query Builder v5 Request Examples

Each body below is the "query" argument of fusescope_execute_builder_query. start and end are absolute Unix milliseconds; replace them with the window the user asked for. Copy the structure, not the attribute names: discover the target workspace's fields with fusescope_get_field_keys first. For filter syntax and field discovery read the signal's guide (fusescope://logs/query-builder-guide, fusescope://traces/query-builder-guide, fusescope://metrics-aggregation-guide); for PromQL read fusescope://promql/instructions.

`
