package dashboard

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Makes the LLM schema aware so it writes correct ClickHouse queries.

const schemaSnapshotFile = "clickhouse_schema.json"

//go:embed clickhouse_schema.json
var schemaSnapshotJSON []byte

var bundledSchema = mustParseSchemaSnapshot(schemaSnapshotJSON)

func schemaReadingInstructions() string {
	return `
## Provenance and compatibility

This reference was read from the tables that fusescope-schema-migrator ` + bundledSchema.SchemaMigratorVersion + `
creates, and is bundled with this MCP server. A tenant may run a different FuseScope
version or have schema changes not represented here. Use the reference to compose
queries, but treat table/column/type errors returned by the target tenant as
authoritative and revise the query rather than insisting that this bundled snapshot
is exact.

## Instructions for using this ClickHouse schema

1. Read every table, column, and data type exactly as written.
2. Never skip or simplify types. Use LowCardinality(...), Map(...), JSON(...),
   AggregateFunction(...), Array(...), Tuple(...), Nullable(...) exactly as shown.
3. Do not infer or assume types absent from this snapshot.
4. When writing queries:
   • use exact column names
   • use exact data types
   • verify the column exists in the table
5. Aliases may be used, but their type is identical to the base column.
6. Do not hallucinate additional fields or hidden schema.
`
}

var LogsSchema string
var MetricsSchema string
var TracesSchema string

func InitClickhouseSchema() {
	LogsSchema = schemaReadingInstructions() + GetClickHouseSchema("logs")
	MetricsSchema = schemaReadingInstructions() + GetClickHouseSchema("metrics")
	TracesSchema = schemaReadingInstructions() + GetClickHouseSchema("traces")
}

var signalDatabases = map[string]string{
	"logs":    "signoz_logs",
	"metrics": "signoz_metrics",
	"traces":  "signoz_traces",
}

// EssentialTables lists, per database, the tables bundled in the snapshot and
// described to clients.
var EssentialTables = map[string][]string{
	"signoz_logs": {
		"logs_v2",
		"distributed_logs_v2",
		"logs_v2_resource",
		"distributed_logs_v2_resource",
		"tag_attributes_v2",
		"distributed_tag_attributes_v2",
	},
	"signoz_metrics": {
		"samples_v4",
		"distributed_samples_v4",
		"time_series_v4",
		"distributed_time_series_v4",
		"time_series_v4_6hrs",
		"distributed_time_series_v4_6hrs",
		"time_series_v4_1day",
		"distributed_time_series_v4_1day",
		"exp_hist",
		"distributed_exp_hist",
	},
	"signoz_traces": {
		"signoz_index_v3",
		"distributed_signoz_index_v3",
		"signoz_spans",
		"distributed_signoz_spans",
		"traces_v3_resource",
		"distributed_traces_v3_resource",
		"tag_attributes_v2",
		"distributed_tag_attributes_v2",
		"dependency_graph_minutes_v2",
		"distributed_dependency_graph_minutes_v2",
		"trace_summary",
		"distributed_trace_summary",
		"signoz_error_index_v2",
		"distributed_signoz_error_index_v2",
		"top_level_operations",
		"distributed_top_level_operations",
		"span_attributes_keys",
		"distributed_span_attributes_keys",
	},
}

// SchemaSnapshot is the JSON form of clickhouse_schema.json, written by
// tools/chschema from ClickHouse's system tables.
type SchemaSnapshot struct {
	SchemaMigratorVersion string  `json:"schema_migrator_version"`
	Tables                []Table `json:"tables"`
}

type Table struct {
	Database string   `json:"database"`
	Name     string   `json:"name"`
	Engine   string   `json:"engine"`
	Columns  []Column `json:"columns"`
	Indexes  []Index  `json:"indexes,omitempty"`
}

// Column mirrors system.columns; DefaultKind is DEFAULT, MATERIALIZED, ALIAS,
// EPHEMERAL or empty.
type Column struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	DefaultKind       string `json:"default_kind,omitempty"`
	DefaultExpression string `json:"default_expression,omitempty"`
}

type Index struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
	Type       string `json:"type"`
}

func mustParseSchemaSnapshot(data []byte) SchemaSnapshot {
	var snapshot SchemaSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		panic(fmt.Sprintf("parse %s: %v", schemaSnapshotFile, err))
	}
	return snapshot
}

func (s SchemaSnapshot) table(database, name string) (Table, bool) {
	for _, table := range s.Tables {
		if table.Database == database && table.Name == name {
			return table, true
		}
	}
	return Table{}, false
}

// GetClickHouseSchema returns the ClickHouse schema for the signal ("logs",
// "metrics" or "traces") as text.
func GetClickHouseSchema(signal string) string {
	var buf bytes.Buffer

	database, ok := signalDatabases[signal]
	if !ok {
		fmt.Fprintf(&buf, "Error: unknown signal type '%s'. Use 'logs', 'metrics', or 'traces'\n", signal)
		return buf.String()
	}

	fmt.Fprintf(&buf, "\n=== %s SCHEMA ===\n", strings.ToUpper(signal))
	renderDatabase(&buf, database)
	return buf.String()
}

func renderDatabase(buf *bytes.Buffer, database string) {
	var tables []Table
	for _, name := range EssentialTables[database] {
		if table, ok := bundledSchema.table(database, name); ok {
			tables = append(tables, table)
		}
	}
	sort.Slice(tables, func(i, j int) bool { return tables[i].Name < tables[j].Name })

	fmt.Fprintf(buf, "\nDatabase: %s\n\n", database)
	fmt.Fprintf(buf, "Total Tables: %d\n\n", len(tables))

	var local, distributed []Table
	for _, table := range tables {
		if strings.HasPrefix(table.Name, "distributed_") {
			distributed = append(distributed, table)
		} else {
			local = append(local, table)
		}
	}

	if len(local) > 0 {
		buf.WriteString("Local Tables\n\n")
		for _, table := range local {
			renderTable(buf, table)
		}
	}

	if len(distributed) > 0 {
		buf.WriteString("Distributed Tables\n\n")
		for _, table := range distributed {
			renderTable(buf, table)
		}
	}
}

var defaultKindLabels = map[string]string{
	"DEFAULT":      "Default",
	"MATERIALIZED": "Materialized",
	"ALIAS":        "Alias",
	"EPHEMERAL":    "Ephemeral",
}

func renderTable(buf *bytes.Buffer, table Table) {
	fmt.Fprintf(buf, "Table: %s\n\n", table.Name)

	fmt.Fprintf(buf, "- Engine: %s\n", table.Engine)
	fmt.Fprintf(buf, "- Total Columns: %d\n", len(table.Columns))
	fmt.Fprintf(buf, "- Total Indexes: %d\n\n", len(table.Indexes))

	if len(table.Columns) > 0 {
		buf.WriteString("Columns:\n")
		for _, col := range table.Columns {
			line := fmt.Sprintf("- %s: %s", col.Name, col.Type)
			if label, ok := defaultKindLabels[col.DefaultKind]; ok {
				line += fmt.Sprintf(" - %s: %s", label, col.DefaultExpression)
			}
			buf.WriteString(line + "\n")
		}
		buf.WriteString("\n")
	}

	if len(table.Indexes) > 0 {
		buf.WriteString("Indexes:\n")
		for _, idx := range table.Indexes {
			fmt.Fprintf(buf, "- %s (%s) - Type: %s\n", idx.Name, idx.Expression, idx.Type)
		}
		buf.WriteString("\n")
	}

	buf.WriteString("---\n\n")
}
