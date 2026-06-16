package plugins

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

func TestFormatWebSQLDDLCreateTable(t *testing.T) {
	input := "CREATE TABLE agent_message (`agent_message_id` text,`seq_no` integer,`content_json` text,PRIMARY KEY(`agent_message_id`))"
	got := formatWebSQLDDL(input)
	if !strings.Contains(got, "CREATE TABLE agent_message (\n") {
		t.Fatalf("expected multiline CREATE TABLE, got:\n%s", got)
	}
	if !strings.Contains(got, "  `seq_no` integer,\n") {
		t.Fatalf("expected indented column, got:\n%s", got)
	}
	if !strings.Contains(got, "  PRIMARY KEY(`agent_message_id`)\n") {
		t.Fatalf("expected primary key clause, got:\n%s", got)
	}
	if !strings.HasSuffix(got, ";") {
		t.Fatalf("expected semicolon, got:\n%s", got)
	}
}

func TestFormatWebSQLDDLKeepsQuotedCommas(t *testing.T) {
	input := "CREATE TABLE demo (id integer primary key, note text default 'a,b', check(length(note) > 0))"
	got := formatWebSQLDDL(input)
	if strings.Count(got, "\n  ") != 3 {
		t.Fatalf("expected three top-level entries, got:\n%s", got)
	}
	if !strings.Contains(got, "note text DEFAULT 'a,b',") {
		t.Fatalf("expected quoted comma to stay in one clause, got:\n%s", got)
	}
}

func TestFormatWebSQLDDLCreateIndex(t *testing.T) {
	input := "CREATE UNIQUE INDEX `idx_msg_run_seq` ON `agent_message` (`agent_run_id`, `seq_no`)"
	got := formatWebSQLDDL(input)
	if !strings.Contains(got, "CREATE UNIQUE INDEX `idx_msg_run_seq` ON `agent_message` (\n") {
		t.Fatalf("expected multiline CREATE INDEX, got:\n%s", got)
	}
	if !strings.Contains(got, "  `agent_run_id`,\n  `seq_no`\n") {
		t.Fatalf("expected indented index columns, got:\n%s", got)
	}
}

func TestReadWebSQLRowsPage(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if _, err := db.ExecContext(context.Background(), "CREATE TABLE demo(id INTEGER PRIMARY KEY, name TEXT); INSERT INTO demo(name) VALUES ('a'),('b'),('c'),('d');"); err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}
	rows, err := db.QueryContext(context.Background(), "SELECT id, name FROM demo ORDER BY id")
	if err != nil {
		t.Fatalf("query sqlite: %v", err)
	}
	defer rows.Close()
	_, gotRows, truncated, skipped, hasNext, err := readWebSQLRowsPage(rows, 1, 2)
	if err != nil {
		t.Fatalf("read page: %v", err)
	}
	if skipped != 1 {
		t.Fatalf("expected skipped=1, got %d", skipped)
	}
	if !truncated || !hasNext {
		t.Fatalf("expected truncated/hasNext true, got truncated=%v hasNext=%v", truncated, hasNext)
	}
	if len(gotRows) != 2 || gotRows[0]["name"] != "b" || gotRows[1]["name"] != "c" {
		t.Fatalf("unexpected rows: %#v", gotRows)
	}
}

func TestExecuteWebSQLSelectPage(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if _, err := db.ExecContext(context.Background(), "CREATE TABLE demo(id INTEGER PRIMARY KEY, name TEXT); INSERT INTO demo(name) VALUES ('alpha'),('beta'),('gamma'),('delta');"); err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}

	service := &WebSQLService{}
	data, keepDB, err := service.executeWebSQL(
		context.Background(),
		db,
		"sqlite",
		"SELECT id, name FROM demo ORDER BY id",
		webSQLConnectionConfig{Driver: "sqlite"},
		webSQLExecuteRequest{MaxRows: 3, PageSize: 2, CursorOffset: 1},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("execute select: %v", err)
	}
	if keepDB {
		t.Fatalf("select query should not retain db")
	}
	if data["statement_type"] != "select" {
		t.Fatalf("expected select statement type, got %#v", data["statement_type"])
	}
	if data["row_count"] != 2 || data["cursor_offset"] != 1 || data["page_size"] != 2 {
		t.Fatalf("unexpected paging metadata: %#v", data)
	}
	if data["has_next"] != true || data["has_prev"] != true || data["next_cursor"] != 3 || data["prev_cursor"] != 0 {
		t.Fatalf("unexpected cursor metadata: %#v", data)
	}
	rows, ok := data["rows"].([]map[string]interface{})
	if !ok || len(rows) != 2 {
		t.Fatalf("expected two result rows, got %#v", data["rows"])
	}
	if rows[0]["name"] != "beta" || rows[1]["name"] != "gamma" {
		t.Fatalf("unexpected result rows: %#v", rows)
	}
}

func TestLoadSQLiteCompletionSchema(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if _, err := db.ExecContext(context.Background(), "CREATE TABLE websql_completion_demo(id INTEGER PRIMARY KEY, name TEXT);"); err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}
	data, err := loadWebSQLSchema(context.Background(), db, "sqlite", "", webSQLSchemaRequest{Scope: "completion"})
	if err != nil {
		t.Fatalf("load completion schema: %v", err)
	}
	tables, ok := data["tables"].([]map[string]interface{})
	if !ok || len(tables) == 0 {
		t.Fatalf("expected completion tables, got %#v", data["tables"])
	}
	var demo map[string]interface{}
	for _, table := range tables {
		if table["name"] == "websql_completion_demo" {
			demo = table
			break
		}
	}
	if demo == nil {
		t.Fatalf("expected demo table in completion schema: %#v", tables)
	}
	columns, ok := demo["columns"].([]map[string]interface{})
	if !ok || len(columns) != 2 {
		t.Fatalf("expected demo columns in completion schema, got %#v", demo["columns"])
	}
	if columns[0]["name"] != "id" || columns[1]["name"] != "name" {
		t.Fatalf("unexpected completion columns: %#v", columns)
	}
}

func TestLoadSQLiteSchemaFolderSearchAndObject(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if _, err := db.ExecContext(context.Background(), `
CREATE TABLE websql_schema_demo(id INTEGER PRIMARY KEY, display_name TEXT);
CREATE INDEX idx_websql_schema_demo_name ON websql_schema_demo(display_name);
CREATE VIEW websql_schema_demo_view AS SELECT id, display_name FROM websql_schema_demo;
`); err != nil {
		t.Fatalf("seed sqlite: %v", err)
	}

	root, err := loadWebSQLSchema(context.Background(), db, "sqlite", "", webSQLSchemaRequest{Scope: "root"})
	if err != nil {
		t.Fatalf("load root schema: %v", err)
	}
	schemas, ok := root["schemas"].([]map[string]interface{})
	if !ok || len(schemas) != 1 {
		t.Fatalf("expected one sqlite schema, got %#v", root["schemas"])
	}
	folders, ok := schemas[0]["folders"].([]map[string]interface{})
	if !ok || len(folders) != 3 {
		t.Fatalf("expected table/view/index folders, got %#v", schemas[0]["folders"])
	}

	folder, err := loadWebSQLSchema(context.Background(), db, "sqlite", "", webSQLSchemaRequest{Scope: "folder", FolderType: "tables"})
	if err != nil {
		t.Fatalf("load table folder: %v", err)
	}
	tables, ok := folder["tables"].([]map[string]interface{})
	if !ok || len(tables) != 1 || tables[0]["name"] != "websql_schema_demo" {
		t.Fatalf("unexpected table folder: %#v", folder["tables"])
	}

	search, err := loadWebSQLSchema(context.Background(), db, "sqlite", "", webSQLSchemaRequest{Scope: "search", Query: "display"})
	if err != nil {
		t.Fatalf("search sqlite schema: %v", err)
	}
	searchTables, ok := search["tables"].([]map[string]interface{})
	if !ok || len(searchTables) == 0 {
		t.Fatalf("expected matching table from column search, got %#v", search["tables"])
	}
	columns, ok := searchTables[0]["columns"].([]map[string]interface{})
	if !ok || len(columns) == 0 || columns[0]["name"] != "display_name" {
		t.Fatalf("expected matching display_name column, got %#v", searchTables[0]["columns"])
	}

	object, err := loadWebSQLSchema(context.Background(), db, "sqlite", "", webSQLSchemaRequest{Scope: "object", ObjectType: "table", ObjectName: "websql_schema_demo"})
	if err != nil {
		t.Fatalf("load sqlite object: %v", err)
	}
	detail, ok := object["object"].(map[string]interface{})
	if !ok || detail["name"] != "websql_schema_demo" || detail["columns_loaded"] != true {
		t.Fatalf("unexpected object detail: %#v", object["object"])
	}
	indexes, ok := detail["indexes"].([]map[string]interface{})
	if !ok || len(indexes) != 1 || indexes[0]["name"] != "idx_websql_schema_demo_name" {
		t.Fatalf("expected object index detail, got %#v", detail["indexes"])
	}
}

func TestNormalizeWebSQLCommitMode(t *testing.T) {
	if got := normalizeWebSQLCommitMode("manual"); got != webSQLCommitModeManual {
		t.Fatalf("expected manual, got %s", got)
	}
	if got := normalizeWebSQLCommitMode(""); got != webSQLCommitModeDirect {
		t.Fatalf("expected direct fallback, got %s", got)
	}
}

func TestNormalizeOracleExecutionSQL(t *testing.T) {
	if got := normalizeWebSQLExecutionSQL("oracle", `SELECT * FROM "BPS_ACCOUNT_RECORD";`); got != `SELECT * FROM "BPS_ACCOUNT_RECORD"` {
		t.Fatalf("expected oracle select semicolon to be stripped, got %q", got)
	}
	if got := normalizeWebSQLExecutionSQL("oracle", `BEGIN NULL; END;`); got != `BEGIN NULL; END;` {
		t.Fatalf("expected oracle pl/sql block to keep semicolon, got %q", got)
	}
	if got := normalizeWebSQLExecutionSQL("sqlite", `SELECT 1;`); got != `SELECT 1;` {
		t.Fatalf("expected sqlite semicolon to be preserved, got %q", got)
	}
}

func TestDetectSQLiteSyntaxErrorLocation(t *testing.T) {
	sqlText := "SELECT name, type FROM  WHERE type IN ('table','view') ORDER BY type, name;"
	loc := detectWebSQLErrorLocation("sqlite", sqlText, `执行查询失败: SQL logic error: near "WHERE": syntax error (1)`)
	if loc.Line != 1 || loc.Column != 25 || loc.Token != "WHERE" || loc.Kind != "syntax_error" {
		t.Fatalf("unexpected sqlite syntax location: %#v", loc)
	}
}

func TestDetectSQLiteUnknownColumnLocation(t *testing.T) {
	sqlText := "SELECT kk FROM user_account"
	loc := detectWebSQLErrorLocation("sqlite", sqlText, "执行查询失败: SQL logic error: no such column: kk (1)")
	if loc.Line != 1 || loc.Column != 8 || loc.Token != "kk" || loc.Kind != "unknown_column" {
		t.Fatalf("unexpected sqlite column location: %#v", loc)
	}
}

func TestDetectSQLiteMultilineUnknownColumnLocation(t *testing.T) {
	cases := []struct {
		name       string
		sqlText    string
		errText    string
		wantLine   int
		wantColumn int
		wantToken  string
		wantKind   string
	}{
		{
			name:       "where-field",
			sqlText:    "select *\nfrom attachment\nwhere attachment_id1=1",
			errText:    "执行查询失败: SQL logic error: no such column: attachment_id1 (1)",
			wantLine:   3,
			wantColumn: 7,
			wantToken:  "attachment_id1",
			wantKind:   "unknown_column",
		},
		{
			name:       "select-list-field",
			sqlText:    "select\n  attachment_id,\n  filename,\n  attachment_id1,\n  path\nfrom attachment;",
			errText:    "执行查询失败: SQL logic error: no such column: attachment_id1 (1)",
			wantLine:   4,
			wantColumn: 3,
			wantToken:  "attachment_id1",
			wantKind:   "unknown_column",
		},
		{
			name:       "unknown-table",
			sqlText:    "select attachment_id\nfrom attachment_missing\nwhere attachment_id=1;",
			errText:    "执行查询失败: SQL logic error: no such table: attachment_missing (1)",
			wantLine:   2,
			wantColumn: 6,
			wantToken:  "attachment_missing",
			wantKind:   "unknown_table",
		},
		{
			name:       "syntax-before-where",
			sqlText:    "SELECT attachment_id\nFROM\nWHERE attachment_id=1;",
			errText:    `执行查询失败: SQL logic error: near "WHERE": syntax error (1)`,
			wantLine:   3,
			wantColumn: 1,
			wantToken:  "WHERE",
			wantKind:   "syntax_error",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			loc := detectWebSQLErrorLocation("sqlite", tt.sqlText, tt.errText)
			if loc.Line != tt.wantLine || loc.Column != tt.wantColumn || loc.Token != tt.wantToken || loc.Kind != tt.wantKind {
				t.Fatalf("unexpected sqlite multiline location: %#v", loc)
			}
		})
	}
}

func TestDetectOraclePositionSyntaxErrorLocation(t *testing.T) {
	sqlText := `SELECT * FROM "BPS_ACCOUNT_RECORD" FETCH FIRST 100 ROWS ONLY`
	pos := strings.Index(sqlText, "FETCH") + 1
	errText := fmt.Sprintf("执行查询失败: ORA-00933: SQL command not properly ended\n error occur at position: %d", pos)
	loc := detectWebSQLErrorLocation("oracle", sqlText, errText)
	if loc.Line != 1 || loc.Column != pos || loc.Token != "FETCH" || loc.Kind != "syntax_error" {
		t.Fatalf("unexpected oracle syntax location: %#v", loc)
	}
}

func TestDetectMySQLSyntaxErrorLocation(t *testing.T) {
	sqlText := "SELECT name, type FROM\nWHERE type IN ('table','view') ORDER BY type, name;"
	errText := "执行查询失败: Error 1064 (42000): You have an error in your SQL syntax; check the manual that corresponds to your MySQL server version for the right syntax to use near 'WHERE type IN ('table','view') ORDER BY type, name' at line 2"
	loc := detectWebSQLErrorLocation("mysql", sqlText, errText)
	if loc.Line != 2 || loc.Column != 1 || loc.Token != "WHERE type IN ('table','view') ORDER BY type, name" || loc.Kind != "syntax_error" {
		t.Fatalf("unexpected mysql syntax location: %#v", loc)
	}
}

func TestDetectMySQLUnknownColumnLocation(t *testing.T) {
	sqlText := "select kk from user_account"
	loc := detectWebSQLErrorLocation("mysql", sqlText, "执行查询失败: Error 1054 (42S22): Unknown column 'kk' in 'field list'")
	if loc.Line != 1 || loc.Column != 8 || loc.Token != "kk" || loc.Kind != "unknown_column" {
		t.Fatalf("unexpected mysql column location: %#v", loc)
	}
}

func TestWebSQLErrorSelectionOffset(t *testing.T) {
	loc := detectWebSQLErrorLocation("mysql", "select kk from user_account", "执行查询失败: Error 1054 (42S22): Unknown column 'kk' in 'field list'")
	applyWebSQLSelectionOffset(&loc, map[string]interface{}{
		"selection_start_line":   4,
		"selection_start_column": 3,
	})
	if loc.Line != 4 || loc.Column != 10 || loc.SQLLine != 1 || loc.SQLColumn != 8 {
		t.Fatalf("unexpected selection-adjusted location: %#v", loc)
	}
}

func TestBuildSQLiteSyntaxErrorHint(t *testing.T) {
	sqlText := "SELECT name, type FROM  WHERE type IN ('table','view') ORDER BY type, name;"
	data := buildWebSQLErrorData("sqlite", sqlText, `执行查询失败: SQL logic error: near "WHERE": syntax error (1)`, map[string]interface{}{}, time.Now())
	location, ok := data["error_location"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected error_location map, got %#v", data["error_location"])
	}
	if location["direction"] != "before" || location["direction_label"] != "重点检查：标记前方" {
		t.Fatalf("unexpected direction hint: %#v", location)
	}
	if !strings.Contains(stringValue(location["hint"]), "WHERE") || !strings.Contains(stringValue(location["hint"]), "FROM") {
		t.Fatalf("expected hint to mention WHERE/FROM, got %#v", location["hint"])
	}
	if !strings.Contains(stringValue(location["context_before"]), "FROM") || !strings.Contains(stringValue(location["context_after"]), "type IN") {
		t.Fatalf("unexpected context: %#v", location)
	}
}
