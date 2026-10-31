package dashboard

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderTableDescribesColumnsAndIndexes(t *testing.T) {
	table := Table{
		Name:   "logs_v2",
		Engine: "MergeTree",
		Columns: []Column{
			{Name: "timestamp", Type: "UInt64"},
			{Name: "_retention_days", Type: "UInt16", DefaultKind: "DEFAULT", DefaultExpression: "15"},
			{Name: "traceID", Type: "FixedString(32)", DefaultKind: "ALIAS", DefaultExpression: "trace_id"},
			{Name: "is_error", Type: "Bool", DefaultKind: "MATERIALIZED", DefaultExpression: "status_code = 2"},
		},
		Indexes: []Index{{Name: "body_idx", Expression: "lower(body)", Type: "ngrambf_v1(4, 15000, 3, 0)"}},
	}

	var buf bytes.Buffer
	renderTable(&buf, table)

	assert.Equal(t, `Table: logs_v2

- Engine: MergeTree
- Total Columns: 4
- Total Indexes: 1

Columns:
- timestamp: UInt64
- _retention_days: UInt16 - Default: 15
- traceID: FixedString(32) - Alias: trace_id
- is_error: Bool - Materialized: status_code = 2

Indexes:
- body_idx (lower(body)) - Type: ngrambf_v1(4, 15000, 3, 0)

---

`, buf.String())
}

// Catches a snapshot that was not regenerated after EssentialTables changed.
func TestBundledSchemaCoversEssentialTables(t *testing.T) {
	for signal, database := range signalDatabases {
		t.Run(signal, func(t *testing.T) {
			schema := GetClickHouseSchema(signal)
			assert.Contains(t, schema, "Database: "+database)
			for _, name := range EssentialTables[database] {
				table, ok := bundledSchema.table(database, name)
				require.True(t, ok, "%s.%s is missing from %s; run make gen-chschema", database, name, schemaSnapshotFile)
				assert.NotEmpty(t, table.Columns, "%s.%s has no columns", database, name)
				assert.Contains(t, schema, "Table: "+name+"\n")
			}
		})
	}
}

func TestGetClickHouseSchemaRejectsUnknownSignal(t *testing.T) {
	assert.True(t, strings.HasPrefix(GetClickHouseSchema("profiles"), "Error: unknown signal type 'profiles'"))
}

func TestInitClickhouseSchemaPrefixesReadingInstructions(t *testing.T) {
	InitClickhouseSchema()
	for _, schema := range []string{LogsSchema, MetricsSchema, TracesSchema} {
		assert.True(t, strings.HasPrefix(schema, schemaReadingInstructions()))
		assert.Contains(t, schema, "fusescope-schema-migrator "+bundledSchema.SchemaMigratorVersion)
	}
}
