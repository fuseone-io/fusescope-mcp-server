// Command chschema writes pkg/dashboard/clickhouse_schema.json from the system
// tables of a ClickHouse server that fusescope-schema-migrator has migrated.
// Run it through make gen-chschema.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/fuseone-io/fusescope-mcp-server/pkg/dashboard"
)

func main() {
	clickhouseURL := flag.String("url", "http://localhost:8123", "ClickHouse HTTP interface")
	migratorVersion := flag.String("migrator-version", "", "fusescope-schema-migrator release that created the schema")
	out := flag.String("out", "pkg/dashboard/clickhouse_schema.json", "snapshot file to write")
	flag.Parse()

	if err := run(*clickhouseURL, *migratorVersion, *out); err != nil {
		fmt.Fprintln(os.Stderr, "chschema:", err)
		os.Exit(1)
	}
}

func run(clickhouseURL, migratorVersion, out string) error {
	if migratorVersion == "" {
		return fmt.Errorf("-migrator-version is required")
	}

	client := chClient{base: clickhouseURL}
	snapshot := dashboard.SchemaSnapshot{SchemaMigratorVersion: migratorVersion}

	databases := make([]string, 0, len(dashboard.EssentialTables))
	for database := range dashboard.EssentialTables {
		databases = append(databases, database)
	}
	sort.Strings(databases)

	for _, database := range databases {
		names := append([]string(nil), dashboard.EssentialTables[database]...)
		sort.Strings(names)
		for _, name := range names {
			table, err := client.describe(database, name)
			if err != nil {
				return fmt.Errorf("%s.%s: %w", database, name, err)
			}
			snapshot.Tables = append(snapshot.Tables, table)
		}
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(out, append(data, '\n'), 0o644)
}

type chClient struct {
	base string
}

func (c chClient) describe(database, name string) (dashboard.Table, error) {
	params := map[string]string{"db": database, "table": name}

	var tables []struct {
		Engine string `json:"engine"`
	}
	if err := c.query("SELECT engine FROM system.tables WHERE database = {db:String} AND name = {table:String}", params, &tables); err != nil {
		return dashboard.Table{}, err
	}
	if len(tables) != 1 {
		return dashboard.Table{}, fmt.Errorf("table not found")
	}

	table := dashboard.Table{Database: database, Name: name, Engine: tables[0].Engine}
	if err := c.query(`SELECT name, type, default_kind, default_expression FROM system.columns
		WHERE database = {db:String} AND table = {table:String} ORDER BY position`, params, &table.Columns); err != nil {
		return dashboard.Table{}, err
	}
	if err := c.query(`SELECT name, expr AS expression, type_full AS type FROM system.data_skipping_indices
		WHERE database = {db:String} AND table = {table:String}`, params, &table.Indexes); err != nil {
		return dashboard.Table{}, err
	}
	return table, nil
}

// query decodes the rows of the result into rows, a pointer to a slice.
func (c chClient) query(sql string, params map[string]string, rows any) error {
	values := url.Values{}
	for key, value := range params {
		values.Set("param_"+key, value)
	}

	resp, err := http.Post(c.base+"/?"+values.Encode(), "text/plain", strings.NewReader(sql+" FORMAT JSON"))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("clickhouse: %s: %s", resp.Status, bytes.TrimSpace(body))
	}

	result := struct {
		Data any `json:"data"`
	}{Data: rows}
	return json.Unmarshal(body, &result)
}
