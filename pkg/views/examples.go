package views

import _ "embed"

// generatedExamples is rendered by tools/specgen/specgen.py from the request
// examples SigNoz publishes on the saved-view operations in the pinned
// OpenAPI spec.
//
//go:embed examples.generated.md
var generatedExamples string

// Examples is the body of fusescope://view/examples: complete SavedView payloads
// for fusescope_create_view and fusescope_update_view, each in the v2 typed spec
// shape ({name, source, schemaVersion, spec{...}}).
var Examples = examplesPreamble + generatedExamples

const examplesPreamble = `# Saved View Examples (v2 typed spec)

Each create example is a complete fusescope_create_view argument; the replace example is the "view" argument of fusescope_update_view. Copy the structure, not the attribute names: discover the target workspace's fields with fusescope_get_field_keys first. A Cost Meter view is source "meter" with signal "metrics" and builder spec source "meter" in every query; see the meter_time_series example.

`
