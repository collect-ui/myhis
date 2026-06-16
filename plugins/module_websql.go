package plugins

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	_ "github.com/glebarez/go-sqlite"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	go_ora "github.com/sijms/go-ora/v2"
	"gorm.io/gorm"

	"moon/model/devops"
)

type WebSQLService struct {
	templateService.BaseHandler
}

type webSQLConnectionConfig struct {
	Driver     string
	DSN        string
	Host       string
	Port       string
	Username   string
	Password   string
	Database   string
	Schema     string
	SQLitePath string
}

type webSQLSchemaRequest struct {
	Scope      string
	FolderType string
	ObjectType string
	ObjectName string
	TableName  string
	Query      string
}

type webSQLExecuteRequest struct {
	MaxRows               int
	CursorOffset          int
	PageSize              int
	CommitMode            string
	TransactionTTLSeconds int
	ConnectionID          string
	ConnectionName        string
}

type webSQLPendingTx struct {
	EventID      string
	DB           *sql.DB
	Tx           *sql.Tx
	CreatedAt    time.Time
	ExpiresAt    time.Time
	RowsAffected int64
	LastInsertID int64
	DurationMs   int64
}

const (
	webSQLCommitModeDirect = "direct"
	webSQLCommitModeManual = "manual"

	webSQLCommitStatusPending      = "pending"
	webSQLCommitStatusCommitted    = "committed"
	webSQLCommitStatusRolledBack   = "rolled_back"
	webSQLCommitStatusTimeout      = "timeout"
	webSQLCommitStatusDirectCommit = "direct_committed"
)

var (
	webSQLPendingTxMu     sync.Mutex
	webSQLPendingTxMap    = map[string]*webSQLPendingTx{}
	webSQLCommitEventOnce sync.Once
	webSQLCommitEventErr  error
)

var (
	mysqlSyntaxNearRE    = regexp.MustCompile(`(?is)near '(.+)' at line ([0-9]+)`)
	mysqlAtLineRE        = regexp.MustCompile(`(?i)at line ([0-9]+)`)
	mysqlUnknownColumnRE = regexp.MustCompile(`(?i)unknown column '([^']+)'`)
	mysqlUnknownTableRE  = regexp.MustCompile(`(?i)table '([^']+)' doesn't exist`)
	oracleInvalidIdentRE = regexp.MustCompile(`(?i)ORA-00904:\s*"?([^":]+)"?:\s*invalid identifier`)
	oracleMissingObjRE   = regexp.MustCompile(`(?i)ORA-(00942|04043|00903)`)
	oraclePositionRE     = regexp.MustCompile(`(?i)position:\s*([0-9]+)`)
	oracleSyntaxRE       = regexp.MustCompile(`(?i)ORA-(00900|00905|00907|00911|00917|00923|00933|00936|01756)`)
	sqlTableReferenceRE  = regexp.MustCompile(`(?i)\b(?:from|join|update|into|table)\s+((?:"[^"]+"|[A-Za-z_][A-Za-z0-9_$#]*)(?:\s*\.\s*(?:"[^"]+"|[A-Za-z_][A-Za-z0-9_$#]*))*)`)
	sqliteNearDoubleRE   = regexp.MustCompile(`(?i)near "([^"]*)"`)
	sqliteNearSingleRE   = regexp.MustCompile(`(?i)near '([^']*)'`)
	sqliteNoSuchColumnRE = regexp.MustCompile(`(?i)no such column: ([^\s,)]+)`)
	sqliteNoSuchTableRE  = regexp.MustCompile(`(?i)no such table: ([^\s,)]+)`)
	sqliteSyntaxErrorRE  = regexp.MustCompile(`(?i)syntax error`)
	sqliteUnrecognizedRE = regexp.MustCompile(`(?i)unrecognized token: "([^"]*)"`)
)

type webSQLErrorLocation struct {
	Line           int
	Column         int
	EndLine        int
	EndColumn      int
	SQLLine        int
	SQLColumn      int
	SQLEndLine     int
	SQLEndColumn   int
	Token          string
	Kind           string
	Direction      string
	DirectionLabel string
	Hint           string
	ContextBefore  string
	ContextAfter   string
}

func (s *WebSQLService) Result(template *config.Template, ts *templateService.TemplateService) *common.Result {
	params := template.GetParams()
	operation := strings.ToLower(stringValue(params["operation"]))
	if operation == "" {
		operation = "execute"
	}

	switch operation {
	case "commit", "rollback", "list_events", "pending_events":
		data, err := s.handleWebSQLTransactionOperation(operation, params)
		if err != nil {
			return common.NotOk(err.Error())
		}
		return common.Ok(data, "操作成功")
	}

	cfg := webSQLConnectionConfig{
		Driver:     normalizeWebSQLDriver(stringValue(params["driver"])),
		DSN:        stringValue(params["dsn"]),
		Host:       stringValue(params["host"]),
		Port:       stringValue(params["port"]),
		Username:   stringValue(params["username"]),
		Password:   stringValue(params["password"]),
		Database:   stringValue(params["database"]),
		Schema:     stringValue(params["schema"]),
		SQLitePath: stringValue(params["sqlite_path"]),
	}
	if cfg.Driver == "" {
		cfg.Driver = "mysql"
	}

	startAt := time.Now()
	db, driverName, err := openWebSQLDB(cfg)
	if err != nil {
		return common.NotOk(err.Error())
	}
	retainDB := false
	defer func() {
		if !retainDB {
			db.Close()
		}
	}()

	timeout := webSQLIntValue(params["timeout_seconds"], 30)
	if timeout <= 0 {
		timeout = 30
	}
	if timeout > 120 {
		timeout = 120
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return common.NotOk(fmt.Sprintf("数据库连接失败: %s", err.Error()))
	}

	switch operation {
	case "ping":
		return common.Ok(map[string]interface{}{
			"driver":      driverName,
			"database":    cfg.Database,
			"schema":      cfg.Schema,
			"duration_ms": time.Since(startAt).Milliseconds(),
		}, "连接成功")
	case "schema":
		schemaName := cfg.Database
		if driverName == "oracle" {
			schemaName = cfg.Schema
		}
		schemaReq := webSQLSchemaRequest{
			Scope:      strings.ToLower(strings.TrimSpace(stringValue(params["schema_scope"]))),
			FolderType: strings.ToLower(strings.TrimSpace(stringValue(params["folder_type"]))),
			ObjectType: strings.ToLower(strings.TrimSpace(stringValue(params["object_type"]))),
			ObjectName: strings.TrimSpace(stringValue(params["object_name"])),
			TableName:  strings.TrimSpace(stringValue(params["table_name"])),
			Query:      strings.TrimSpace(stringValue(params["query"])),
		}
		data, err := loadWebSQLSchema(ctx, db, driverName, schemaName, schemaReq)
		if err != nil {
			return common.NotOk(err.Error())
		}
		data["driver"] = driverName
		data["duration_ms"] = time.Since(startAt).Milliseconds()
		return common.Ok(data, "结构加载成功")
	case "ddl", "object_detail":
		schemaName := cfg.Database
		if driverName == "oracle" {
			schemaName = cfg.Schema
		}
		objectType := strings.ToLower(strings.TrimSpace(stringValue(params["object_type"])))
		objectName := strings.TrimSpace(stringValue(params["object_name"]))
		tableName := strings.TrimSpace(stringValue(params["table_name"]))
		data, err := loadWebSQLObjectDetail(ctx, db, driverName, schemaName, objectType, objectName, tableName)
		if err != nil {
			return common.NotOk(err.Error())
		}
		data["driver"] = driverName
		data["duration_ms"] = time.Since(startAt).Milliseconds()
		return common.Ok(data, "对象结构加载成功")
	case "execute":
		sqlText := strings.TrimSpace(stringValue(params["sql"]))
		if sqlText == "" {
			return common.NotOk("sql 不能为空")
		}
		maxRows := webSQLIntValue(params["max_rows"], 500)
		if maxRows <= 0 {
			maxRows = 500
		}
		if maxRows > 5000 {
			maxRows = 5000
		}
		pageSize := webSQLIntValue(params["page_size"], 0)
		if pageSize < 0 {
			pageSize = 0
		}
		if pageSize > 500 {
			pageSize = 500
		}
		execReq := webSQLExecuteRequest{
			MaxRows:               maxRows,
			CursorOffset:          webSQLIntValue(params["cursor_offset"], 0),
			PageSize:              pageSize,
			CommitMode:            stringValue(params["commit_mode"]),
			TransactionTTLSeconds: webSQLIntValue(params["transaction_ttl_seconds"], 120),
			ConnectionID:          stringValue(params["connection_id"]),
			ConnectionName:        stringValue(params["connection_name"]),
		}
		data, keepDB, err := s.executeWebSQL(ctx, db, driverName, sqlText, cfg, execReq, startAt)
		if err != nil {
			return webSQLNotOkWithData(err.Error(), buildWebSQLErrorData(driverName, sqlText, err.Error(), params, startAt))
		}
		retainDB = keepDB
		return common.Ok(data, "执行成功")
	default:
		return common.NotOk("operation 仅支持 execute/schema/ping/ddl/object_detail/commit/rollback/list_events")
	}
}

func normalizeWebSQLDriver(raw string) string {
	driver := strings.ToLower(strings.TrimSpace(raw))
	switch driver {
	case "mysql":
		return "mysql"
	case "oracle", "ora":
		return "oracle"
	case "sqlite", "sqlite3":
		return "sqlite"
	default:
		return driver
	}
}

func openWebSQLDB(cfg webSQLConnectionConfig) (*sql.DB, string, error) {
	driver := normalizeWebSQLDriver(cfg.Driver)
	switch driver {
	case "mysql":
		dsn := strings.TrimSpace(cfg.DSN)
		databaseName := strings.TrimSpace(cfg.Database)
		if dsn == "" {
			host := strings.TrimSpace(cfg.Host)
			if host == "" {
				host = "127.0.0.1"
			}
			port := strings.TrimSpace(cfg.Port)
			if port == "" {
				port = "3306"
			}
			mysqlCfg := mysqlDriver.NewConfig()
			mysqlCfg.Net = "tcp"
			mysqlCfg.Addr = net.JoinHostPort(host, port)
			mysqlCfg.User = cfg.Username
			mysqlCfg.Passwd = cfg.Password
			mysqlCfg.DBName = databaseName
			mysqlCfg.ParseTime = true
			mysqlCfg.Params = map[string]string{
				"charset":      "utf8mb4",
				"loc":          "Local",
				"timeout":      "10s",
				"readTimeout":  "30s",
				"writeTimeout": "30s",
			}
			dsn = mysqlCfg.FormatDSN()
		} else if databaseName != "" {
			mysqlCfg, err := mysqlDriver.ParseDSN(dsn)
			if err != nil {
				return nil, "", err
			}
			mysqlCfg.DBName = databaseName
			dsn = mysqlCfg.FormatDSN()
		}
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return nil, "", err
		}
		db.SetMaxOpenConns(2)
		db.SetMaxIdleConns(1)
		db.SetConnMaxLifetime(3 * time.Minute)
		return db, "mysql", nil
	case "oracle":
		dsn := strings.TrimSpace(cfg.DSN)
		if dsn == "" {
			host := strings.TrimSpace(cfg.Host)
			if host == "" {
				host = "127.0.0.1"
			}
			port := strings.TrimSpace(cfg.Port)
			if port == "" {
				port = "1521"
			}
			portNumber, err := strconv.Atoi(port)
			if err != nil {
				return nil, "", fmt.Errorf("oracle port 必须是数字: %s", port)
			}
			serviceName := strings.TrimSpace(cfg.Database)
			if serviceName == "" {
				return nil, "", fmt.Errorf("oracle service name 或 dsn 不能为空")
			}
			dsn = go_ora.BuildUrl(host, portNumber, serviceName, cfg.Username, cfg.Password, map[string]string{
				"CONNECTION TIMEOUT": "10",
				"READ TIMEOUT":       "30",
			})
		}
		db, err := sql.Open("oracle", dsn)
		if err != nil {
			return nil, "", err
		}
		db.SetMaxOpenConns(2)
		db.SetMaxIdleConns(1)
		db.SetConnMaxLifetime(3 * time.Minute)
		return db, "oracle", nil
	case "sqlite":
		dsn := strings.TrimSpace(cfg.DSN)
		if dsn == "" {
			dsn = strings.TrimSpace(cfg.SQLitePath)
		}
		if dsn == "" {
			return nil, "", fmt.Errorf("sqlite_path 或 dsn 不能为空")
		}
		db, err := sql.Open("sqlite", dsn)
		if err != nil {
			return nil, "", err
		}
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		db.SetConnMaxLifetime(0)
		return db, "sqlite", nil
	default:
		return nil, "", fmt.Errorf("暂不支持数据库类型: %s", cfg.Driver)
	}
}

func webSQLNotOkWithData(msg string, data map[string]interface{}) *common.Result {
	if data == nil {
		return common.NotOk(msg)
	}
	return &common.Result{
		Success: false,
		Code:    common.UnSuccessValue,
		Msg:     msg,
		Data:    data,
	}
}

func buildWebSQLErrorData(driverName string, sqlText string, errText string, params map[string]interface{}, startAt time.Time) map[string]interface{} {
	loc := detectWebSQLErrorLocation(driverName, sqlText, errText)
	enrichWebSQLErrorLocation(sqlText, &loc)
	applyWebSQLSelectionOffset(&loc, params)
	location := webSQLErrorLocationMap(driverName, errText, loc)
	marker := webSQLErrorMarker(errText, loc)
	return map[string]interface{}{
		"driver":         driverName,
		"statement_type": "ERROR",
		"columns":        []map[string]interface{}{},
		"rows":           []map[string]interface{}{},
		"row_count":      0,
		"duration_ms":    time.Since(startAt).Milliseconds(),
		"executed_at":    time.Now().Format(time.RFC3339Nano),
		"error":          errText,
		"error_location": location,
		"markers":        []map[string]interface{}{marker},
	}
}

func detectWebSQLErrorLocation(driverName string, sqlText string, errText string) webSQLErrorLocation {
	driver := normalizeWebSQLDriver(driverName)
	switch driver {
	case "mysql":
		if loc, ok := detectMySQLErrorLocation(sqlText, errText); ok {
			return loc
		}
	case "oracle":
		if loc, ok := detectOracleErrorLocation(sqlText, errText); ok {
			return loc
		}
	case "sqlite":
		if loc, ok := detectSQLiteErrorLocation(sqlText, errText); ok {
			return loc
		}
	}
	return fallbackWebSQLErrorLocation(sqlText, 1, "sql_error")
}

func detectMySQLErrorLocation(sqlText string, errText string) (webSQLErrorLocation, bool) {
	if match := mysqlUnknownColumnRE.FindStringSubmatch(errText); len(match) >= 2 {
		return locateWebSQLNeedle(sqlText, match[1], 0, "unknown_column")
	}
	if match := mysqlUnknownTableRE.FindStringSubmatch(errText); len(match) >= 2 {
		tableName := trimSQLQualifier(match[1])
		if loc, ok := locateWebSQLNeedle(sqlText, tableName, 0, "unknown_table"); ok {
			return loc, true
		}
		return locateWebSQLNeedle(sqlText, match[1], 0, "unknown_table")
	}
	if match := mysqlSyntaxNearRE.FindStringSubmatch(errText); len(match) >= 3 {
		lineNo, _ := strconv.Atoi(match[2])
		if loc, ok := locateWebSQLNeedle(sqlText, match[1], lineNo, "syntax_error"); ok {
			return loc, true
		}
		return fallbackWebSQLErrorLocation(sqlText, lineNo, "syntax_error"), true
	}
	if match := mysqlAtLineRE.FindStringSubmatch(errText); len(match) >= 2 {
		lineNo, _ := strconv.Atoi(match[1])
		return fallbackWebSQLErrorLocation(sqlText, lineNo, "syntax_error"), true
	}
	return webSQLErrorLocation{}, false
}

func detectOracleErrorLocation(sqlText string, errText string) (webSQLErrorLocation, bool) {
	if match := oracleInvalidIdentRE.FindStringSubmatch(errText); len(match) >= 2 {
		return locateWebSQLNeedle(sqlText, match[1], 0, "unknown_column")
	}
	if oracleMissingObjRE.MatchString(errText) {
		if loc, ok := locateWebSQLTableReference(sqlText); ok {
			loc.Kind = "unknown_table"
			return loc, true
		}
		return fallbackWebSQLErrorLocation(sqlText, 1, "unknown_table"), true
	}
	if oracleSyntaxRE.MatchString(errText) {
		if loc, ok := locateOracleErrorPosition(sqlText, errText, "syntax_error"); ok {
			return loc, true
		}
		return fallbackWebSQLErrorLocation(sqlText, 1, "syntax_error"), true
	}
	return webSQLErrorLocation{}, false
}

func locateOracleErrorPosition(sqlText string, errText string, kind string) (webSQLErrorLocation, bool) {
	match := oraclePositionRE.FindStringSubmatch(errText)
	if len(match) < 2 {
		return webSQLErrorLocation{}, false
	}
	pos, err := strconv.Atoi(match[1])
	if err != nil || pos <= 0 {
		return webSQLErrorLocation{}, false
	}
	offset := pos - 1
	if offset < 0 {
		offset = 0
	}
	if offset > len(sqlText) {
		offset = len(sqlText)
	}
	line, column := sqlOffsetToLineColumn(sqlText, offset)
	endLine, endColumn := sqlOffsetToLineColumn(sqlText, minWebSQLInt(offset+1, len(sqlText)))
	token := webSQLTokenAtOffset(sqlText, offset)
	if token == "" {
		token = "标记位置"
	}
	if endLine == line && endColumn <= column {
		endColumn = column + 1
	}
	return webSQLErrorLocation{
		Line:         line,
		Column:       column,
		EndLine:      endLine,
		EndColumn:    endColumn,
		SQLLine:      line,
		SQLColumn:    column,
		SQLEndLine:   endLine,
		SQLEndColumn: endColumn,
		Token:        token,
		Kind:         kind,
	}, true
}

func webSQLTokenAtOffset(sqlText string, offset int) string {
	if sqlText == "" {
		return ""
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(sqlText) {
		offset = len(sqlText) - 1
	}
	isTokenByte := func(ch byte) bool {
		return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '$' || ch == '#' || ch == '"' || ch == '`'
	}
	start := offset
	for start > 0 && isTokenByte(sqlText[start-1]) {
		start--
	}
	end := offset
	for end < len(sqlText) && isTokenByte(sqlText[end]) {
		end++
	}
	return strings.Trim(strings.TrimSpace(sqlText[start:end]), "\"`")
}

func locateWebSQLTableReference(sqlText string) (webSQLErrorLocation, bool) {
	match := sqlTableReferenceRE.FindStringSubmatchIndex(sqlText)
	if len(match) < 4 || match[2] < 0 || match[3] <= match[2] {
		return webSQLErrorLocation{}, false
	}
	token := strings.TrimSpace(sqlText[match[2]:match[3]])
	parts := webSQLLocationCandidates(token)
	for _, part := range parts {
		if loc, ok := locateWebSQLNeedle(sqlText, part, 0, "unknown_table"); ok {
			return loc, true
		}
	}
	return webSQLErrorLocation{}, false
}

func detectSQLiteErrorLocation(sqlText string, errText string) (webSQLErrorLocation, bool) {
	if match := sqliteNoSuchColumnRE.FindStringSubmatch(errText); len(match) >= 2 {
		return locateWebSQLNeedle(sqlText, match[1], 0, "unknown_column")
	}
	if match := sqliteNoSuchTableRE.FindStringSubmatch(errText); len(match) >= 2 {
		return locateWebSQLNeedle(sqlText, trimSQLQualifier(match[1]), 0, "unknown_table")
	}
	if match := sqliteUnrecognizedRE.FindStringSubmatch(errText); len(match) >= 2 {
		return locateWebSQLNeedle(sqlText, match[1], 0, "syntax_error")
	}
	if match := sqliteNearDoubleRE.FindStringSubmatch(errText); len(match) >= 2 {
		if strings.TrimSpace(match[1]) == "" {
			return fallbackWebSQLErrorLocation(sqlText, 1, "syntax_error"), true
		}
		return locateWebSQLNeedle(sqlText, match[1], 0, "syntax_error")
	}
	if match := sqliteNearSingleRE.FindStringSubmatch(errText); len(match) >= 2 {
		if strings.TrimSpace(match[1]) == "" {
			return fallbackWebSQLErrorLocation(sqlText, 1, "syntax_error"), true
		}
		return locateWebSQLNeedle(sqlText, match[1], 0, "syntax_error")
	}
	if sqliteSyntaxErrorRE.MatchString(errText) {
		return fallbackWebSQLErrorLocation(sqlText, 1, "syntax_error"), true
	}
	return webSQLErrorLocation{}, false
}

func locateWebSQLNeedle(sqlText string, rawNeedle string, preferredLine int, kind string) (webSQLErrorLocation, bool) {
	candidates := webSQLLocationCandidates(rawNeedle)
	for _, candidate := range candidates {
		offset, matched := indexSQLTextFold(sqlText, candidate, preferredLine)
		if offset < 0 {
			continue
		}
		line, column := sqlOffsetToLineColumn(sqlText, offset)
		endLine, endColumn := sqlOffsetToLineColumn(sqlText, offset+len(matched))
		if endLine == line && endColumn <= column {
			endColumn = column + 1
		}
		return webSQLErrorLocation{
			Line:         line,
			Column:       column,
			EndLine:      endLine,
			EndColumn:    endColumn,
			SQLLine:      line,
			SQLColumn:    column,
			SQLEndLine:   endLine,
			SQLEndColumn: endColumn,
			Token:        candidate,
			Kind:         kind,
		}, true
	}
	if preferredLine > 0 {
		return fallbackWebSQLErrorLocation(sqlText, preferredLine, kind), true
	}
	return webSQLErrorLocation{}, false
}

func webSQLLocationCandidates(rawNeedle string) []string {
	seen := map[string]bool{}
	candidates := make([]string, 0, 4)
	add := func(value string) {
		value = strings.TrimSpace(value)
		value = strings.Trim(value, "`\"'")
		if value == "" || seen[strings.ToLower(value)] {
			return
		}
		seen[strings.ToLower(value)] = true
		candidates = append(candidates, value)
	}
	add(rawNeedle)
	add(trimSQLQualifier(rawNeedle))
	add(firstWebSQLNeedleWord(rawNeedle))
	return candidates
}

func trimSQLQualifier(value string) string {
	text := strings.TrimSpace(value)
	text = strings.Trim(text, "`\"'")
	if idx := strings.LastIndex(text, "."); idx >= 0 && idx < len(text)-1 {
		return strings.Trim(strings.TrimSpace(text[idx+1:]), "`\"'")
	}
	return text
}

func firstWebSQLNeedleWord(value string) string {
	text := strings.TrimSpace(value)
	text = strings.Trim(text, "`\"'")
	start := -1
	end := -1
	for i, r := range text {
		ok := r == '_' || r == '$' || r == '.' || (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
		if ok && start < 0 {
			start = i
		}
		if start >= 0 && !ok {
			end = i
			break
		}
	}
	if start < 0 {
		return ""
	}
	if end < 0 {
		end = len(text)
	}
	return trimSQLQualifier(text[start:end])
}

func indexSQLTextFold(sqlText string, needle string, preferredLine int) (int, string) {
	if strings.TrimSpace(needle) == "" {
		return -1, ""
	}
	startOffset := 0
	if preferredLine > 1 {
		startOffset = sqlLineStartOffset(sqlText, preferredLine)
	}
	if idx := strings.Index(sqlText[startOffset:], needle); idx >= 0 {
		return startOffset + idx, needle
	}
	lowerHaystack := strings.ToLower(sqlText[startOffset:])
	lowerNeedle := strings.ToLower(needle)
	if idx := strings.Index(lowerHaystack, lowerNeedle); idx >= 0 {
		return startOffset + idx, sqlText[startOffset+idx : startOffset+idx+len(needle)]
	}
	if startOffset > 0 {
		if idx := strings.Index(sqlText, needle); idx >= 0 {
			return idx, needle
		}
		lowerAll := strings.ToLower(sqlText)
		if idx := strings.Index(lowerAll, lowerNeedle); idx >= 0 {
			return idx, sqlText[idx : idx+len(needle)]
		}
	}
	return -1, ""
}

func sqlLineStartOffset(sqlText string, lineNo int) int {
	if lineNo <= 1 {
		return 0
	}
	line := 1
	for i, r := range sqlText {
		if line == lineNo {
			return i
		}
		if r == '\n' {
			line++
		}
	}
	return len(sqlText)
}

func sqlOffsetToLineColumn(sqlText string, offset int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(sqlText) {
		offset = len(sqlText)
	}
	line := 1
	column := 1
	for i, r := range sqlText {
		if i >= offset {
			break
		}
		if r == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	return line, column
}

func fallbackWebSQLErrorLocation(sqlText string, preferredLine int, kind string) webSQLErrorLocation {
	lines := strings.Split(sqlText, "\n")
	lineNo := preferredLine
	if lineNo <= 0 {
		lineNo = 1
	}
	if lineNo > len(lines) {
		lineNo = len(lines)
	}
	if lineNo <= 0 {
		lineNo = 1
	}
	lineText := ""
	if len(lines) > 0 {
		lineText = lines[lineNo-1]
	}
	column := 1
	for _, r := range lineText {
		if r != ' ' && r != '\t' && r != '\r' {
			break
		}
		column++
	}
	endColumn := column + 1
	if lineText == "" {
		endColumn = column
	}
	return webSQLErrorLocation{
		Line:         lineNo,
		Column:       column,
		EndLine:      lineNo,
		EndColumn:    endColumn,
		SQLLine:      lineNo,
		SQLColumn:    column,
		SQLEndLine:   lineNo,
		SQLEndColumn: endColumn,
		Kind:         kind,
	}
}

func applyWebSQLSelectionOffset(loc *webSQLErrorLocation, params map[string]interface{}) {
	if loc == nil {
		return
	}
	startLine := webSQLIntValue(params["selection_start_line"], 1)
	startColumn := webSQLIntValue(params["selection_start_column"], 1)
	if startLine < 1 {
		startLine = 1
	}
	if startColumn < 1 {
		startColumn = 1
	}
	loc.SQLLine = loc.Line
	loc.SQLColumn = loc.Column
	loc.SQLEndLine = loc.EndLine
	loc.SQLEndColumn = loc.EndColumn
	if startLine > 1 {
		loc.Line += startLine - 1
		loc.EndLine += startLine - 1
	}
	if startColumn > 1 {
		if loc.SQLLine == 1 {
			loc.Column += startColumn - 1
		}
		if loc.SQLEndLine == 1 {
			loc.EndColumn += startColumn - 1
		}
	}
}

func enrichWebSQLErrorLocation(sqlText string, loc *webSQLErrorLocation) {
	if loc == nil {
		return
	}
	before, after := webSQLErrorContext(sqlText, *loc)
	loc.ContextBefore = before
	loc.ContextAfter = after
	loc.Direction, loc.DirectionLabel, loc.Hint = webSQLErrorHint(*loc)
}

func webSQLErrorContext(sqlText string, loc webSQLErrorLocation) (string, string) {
	line := loc.SQLLine
	column := loc.SQLColumn
	endLine := loc.SQLEndLine
	endColumn := loc.SQLEndColumn
	if line <= 0 {
		line = loc.Line
	}
	if column <= 0 {
		column = loc.Column
	}
	if endLine <= 0 {
		endLine = loc.EndLine
	}
	if endColumn <= 0 {
		endColumn = loc.EndColumn
	}
	startOffset := sqlLineColumnToOffset(sqlText, line, column)
	endOffset := sqlLineColumnToOffset(sqlText, endLine, endColumn)
	if endOffset <= startOffset && loc.Token != "" {
		endOffset = startOffset + len(loc.Token)
		if endOffset > len(sqlText) {
			endOffset = len(sqlText)
		}
	}
	if endOffset < startOffset {
		endOffset = startOffset
	}
	beforeStart := startOffset - 80
	if beforeStart < 0 {
		beforeStart = 0
	}
	afterEnd := endOffset + 80
	if afterEnd > len(sqlText) {
		afterEnd = len(sqlText)
	}
	return compactSQLSnippet(sqlText[beforeStart:startOffset]), compactSQLSnippet(sqlText[endOffset:afterEnd])
}

func sqlLineColumnToOffset(sqlText string, lineNo int, columnNo int) int {
	if lineNo <= 1 && columnNo <= 1 {
		return 0
	}
	if lineNo <= 0 {
		lineNo = 1
	}
	if columnNo <= 0 {
		columnNo = 1
	}
	line := 1
	column := 1
	for i, r := range sqlText {
		if line == lineNo && column == columnNo {
			return i
		}
		if r == '\n' {
			line++
			column = 1
			continue
		}
		column++
	}
	return len(sqlText)
}

func compactSQLSnippet(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func webSQLErrorHint(loc webSQLErrorLocation) (string, string, string) {
	token := strings.TrimSpace(loc.Token)
	tokenLabel := token
	if tokenLabel == "" {
		tokenLabel = "标记位置"
	}
	switch loc.Kind {
	case "unknown_column":
		return "at", "问题点：标记字段", fmt.Sprintf("字段 %s 不存在；请检查字段名是否写错，或检查它前后的表名、别名是否正确。", quoteWebSQLHintToken(tokenLabel))
	case "unknown_table":
		return "at", "问题点：标记表名", fmt.Sprintf("表或视图 %s 不存在；请检查表名、库名/schema 和连接的数据库是否正确。", quoteWebSQLHintToken(tokenLabel))
	case "syntax_error":
		beforeWord := lastWebSQLContextWord(loc.ContextBefore)
		if isWebSQLObjectIntroducer(beforeWord) {
			return "before", "重点检查：标记前方", fmt.Sprintf("数据库在 %s 附近报错，但它前面的 %s 后面缺少对象或语法不完整；请先检查标记前方。", quoteWebSQLHintToken(tokenLabel), beforeWord)
		}
		if isWebSQLOperatorLike(beforeWord) {
			return "after", "重点检查：标记后方", fmt.Sprintf("标记前方以 %s 结尾，后面通常还需要表达式或字段；请检查标记后方是否缺内容。", beforeWord)
		}
		return "near", "重点检查：标记附近", fmt.Sprintf("语法错误位于 %s 附近；请同时检查标记前后的关键字、逗号、括号和表达式是否完整。", quoteWebSQLHintToken(tokenLabel))
	default:
		return "near", "重点检查：标记附近", fmt.Sprintf("错误位于 %s 附近；请结合数据库错误信息检查标记前后 SQL。", quoteWebSQLHintToken(tokenLabel))
	}
}

func quoteWebSQLHintToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" || token == "标记位置" {
		return token
	}
	return "「" + token + "」"
}

func lastWebSQLContextWord(value string) string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return !(r == '_' || r == '$' || r == '.' || (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z'))
	})
	for i := len(fields) - 1; i >= 0; i-- {
		field := strings.Trim(fields[i], "`\"[]")
		if field != "" {
			return strings.ToUpper(field)
		}
	}
	return ""
}

func isWebSQLObjectIntroducer(word string) bool {
	switch strings.ToUpper(strings.TrimSpace(word)) {
	case "FROM", "JOIN", "INTO", "UPDATE", "TABLE", "VIEW", "DESC", "DESCRIBE":
		return true
	default:
		return false
	}
}

func isWebSQLOperatorLike(word string) bool {
	switch strings.ToUpper(strings.TrimSpace(word)) {
	case "AND", "OR", "ON", "WHERE", "SET", "=", ">", "<", ">=", "<=", "<>", "!=":
		return true
	default:
		return false
	}
}

func webSQLErrorLocationMap(driverName string, errText string, loc webSQLErrorLocation) map[string]interface{} {
	return map[string]interface{}{
		"driver":          driverName,
		"kind":            loc.Kind,
		"token":           loc.Token,
		"direction":       loc.Direction,
		"direction_label": loc.DirectionLabel,
		"hint":            loc.Hint,
		"context_before":  loc.ContextBefore,
		"context_after":   loc.ContextAfter,
		"line":            loc.Line,
		"column":          loc.Column,
		"end_line":        loc.EndLine,
		"end_column":      loc.EndColumn,
		"sql_line":        loc.SQLLine,
		"sql_column":      loc.SQLColumn,
		"sql_end_line":    loc.SQLEndLine,
		"sql_end_column":  loc.SQLEndColumn,
		"message":         errText,
	}
}

func webSQLErrorMarker(errText string, loc webSQLErrorLocation) map[string]interface{} {
	message := errText
	if loc.Line > 0 && loc.Column > 0 {
		message = fmt.Sprintf("%s（行 %d，列 %d）", errText, loc.Line, loc.Column)
	}
	if loc.Hint != "" {
		message = fmt.Sprintf("%s。%s：%s", message, loc.DirectionLabel, loc.Hint)
	}
	return map[string]interface{}{
		"severity":        "error",
		"source":          "websql",
		"message":         message,
		"raw_message":     errText,
		"token":           loc.Token,
		"direction":       loc.Direction,
		"direction_label": loc.DirectionLabel,
		"hint":            loc.Hint,
		"startLineNumber": loc.Line,
		"startColumn":     loc.Column,
		"endLineNumber":   loc.EndLine,
		"endColumn":       loc.EndColumn,
	}
}

func (s *WebSQLService) executeWebSQL(ctx context.Context, db *sql.DB, driverName string, sqlText string, cfg webSQLConnectionConfig, req webSQLExecuteRequest, startAt time.Time) (map[string]interface{}, bool, error) {
	sqlText = normalizeWebSQLExecutionSQL(driverName, sqlText)
	statementType := firstSQLKeyword(sqlText)
	if isWebSQLQuery(statementType) {
		rows, err := db.QueryContext(ctx, sqlText)
		if err != nil {
			return nil, false, fmt.Errorf("执行查询失败: %s", err.Error())
		}
		defer rows.Close()
		pageSize := req.PageSize
		if pageSize <= 0 {
			pageSize = req.MaxRows
		}
		if pageSize <= 0 {
			pageSize = 50
		}
		if pageSize > req.MaxRows && req.MaxRows > 0 {
			pageSize = req.MaxRows
		}
		if pageSize > 500 {
			pageSize = 500
		}
		cursorOffset := req.CursorOffset
		if cursorOffset < 0 {
			cursorOffset = 0
		}
		columns, rowList, truncated, skipped, hasNext, err := readWebSQLRowsPage(rows, cursorOffset, pageSize)
		if err != nil {
			return nil, false, err
		}
		nextCursor := cursorOffset + len(rowList)
		if !hasNext {
			nextCursor = -1
		}
		prevCursor := cursorOffset - pageSize
		if prevCursor < 0 {
			prevCursor = 0
		}
		return map[string]interface{}{
			"driver":         driverName,
			"statement_type": statementType,
			"columns":        columns,
			"rows":           rowList,
			"row_count":      len(rowList),
			"truncated":      truncated,
			"max_rows":       req.MaxRows,
			"cursor_offset":  cursorOffset,
			"page_size":      pageSize,
			"skipped_rows":   skipped,
			"has_next":       hasNext,
			"has_prev":       cursorOffset > 0,
			"next_cursor":    nextCursor,
			"prev_cursor":    prevCursor,
			"duration_ms":    time.Since(startAt).Milliseconds(),
			"executed_at":    time.Now().Format(time.RFC3339Nano),
		}, false, nil
	}

	commitMode := normalizeWebSQLCommitMode(req.CommitMode)
	if commitMode == webSQLCommitModeManual {
		return s.executeWebSQLManualMutation(ctx, db, driverName, sqlText, statementType, cfg, req, startAt)
	}

	result, err := db.ExecContext(ctx, sqlText)
	if err != nil {
		return nil, false, fmt.Errorf("执行SQL失败: %s", err.Error())
	}
	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()
	data := map[string]interface{}{
		"driver":         driverName,
		"statement_type": statementType,
		"columns":        []map[string]interface{}{},
		"rows":           []map[string]interface{}{},
		"row_count":      0,
		"rows_affected":  rowsAffected,
		"last_insert_id": lastInsertID,
		"commit_mode":    webSQLCommitModeDirect,
		"commit_status":  webSQLCommitStatusDirectCommit,
		"duration_ms":    time.Since(startAt).Milliseconds(),
		"executed_at":    time.Now().Format(time.RFC3339Nano),
	}
	eventID, err := s.saveWebSQLCommitEvent(&devops.WebSQLCommitEvent{
		WebSQLCommitEventID: "websql_event_" + uuid.NewString(),
		Status:              webSQLCommitStatusDirectCommit,
		CommitMode:          webSQLCommitModeDirect,
		Driver:              driverName,
		DatabaseName:        cfg.Database,
		ConnectionID:        req.ConnectionID,
		ConnectionName:      req.ConnectionName,
		StatementType:       statementType,
		SQLText:             truncateLogText(sqlText, 500000),
		RowsAffected:        rowsAffected,
		LastInsertID:        lastInsertID,
		CreateTime:          time.Now().Format(time.RFC3339Nano),
		FinishTime:          time.Now().Format(time.RFC3339Nano),
		DurationMs:          time.Since(startAt).Milliseconds(),
	})
	if err != nil {
		data["event_log_error"] = err.Error()
	} else {
		data["event_id"] = eventID
	}
	return data, false, nil
}

func (s *WebSQLService) executeWebSQLManualMutation(ctx context.Context, db *sql.DB, driverName string, sqlText string, statementType string, cfg webSQLConnectionConfig, req webSQLExecuteRequest, startAt time.Time) (map[string]interface{}, bool, error) {
	ttlSeconds := req.TransactionTTLSeconds
	if ttlSeconds <= 0 {
		ttlSeconds = 120
	}
	if ttlSeconds > 600 {
		ttlSeconds = 600
	}
	now := time.Now()
	expireAt := now.Add(time.Duration(ttlSeconds) * time.Second)
	event := &devops.WebSQLCommitEvent{
		WebSQLCommitEventID: "websql_event_" + uuid.NewString(),
		Status:              webSQLCommitStatusPending,
		CommitMode:          webSQLCommitModeManual,
		Driver:              driverName,
		DatabaseName:        cfg.Database,
		ConnectionID:        req.ConnectionID,
		ConnectionName:      req.ConnectionName,
		StatementType:       statementType,
		SQLText:             truncateLogText(sqlText, 500000),
		CreateTime:          now.Format(time.RFC3339Nano),
		ExpireTime:          expireAt.Format(time.RFC3339Nano),
		DurationMs:          time.Since(startAt).Milliseconds(),
	}
	if _, err := s.saveWebSQLCommitEvent(event); err != nil {
		return nil, false, fmt.Errorf("SQL已回滚，提交事件写入失败: %s", err.Error())
	}

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		_ = s.updateWebSQLCommitEventStatus(event.WebSQLCommitEventID, webSQLCommitStatusRolledBack, err.Error())
		return nil, false, fmt.Errorf("开启事务失败: %s", err.Error())
	}
	result, err := tx.ExecContext(ctx, sqlText)
	if err != nil {
		_ = tx.Rollback()
		_ = s.updateWebSQLCommitEventStatus(event.WebSQLCommitEventID, webSQLCommitStatusRolledBack, err.Error())
		return nil, false, fmt.Errorf("执行SQL失败: %s", err.Error())
	}
	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()
	durationMs := time.Since(startAt).Milliseconds()
	registerWebSQLPendingTx(&webSQLPendingTx{
		EventID:      event.WebSQLCommitEventID,
		DB:           db,
		Tx:           tx,
		CreatedAt:    now,
		ExpiresAt:    expireAt,
		RowsAffected: rowsAffected,
		LastInsertID: lastInsertID,
		DurationMs:   durationMs,
	}, s.GetGormDb())

	return map[string]interface{}{
		"driver":              driverName,
		"statement_type":      statementType,
		"columns":             []map[string]interface{}{},
		"rows":                []map[string]interface{}{},
		"row_count":           0,
		"rows_affected":       rowsAffected,
		"last_insert_id":      lastInsertID,
		"commit_mode":         webSQLCommitModeManual,
		"commit_status":       webSQLCommitStatusPending,
		"pending_commit":      true,
		"event_id":            event.WebSQLCommitEventID,
		"expire_at":           event.ExpireTime,
		"ttl_seconds":         ttlSeconds,
		"duration_ms":         durationMs,
		"executed_at":         now.Format(time.RFC3339Nano),
		"commit_warning_text": "更新已在事务中执行但未提交，请在超时前手动提交或回滚。",
	}, true, nil
}

func normalizeWebSQLCommitMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "manual", "pending", "transaction", "tx":
		return webSQLCommitModeManual
	default:
		return webSQLCommitModeDirect
	}
}

func registerWebSQLPendingTx(entry *webSQLPendingTx, gormDB *gorm.DB) {
	if entry == nil || entry.Tx == nil || strings.TrimSpace(entry.EventID) == "" {
		return
	}
	webSQLPendingTxMu.Lock()
	webSQLPendingTxMap[entry.EventID] = entry
	webSQLPendingTxMu.Unlock()

	delay := time.Until(entry.ExpiresAt)
	if delay < 0 {
		delay = 0
	}
	time.AfterFunc(delay, func() {
		expireWebSQLPendingTx(entry.EventID, gormDB)
	})
}

func expireWebSQLPendingTx(eventID string, gormDB *gorm.DB) {
	webSQLPendingTxMu.Lock()
	entry := webSQLPendingTxMap[eventID]
	if entry == nil {
		webSQLPendingTxMu.Unlock()
		return
	}
	if time.Now().Before(entry.ExpiresAt) {
		webSQLPendingTxMu.Unlock()
		return
	}
	delete(webSQLPendingTxMap, eventID)
	webSQLPendingTxMu.Unlock()

	errText := ""
	if err := entry.Tx.Rollback(); err != nil && !strings.Contains(strings.ToLower(err.Error()), "done") {
		errText = err.Error()
	}
	if entry.DB != nil {
		_ = entry.DB.Close()
	}
	if gormDB != nil {
		updateWebSQLCommitEventStatusWithEntry(gormDB, eventID, webSQLCommitStatusTimeout, errText, entry)
	}
}

func (s *WebSQLService) handleWebSQLTransactionOperation(operation string, params map[string]interface{}) (map[string]interface{}, error) {
	switch operation {
	case "list_events", "pending_events":
		return s.listWebSQLCommitEvents(params)
	case "commit", "rollback":
		eventID := stringValue(params["event_id"])
		if eventID == "" {
			return nil, fmt.Errorf("event_id 不能为空")
		}
		return s.finishWebSQLPendingTx(operation, eventID)
	default:
		return nil, fmt.Errorf("operation 仅支持 commit/rollback/list_events")
	}
}

func (s *WebSQLService) finishWebSQLPendingTx(operation string, eventID string) (map[string]interface{}, error) {
	webSQLPendingTxMu.Lock()
	entry := webSQLPendingTxMap[eventID]
	if entry == nil {
		webSQLPendingTxMu.Unlock()
		return nil, fmt.Errorf("提交事件不存在或已超时: %s", eventID)
	}
	delete(webSQLPendingTxMap, eventID)
	webSQLPendingTxMu.Unlock()

	status := webSQLCommitStatusCommitted
	var err error
	if time.Now().After(entry.ExpiresAt) {
		err = entry.Tx.Rollback()
		status = webSQLCommitStatusTimeout
	} else if operation == "rollback" {
		err = entry.Tx.Rollback()
		status = webSQLCommitStatusRolledBack
	} else {
		err = entry.Tx.Commit()
	}
	if entry.DB != nil {
		_ = entry.DB.Close()
	}

	errText := ""
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "done") {
		errText = err.Error()
	}
	if updateErr := s.updateWebSQLCommitEventStatusWithEntry(eventID, status, errText, entry); updateErr != nil && errText == "" {
		errText = updateErr.Error()
	}
	if errText != "" {
		return nil, fmt.Errorf("事务%s失败: %s", map[bool]string{true: "提交", false: "回滚"}[operation == "commit"], errText)
	}
	return map[string]interface{}{
		"event_id":      eventID,
		"commit_status": status,
		"finished_at":   time.Now().Format(time.RFC3339Nano),
	}, nil
}

func (s *WebSQLService) ensureWebSQLCommitEventTable() error {
	gormDB := s.GetGormDb()
	if gormDB == nil {
		return fmt.Errorf("数据库未初始化")
	}
	webSQLCommitEventOnce.Do(func() {
		webSQLCommitEventErr = gormDB.AutoMigrate(&devops.WebSQLCommitEvent{})
	})
	return webSQLCommitEventErr
}

func (s *WebSQLService) saveWebSQLCommitEvent(entry *devops.WebSQLCommitEvent) (string, error) {
	if entry == nil {
		return "", nil
	}
	if err := s.ensureWebSQLCommitEventTable(); err != nil {
		return entry.WebSQLCommitEventID, err
	}
	if strings.TrimSpace(entry.WebSQLCommitEventID) == "" {
		entry.WebSQLCommitEventID = "websql_event_" + uuid.NewString()
	}
	if strings.TrimSpace(entry.CreateTime) == "" {
		entry.CreateTime = time.Now().Format(time.RFC3339Nano)
	}
	if strings.TrimSpace(entry.Status) == "" {
		entry.Status = webSQLCommitStatusPending
	}
	return entry.WebSQLCommitEventID, s.GetGormDb().Create(entry).Error
}

func (s *WebSQLService) updateWebSQLCommitEventStatus(eventID string, status string, errText string) error {
	return s.updateWebSQLCommitEventStatusWithEntry(eventID, status, errText, nil)
}

func (s *WebSQLService) updateWebSQLCommitEventStatusWithEntry(eventID string, status string, errText string, entry *webSQLPendingTx) error {
	gormDB := s.GetGormDb()
	if gormDB == nil {
		return fmt.Errorf("数据库未初始化")
	}
	if err := s.ensureWebSQLCommitEventTable(); err != nil {
		return err
	}
	return updateWebSQLCommitEventStatusWithEntry(gormDB, eventID, status, errText, entry)
}

func updateWebSQLCommitEventStatus(gormDB *gorm.DB, eventID string, status string, errText string) error {
	return updateWebSQLCommitEventStatusWithEntry(gormDB, eventID, status, errText, nil)
}

func updateWebSQLCommitEventStatusWithEntry(gormDB *gorm.DB, eventID string, status string, errText string, entry *webSQLPendingTx) error {
	if gormDB == nil {
		return fmt.Errorf("数据库未初始化")
	}
	updates := map[string]interface{}{
		"status":      status,
		"finish_time": time.Now().Format(time.RFC3339Nano),
	}
	if strings.TrimSpace(errText) != "" {
		updates["error_text"] = truncateLogText(errText, 4000)
	}
	if entry != nil {
		updates["rows_affected"] = entry.RowsAffected
		updates["last_insert_id"] = entry.LastInsertID
		updates["duration_ms"] = entry.DurationMs
	}
	return gormDB.Model(&devops.WebSQLCommitEvent{}).
		Where("websql_commit_event_id = ?", eventID).
		Updates(updates).Error
}

func (s *WebSQLService) listWebSQLCommitEvents(params map[string]interface{}) (map[string]interface{}, error) {
	if err := s.ensureWebSQLCommitEventTable(); err != nil {
		return nil, err
	}
	limit := webSQLIntValue(params["size"], 20)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	status := strings.TrimSpace(stringValue(params["event_status"]))
	db := s.GetGormDb().Model(&devops.WebSQLCommitEvent{})
	if status != "" {
		db = db.Where("status = ?", status)
	}
	var rows []devops.WebSQLCommitEvent
	if err := db.Order("create_time DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	webSQLPendingTxMu.Lock()
	pendingCount := len(webSQLPendingTxMap)
	webSQLPendingTxMu.Unlock()
	return map[string]interface{}{
		"items":         rows,
		"pending_count": pendingCount,
		"size":          limit,
	}, nil
}

func readWebSQLRows(rows *sql.Rows, maxRows int) ([]map[string]interface{}, []map[string]interface{}, bool, error) {
	columns, rowList, truncated, _, _, err := readWebSQLRowsPage(rows, 0, maxRows)
	return columns, rowList, truncated, err
}

func readWebSQLRowsPage(rows *sql.Rows, offset int, limit int) ([]map[string]interface{}, []map[string]interface{}, bool, int, bool, error) {
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, nil, false, 0, false, fmt.Errorf("读取字段失败: %s", err.Error())
	}
	columnTypes, _ := rows.ColumnTypes()
	columns := make([]map[string]interface{}, 0, len(columnNames))
	for i, name := range columnNames {
		dbType := ""
		nullable := false
		if i < len(columnTypes) && columnTypes[i] != nil {
			dbType = columnTypes[i].DatabaseTypeName()
			nullable, _ = columnTypes[i].Nullable()
		}
		columns = append(columns, map[string]interface{}{
			"field":     name,
			"header":    name,
			"type":      dbType,
			"nullable":  nullable,
			"sortable":  true,
			"filter":    true,
			"resizable": true,
		})
	}

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 50
	}
	rawValues := make([]sql.RawBytes, len(columnNames))
	scanArgs := make([]interface{}, len(columnNames))
	for i := range rawValues {
		scanArgs[i] = &rawValues[i]
	}

	rowList := make([]map[string]interface{}, 0)
	truncated := false
	hasNext := false
	skipped := 0
	for rows.Next() {
		if skipped < offset {
			if err := rows.Scan(scanArgs...); err != nil {
				return columns, rowList, truncated, skipped, hasNext, fmt.Errorf("读取行失败: %s", err.Error())
			}
			skipped++
			continue
		}
		if len(rowList) >= limit {
			truncated = true
			hasNext = true
			break
		}
		if err := rows.Scan(scanArgs...); err != nil {
			return columns, rowList, truncated, skipped, hasNext, fmt.Errorf("读取行失败: %s", err.Error())
		}
		row := make(map[string]interface{}, len(columnNames))
		for i, name := range columnNames {
			if rawValues[i] == nil {
				row[name] = nil
				continue
			}
			row[name] = string(rawValues[i])
		}
		rowList = append(rowList, row)
	}
	if err := rows.Err(); err != nil {
		return columns, rowList, truncated, skipped, hasNext, fmt.Errorf("读取结果失败: %s", err.Error())
	}
	return columns, rowList, truncated, skipped, hasNext, nil
}

func loadWebSQLSchema(ctx context.Context, db *sql.DB, driverName string, databaseName string, req webSQLSchemaRequest) (map[string]interface{}, error) {
	switch driverName {
	case "mysql":
		return loadMySQLSchema(ctx, db, databaseName, req)
	case "oracle":
		return loadOracleSchema(ctx, db, databaseName, req)
	case "sqlite":
		return loadSQLiteSchema(ctx, db, req)
	default:
		return nil, fmt.Errorf("暂不支持数据库类型: %s", driverName)
	}
}

func resolveMySQLDatabase(ctx context.Context, db *sql.DB, databaseName string) (string, error) {
	databaseName = strings.TrimSpace(databaseName)
	if databaseName != "" {
		return databaseName, nil
	}
	var current sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&current); err != nil {
		return "", fmt.Errorf("读取当前数据库失败: %s", err.Error())
	}
	if current.Valid {
		return strings.TrimSpace(current.String), nil
	}
	return "", nil
}

func loadMySQLSchemaRoot(ctx context.Context, db *sql.DB, databaseName string) (map[string]interface{}, error) {
	databaseName, err := resolveMySQLDatabase(ctx, db, databaseName)
	if err != nil {
		return nil, err
	}
	data := map[string]interface{}{
		"database": databaseName,
		"schemas":  []map[string]interface{}{},
	}
	if strings.TrimSpace(databaseName) == "" {
		rows, err := db.QueryContext(ctx, "SHOW DATABASES")
		if err != nil {
			return nil, fmt.Errorf("读取数据库列表失败: %s", err.Error())
		}
		defer rows.Close()
		dbList := make([]map[string]interface{}, 0)
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return nil, err
			}
			dbList = append(dbList, map[string]interface{}{"name": name, "type": "database"})
		}
		data["databases"] = dbList
		return data, rows.Err()
	}

	tableCount, viewCount, err := loadMySQLTableViewCounts(ctx, db, databaseName)
	if err != nil {
		return nil, err
	}
	indexCount, err := loadMySQLIndexCount(ctx, db, databaseName)
	if err != nil {
		return nil, err
	}
	data["schemas"] = []map[string]interface{}{
		{
			"name": databaseName,
			"type": "schema",
			"folders": []map[string]interface{}{
				{"name": "tables", "type": "tables", "label": "Tables", "count": tableCount},
				{"name": "views", "type": "views", "label": "Views", "count": viewCount},
				{"name": "indexes", "type": "indexes", "label": "Indexes", "count": indexCount},
			},
		},
	}
	return data, nil
}

func loadMySQLSchemaFolder(ctx context.Context, db *sql.DB, databaseName string, folderType string) (map[string]interface{}, error) {
	databaseName, err := resolveMySQLDatabase(ctx, db, databaseName)
	if err != nil {
		return nil, err
	}
	if databaseName == "" {
		return nil, fmt.Errorf("database 不能为空")
	}
	switch folderType {
	case "tables", "table":
		items, err := loadMySQLTableList(ctx, db, databaseName, "BASE TABLE")
		return map[string]interface{}{"database": databaseName, "folder_type": "tables", "items": items, "tables": items}, err
	case "views", "view":
		items, err := loadMySQLTableList(ctx, db, databaseName, "VIEW")
		return map[string]interface{}{"database": databaseName, "folder_type": "views", "items": items, "views": items}, err
	case "indexes", "index":
		items, err := loadMySQLIndexes(ctx, db, databaseName, map[string]map[string]interface{}{})
		return map[string]interface{}{"database": databaseName, "folder_type": "indexes", "items": items, "indexes": items}, err
	default:
		return nil, fmt.Errorf("folder_type 仅支持 tables/views/indexes")
	}
}

func loadMySQLSchemaObject(ctx context.Context, db *sql.DB, databaseName string, objectType string, objectName string, tableName string) (map[string]interface{}, error) {
	if objectName == "" {
		return nil, fmt.Errorf("object_name 不能为空")
	}
	databaseName, err := resolveMySQLDatabase(ctx, db, databaseName)
	if err != nil {
		return nil, err
	}
	if databaseName == "" {
		return nil, fmt.Errorf("database 不能为空")
	}
	if objectType == "index" {
		if tableName == "" {
			tableName = objectName
		}
		detail, err := loadMySQLIndexDetail(ctx, db, databaseName, objectName, tableName)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"object": detail}, nil
	}
	columns, err := loadMySQLColumnsForTable(ctx, db, databaseName, objectName)
	if err != nil {
		return nil, err
	}
	indexes, err := loadMySQLIndexesForTable(ctx, db, databaseName, objectName)
	if err != nil {
		return nil, err
	}
	kind := "table"
	if objectType == "view" {
		kind = "view"
	}
	object := map[string]interface{}{
		"database":       databaseName,
		"name":           objectName,
		"type":           kind,
		"columns":        columns,
		"indexes":        indexes,
		"columns_loaded": true,
	}
	return map[string]interface{}{"object": object}, nil
}

func loadMySQLSchemaSearch(ctx context.Context, db *sql.DB, databaseName string, query string) (map[string]interface{}, error) {
	databaseName, err := resolveMySQLDatabase(ctx, db, databaseName)
	if err != nil {
		return nil, err
	}
	if databaseName == "" {
		return nil, fmt.Errorf("database 不能为空")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return loadMySQLSchemaRoot(ctx, db, databaseName)
	}
	like := "%" + query + "%"
	tableRows, err := db.QueryContext(ctx, `
SELECT table_name, table_type, COALESCE(table_comment, ''), table_rows
FROM information_schema.tables
WHERE table_schema = ? AND table_name LIKE ?
ORDER BY CASE table_type WHEN 'BASE TABLE' THEN 0 WHEN 'VIEW' THEN 1 ELSE 2 END, table_name`, databaseName, like)
	if err != nil {
		return nil, fmt.Errorf("搜索表失败: %s", err.Error())
	}
	tables, views, tableIndex, err := readMySQLTableRows(tableRows)
	if err != nil {
		return nil, err
	}
	fieldTableRows, err := db.QueryContext(ctx, `
SELECT DISTINCT t.table_name, t.table_type, COALESCE(t.table_comment, ''), t.table_rows
FROM information_schema.tables t
JOIN information_schema.columns c ON c.table_schema = t.table_schema AND c.table_name = t.table_name
WHERE t.table_schema = ? AND c.column_name LIKE ?
ORDER BY CASE t.table_type WHEN 'BASE TABLE' THEN 0 WHEN 'VIEW' THEN 1 ELSE 2 END, t.table_name`, databaseName, like)
	if err != nil {
		return nil, fmt.Errorf("搜索字段表失败: %s", err.Error())
	}
	fieldTables, fieldViews, _, err := readMySQLTableRows(fieldTableRows)
	if err != nil {
		return nil, err
	}
	for _, item := range fieldTables {
		name := stringValue(item["name"])
		if tableIndex[name] == nil {
			tables = append(tables, item)
			tableIndex[name] = item
		}
	}
	for _, item := range fieldViews {
		name := stringValue(item["name"])
		if tableIndex[name] == nil {
			views = append(views, item)
			tableIndex[name] = item
		}
	}

	columnRows, err := db.QueryContext(ctx, `
SELECT table_name, column_name, column_type, is_nullable, column_key, column_default, extra, column_comment, ordinal_position
FROM information_schema.columns
WHERE table_schema = ? AND column_name LIKE ?
ORDER BY table_name, ordinal_position`, databaseName, like)
	if err != nil {
		return nil, fmt.Errorf("搜索字段失败: %s", err.Error())
	}
	if err := appendMySQLColumnRows(columnRows, tableIndex); err != nil {
		return nil, err
	}
	for tableName, table := range tableIndex {
		columns, _ := table["columns"].([]map[string]interface{})
		if len(columns) > 0 {
			continue
		}
		loadedColumns, err := loadMySQLColumnsForTable(ctx, db, databaseName, tableName)
		if err != nil {
			table["column_error"] = err.Error()
			continue
		}
		table["columns"] = loadedColumns
		table["columns_loaded"] = true
	}

	indexRows, err := db.QueryContext(ctx, `
SELECT table_name, index_name, non_unique, seq_in_index, column_name, index_type
FROM information_schema.statistics
WHERE table_schema = ?
  AND (index_name LIKE ? OR table_name LIKE ? OR column_name LIKE ?)
ORDER BY table_name, index_name, seq_in_index`, databaseName, like, like, like)
	if err != nil {
		return nil, fmt.Errorf("搜索索引失败: %s", err.Error())
	}
	indexes, err := readMySQLIndexRows(indexRows, tableIndex)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"database": databaseName,
		"tables":   tables,
		"views":    views,
		"indexes":  indexes,
		"schemas": []map[string]interface{}{
			{
				"name":    databaseName,
				"type":    "schema",
				"tables":  tables,
				"views":   views,
				"indexes": indexes,
			},
		},
	}, nil
}

func loadMySQLTableViewCounts(ctx context.Context, db *sql.DB, databaseName string) (int64, int64, error) {
	rows, err := db.QueryContext(ctx, `
SELECT
  SUM(CASE WHEN table_type = 'BASE TABLE' THEN 1 ELSE 0 END) AS table_count,
  SUM(CASE WHEN table_type = 'VIEW' THEN 1 ELSE 0 END) AS view_count
FROM information_schema.tables
WHERE table_schema = ?`, databaseName)
	if err != nil {
		return 0, 0, fmt.Errorf("读取表视图数量失败: %s", err.Error())
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, 0, nil
	}
	var tableCount sql.NullInt64
	var viewCount sql.NullInt64
	if err := rows.Scan(&tableCount, &viewCount); err != nil {
		return 0, 0, err
	}
	return tableCount.Int64, viewCount.Int64, rows.Err()
}

func loadMySQLIndexCount(ctx context.Context, db *sql.DB, databaseName string) (int64, error) {
	rows, err := db.QueryContext(ctx, `
SELECT COUNT(*)
FROM (
  SELECT DISTINCT table_name, index_name
  FROM information_schema.statistics
  WHERE table_schema = ?
) t`, databaseName)
	if err != nil {
		return 0, fmt.Errorf("读取索引数量失败: %s", err.Error())
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, nil
	}
	var count int64
	if err := rows.Scan(&count); err != nil {
		return 0, err
	}
	return count, rows.Err()
}

func loadMySQLTableList(ctx context.Context, db *sql.DB, databaseName string, tableType string) ([]map[string]interface{}, error) {
	rows, err := db.QueryContext(ctx, `
SELECT table_name, table_type, COALESCE(table_comment, ''), table_rows
FROM information_schema.tables
WHERE table_schema = ? AND table_type = ?
ORDER BY table_name`, databaseName, tableType)
	if err != nil {
		return nil, fmt.Errorf("读取表列表失败: %s", err.Error())
	}
	tables, views, _, err := readMySQLTableRows(rows)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(tableType, "VIEW") {
		return views, nil
	}
	return tables, nil
}

func readMySQLTableRows(rows *sql.Rows) ([]map[string]interface{}, []map[string]interface{}, map[string]map[string]interface{}, error) {
	defer rows.Close()
	tableList := make([]map[string]interface{}, 0)
	viewList := make([]map[string]interface{}, 0)
	tableIndex := map[string]map[string]interface{}{}
	for rows.Next() {
		var tableName string
		var tableType string
		var tableComment string
		var tableRowsCount sql.NullInt64
		if err := rows.Scan(&tableName, &tableType, &tableComment, &tableRowsCount); err != nil {
			return nil, nil, nil, err
		}
		kind := "table"
		if strings.EqualFold(tableType, "VIEW") {
			kind = "view"
		}
		item := map[string]interface{}{
			"name":           tableName,
			"type":           kind,
			"table_type":     tableType,
			"comment":        tableComment,
			"columns":        []map[string]interface{}{},
			"indexes":        []map[string]interface{}{},
			"columns_loaded": false,
		}
		if tableRowsCount.Valid {
			item["rows"] = tableRowsCount.Int64
		}
		if kind == "view" {
			viewList = append(viewList, item)
		} else {
			tableList = append(tableList, item)
		}
		tableIndex[tableName] = item
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, err
	}
	return tableList, viewList, tableIndex, nil
}

func appendMySQLColumnRows(rows *sql.Rows, tableIndex map[string]map[string]interface{}) error {
	defer rows.Close()
	for rows.Next() {
		var tableName string
		var columnName string
		var columnType string
		var nullable string
		var columnKey sql.NullString
		var columnDefault sql.NullString
		var extra sql.NullString
		var columnComment sql.NullString
		var ordinalPosition int
		if err := rows.Scan(&tableName, &columnName, &columnType, &nullable, &columnKey, &columnDefault, &extra, &columnComment, &ordinalPosition); err != nil {
			return err
		}
		table := tableIndex[tableName]
		if table == nil {
			continue
		}
		columns, _ := table["columns"].([]map[string]interface{})
		columns = append(columns, map[string]interface{}{
			"name":     columnName,
			"type":     columnType,
			"nullable": strings.EqualFold(nullable, "YES"),
			"key":      columnKey.String,
			"default":  columnDefault.String,
			"extra":    extra.String,
			"comment":  columnComment.String,
			"ordinal":  ordinalPosition,
		})
		table["columns"] = columns
		table["columns_loaded"] = true
	}
	return rows.Err()
}

func loadMySQLColumnsForTable(ctx context.Context, db *sql.DB, databaseName string, tableName string) ([]map[string]interface{}, error) {
	table := map[string]interface{}{"columns": []map[string]interface{}{}}
	columnRows, err := db.QueryContext(ctx, `
SELECT table_name, column_name, column_type, is_nullable, column_key, column_default, extra, column_comment, ordinal_position
FROM information_schema.columns
WHERE table_schema = ? AND table_name = ?
ORDER BY ordinal_position`, databaseName, tableName)
	if err != nil {
		return nil, fmt.Errorf("读取字段列表失败: %s", err.Error())
	}
	if err := appendMySQLColumnRows(columnRows, map[string]map[string]interface{}{tableName: table}); err != nil {
		return nil, err
	}
	columns, _ := table["columns"].([]map[string]interface{})
	return columns, nil
}

func readMySQLIndexRows(rows *sql.Rows, tableIndex map[string]map[string]interface{}) ([]map[string]interface{}, error) {
	defer rows.Close()
	indexList := make([]map[string]interface{}, 0)
	indexMap := map[string]map[string]interface{}{}
	for rows.Next() {
		var tableName string
		var indexName string
		var nonUnique int
		var seqInIndex int
		var columnName sql.NullString
		var indexType sql.NullString
		if err := rows.Scan(&tableName, &indexName, &nonUnique, &seqInIndex, &columnName, &indexType); err != nil {
			return nil, err
		}
		key := tableName + "\x00" + indexName
		item := indexMap[key]
		if item == nil {
			item = map[string]interface{}{
				"name":       indexName,
				"type":       "index",
				"table_name": tableName,
				"unique":     nonUnique == 0,
				"index_type": indexType.String,
				"columns":    []string{},
			}
			indexMap[key] = item
			indexList = append(indexList, item)
			if table := tableIndex[tableName]; table != nil {
				indexes, _ := table["indexes"].([]map[string]interface{})
				indexes = append(indexes, item)
				table["indexes"] = indexes
			}
		}
		columns, _ := item["columns"].([]string)
		if columnName.Valid {
			columns = append(columns, columnName.String)
		}
		item["columns"] = columns
		item["seq_in_index"] = seqInIndex
	}
	return indexList, rows.Err()
}

func loadMySQLIndexesForTable(ctx context.Context, db *sql.DB, databaseName string, tableName string) ([]map[string]interface{}, error) {
	rows, err := db.QueryContext(ctx, `
SELECT table_name, index_name, non_unique, seq_in_index, column_name, index_type
FROM information_schema.statistics
WHERE table_schema = ? AND table_name = ?
ORDER BY table_name, index_name, seq_in_index`, databaseName, tableName)
	if err != nil {
		return nil, fmt.Errorf("读取索引列表失败: %s", err.Error())
	}
	return readMySQLIndexRows(rows, map[string]map[string]interface{}{})
}

func loadMySQLSchema(ctx context.Context, db *sql.DB, databaseName string, req webSQLSchemaRequest) (map[string]interface{}, error) {
	switch req.Scope {
	case "root":
		return loadMySQLSchemaRoot(ctx, db, databaseName)
	case "folder":
		return loadMySQLSchemaFolder(ctx, db, databaseName, req.FolderType)
	case "object":
		return loadMySQLSchemaObject(ctx, db, databaseName, req.ObjectType, req.ObjectName, req.TableName)
	case "search":
		return loadMySQLSchemaSearch(ctx, db, databaseName, req.Query)
	case "completion":
		return loadMySQLSchema(ctx, db, databaseName, webSQLSchemaRequest{})
	}
	databaseName, err := resolveMySQLDatabase(ctx, db, databaseName)
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"database": databaseName,
		"tables":   []map[string]interface{}{},
		"views":    []map[string]interface{}{},
		"indexes":  []map[string]interface{}{},
		"schemas":  []map[string]interface{}{},
	}
	if strings.TrimSpace(databaseName) == "" {
		rows, err := db.QueryContext(ctx, "SHOW DATABASES")
		if err != nil {
			return nil, fmt.Errorf("读取数据库列表失败: %s", err.Error())
		}
		defer rows.Close()
		dbList := make([]map[string]interface{}, 0)
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return nil, err
			}
			dbList = append(dbList, map[string]interface{}{"name": name, "type": "database"})
		}
		data["databases"] = dbList
		return data, rows.Err()
	}

	tableRows, err := db.QueryContext(ctx, `
SELECT table_name, table_type, COALESCE(table_comment, ''), table_rows
FROM information_schema.tables
WHERE table_schema = ?
ORDER BY CASE table_type WHEN 'BASE TABLE' THEN 0 WHEN 'VIEW' THEN 1 ELSE 2 END, table_name`, databaseName)
	if err != nil {
		return nil, fmt.Errorf("读取表列表失败: %s", err.Error())
	}
	defer tableRows.Close()

	tableList := make([]map[string]interface{}, 0)
	viewList := make([]map[string]interface{}, 0)
	tableIndex := map[string]map[string]interface{}{}
	for tableRows.Next() {
		var tableName string
		var tableType string
		var tableComment string
		var tableRowsCount sql.NullInt64
		if err := tableRows.Scan(&tableName, &tableType, &tableComment, &tableRowsCount); err != nil {
			return nil, err
		}
		kind := "table"
		if strings.EqualFold(tableType, "VIEW") {
			kind = "view"
		}
		item := map[string]interface{}{
			"name":       tableName,
			"type":       kind,
			"table_type": tableType,
			"comment":    tableComment,
			"columns":    []map[string]interface{}{},
			"indexes":    []map[string]interface{}{},
		}
		if tableRowsCount.Valid {
			item["rows"] = tableRowsCount.Int64
		}
		if kind == "view" {
			viewList = append(viewList, item)
		} else {
			tableList = append(tableList, item)
		}
		tableIndex[tableName] = item
	}
	if err := tableRows.Err(); err != nil {
		return nil, err
	}

	columnRows, err := db.QueryContext(ctx, `
SELECT table_name, column_name, column_type, is_nullable, column_key, column_default, extra, column_comment, ordinal_position
FROM information_schema.columns
WHERE table_schema = ?
ORDER BY table_name, ordinal_position`, databaseName)
	if err != nil {
		return nil, fmt.Errorf("读取字段列表失败: %s", err.Error())
	}
	defer columnRows.Close()
	for columnRows.Next() {
		var tableName string
		var columnName string
		var columnType string
		var nullable string
		var columnKey sql.NullString
		var columnDefault sql.NullString
		var extra sql.NullString
		var columnComment sql.NullString
		var ordinalPosition int
		if err := columnRows.Scan(&tableName, &columnName, &columnType, &nullable, &columnKey, &columnDefault, &extra, &columnComment, &ordinalPosition); err != nil {
			return nil, err
		}
		table := tableIndex[tableName]
		if table == nil {
			continue
		}
		columns, _ := table["columns"].([]map[string]interface{})
		columns = append(columns, map[string]interface{}{
			"name":     columnName,
			"type":     columnType,
			"nullable": strings.EqualFold(nullable, "YES"),
			"key":      columnKey.String,
			"default":  columnDefault.String,
			"extra":    extra.String,
			"comment":  columnComment.String,
			"ordinal":  ordinalPosition,
		})
		table["columns"] = columns
	}
	if err := columnRows.Err(); err != nil {
		return nil, err
	}
	indexList, err := loadMySQLIndexes(ctx, db, databaseName, tableIndex)
	if err != nil {
		return nil, err
	}
	data["tables"] = tableList
	data["views"] = viewList
	data["indexes"] = indexList
	data["schemas"] = []map[string]interface{}{
		{
			"name":    databaseName,
			"type":    "schema",
			"tables":  tableList,
			"views":   viewList,
			"indexes": indexList,
		},
	}
	return data, nil
}

func resolveOracleSchema(ctx context.Context, db *sql.DB, schemaName string) (string, error) {
	schemaName = strings.Trim(strings.TrimSpace(schemaName), `"`)
	if schemaName != "" {
		return strings.ToUpper(schemaName), nil
	}
	var current sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA') FROM DUAL").Scan(&current); err != nil {
		return "", fmt.Errorf("读取当前schema失败: %s", err.Error())
	}
	if current.Valid && strings.TrimSpace(current.String) != "" {
		return strings.ToUpper(strings.TrimSpace(current.String)), nil
	}
	if err := db.QueryRowContext(ctx, "SELECT USER FROM DUAL").Scan(&current); err != nil {
		return "", fmt.Errorf("读取当前用户失败: %s", err.Error())
	}
	if current.Valid {
		return strings.ToUpper(strings.TrimSpace(current.String)), nil
	}
	return "", nil
}

func normalizeOracleObjectName(value string) string {
	return strings.Trim(strings.TrimSpace(value), `"`)
}

func loadOracleSchemaRoot(ctx context.Context, db *sql.DB, schemaName string) (map[string]interface{}, error) {
	schemaName, err := resolveOracleSchema(ctx, db, schemaName)
	if err != nil {
		return nil, err
	}
	if schemaName == "" {
		return nil, fmt.Errorf("schema 不能为空")
	}
	rows, err := db.QueryContext(ctx, `
SELECT object_type, COUNT(*)
FROM all_objects
WHERE owner = :1
  AND object_type IN ('TABLE','VIEW','INDEX')
  AND object_name NOT LIKE 'BIN$%'
GROUP BY object_type`, schemaName)
	if err != nil {
		return nil, fmt.Errorf("读取Oracle目录数量失败: %s", err.Error())
	}
	defer rows.Close()
	counts := map[string]int64{"TABLE": 0, "VIEW": 0, "INDEX": 0}
	for rows.Next() {
		var typ string
		var count int64
		if err := rows.Scan(&typ, &count); err != nil {
			return nil, err
		}
		counts[strings.ToUpper(typ)] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"database": schemaName,
		"schema":   schemaName,
		"schemas": []map[string]interface{}{
			{
				"name": schemaName,
				"type": "schema",
				"folders": []map[string]interface{}{
					{"name": "tables", "type": "tables", "label": "Tables", "count": counts["TABLE"]},
					{"name": "views", "type": "views", "label": "Views", "count": counts["VIEW"]},
					{"name": "indexes", "type": "indexes", "label": "Indexes", "count": counts["INDEX"]},
				},
			},
		},
	}, nil
}

func loadOracleSchemaFolder(ctx context.Context, db *sql.DB, schemaName string, folderType string) (map[string]interface{}, error) {
	schemaName, err := resolveOracleSchema(ctx, db, schemaName)
	if err != nil {
		return nil, err
	}
	if schemaName == "" {
		return nil, fmt.Errorf("schema 不能为空")
	}
	switch folderType {
	case "tables", "table":
		items, err := loadOracleObjectList(ctx, db, schemaName, "TABLE")
		return map[string]interface{}{"database": schemaName, "schema": schemaName, "folder_type": "tables", "items": items, "tables": items}, err
	case "views", "view":
		items, err := loadOracleObjectList(ctx, db, schemaName, "VIEW")
		return map[string]interface{}{"database": schemaName, "schema": schemaName, "folder_type": "views", "items": items, "views": items}, err
	case "indexes", "index":
		items, err := loadOracleIndexes(ctx, db, schemaName, map[string]map[string]interface{}{})
		return map[string]interface{}{"database": schemaName, "schema": schemaName, "folder_type": "indexes", "items": items, "indexes": items}, err
	default:
		return nil, fmt.Errorf("folder_type 仅支持 tables/views/indexes")
	}
}

func loadOracleSchemaObject(ctx context.Context, db *sql.DB, schemaName string, objectType string, objectName string, tableName string) (map[string]interface{}, error) {
	if objectName == "" {
		return nil, fmt.Errorf("object_name 不能为空")
	}
	schemaName, err := resolveOracleSchema(ctx, db, schemaName)
	if err != nil {
		return nil, err
	}
	if schemaName == "" {
		return nil, fmt.Errorf("schema 不能为空")
	}
	objectName = normalizeOracleObjectName(objectName)
	tableName = normalizeOracleObjectName(tableName)
	if objectType == "index" {
		if tableName == "" {
			tableName = objectName
		}
		detail, err := loadOracleIndexDetail(ctx, db, schemaName, objectName, tableName)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"object": detail}, nil
	}
	columns, err := loadOracleColumnsForTable(ctx, db, schemaName, objectName)
	if err != nil {
		return nil, err
	}
	indexes, err := loadOracleIndexesForTable(ctx, db, schemaName, objectName)
	if err != nil {
		return nil, err
	}
	kind := "table"
	if objectType == "view" {
		kind = "view"
	}
	object := map[string]interface{}{
		"database":       schemaName,
		"schema":         schemaName,
		"name":           objectName,
		"type":           kind,
		"columns":        columns,
		"indexes":        indexes,
		"columns_loaded": true,
	}
	return map[string]interface{}{"object": object}, nil
}

func loadOracleSchemaSearch(ctx context.Context, db *sql.DB, schemaName string, query string) (map[string]interface{}, error) {
	schemaName, err := resolveOracleSchema(ctx, db, schemaName)
	if err != nil {
		return nil, err
	}
	if schemaName == "" {
		return nil, fmt.Errorf("schema 不能为空")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return loadOracleSchemaRoot(ctx, db, schemaName)
	}
	like := "%" + strings.ToUpper(query) + "%"
	objectRows, err := db.QueryContext(ctx, `
SELECT o.object_name, o.object_type, NVL(tc.comments, ''), t.num_rows
FROM all_objects o
LEFT JOIN all_tab_comments tc ON tc.owner = o.owner AND tc.table_name = o.object_name
LEFT JOIN all_tables t ON t.owner = o.owner AND t.table_name = o.object_name
WHERE o.owner = :1
  AND o.object_type IN ('TABLE','VIEW')
  AND o.object_name NOT LIKE 'BIN$%'
  AND UPPER(o.object_name) LIKE :2
ORDER BY CASE o.object_type WHEN 'TABLE' THEN 0 WHEN 'VIEW' THEN 1 ELSE 2 END, o.object_name`, schemaName, like)
	if err != nil {
		return nil, fmt.Errorf("搜索Oracle对象失败: %s", err.Error())
	}
	tables, views, tableIndex, err := readOracleObjectRows(objectRows)
	if err != nil {
		return nil, err
	}

	columnObjectRows, err := db.QueryContext(ctx, `
SELECT DISTINCT c.table_name
FROM all_tab_columns c
JOIN all_objects o ON o.owner = c.owner AND o.object_name = c.table_name
WHERE c.owner = :1
  AND o.object_type IN ('TABLE','VIEW')
  AND UPPER(c.column_name) LIKE :2
ORDER BY c.table_name`, schemaName, like)
	if err != nil {
		return nil, fmt.Errorf("搜索Oracle字段表失败: %s", err.Error())
	}
	for columnObjectRows.Next() {
		var tableName string
		if err := columnObjectRows.Scan(&tableName); err != nil {
			columnObjectRows.Close()
			return nil, err
		}
		if tableIndex[tableName] != nil {
			continue
		}
		item, err := loadOracleObjectByName(ctx, db, schemaName, tableName)
		if err != nil || item == nil {
			continue
		}
		tableIndex[tableName] = item
		if stringValue(item["type"]) == "view" {
			views = append(views, item)
		} else {
			tables = append(tables, item)
		}
	}
	if err := columnObjectRows.Err(); err != nil {
		columnObjectRows.Close()
		return nil, err
	}
	columnObjectRows.Close()

	for tableName, table := range tableIndex {
		columns, err := loadOracleColumnsForTable(ctx, db, schemaName, tableName)
		if err != nil {
			table["column_error"] = err.Error()
			continue
		}
		table["columns"] = columns
		table["columns_loaded"] = true
	}
	indexes, err := loadOracleSearchIndexes(ctx, db, schemaName, query, tableIndex)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"database": schemaName,
		"schema":   schemaName,
		"tables":   tables,
		"views":    views,
		"indexes":  indexes,
		"schemas": []map[string]interface{}{
			{
				"name":    schemaName,
				"type":    "schema",
				"tables":  tables,
				"views":   views,
				"indexes": indexes,
			},
		},
	}, nil
}

func loadOracleSchema(ctx context.Context, db *sql.DB, schemaName string, req webSQLSchemaRequest) (map[string]interface{}, error) {
	switch req.Scope {
	case "root":
		return loadOracleSchemaRoot(ctx, db, schemaName)
	case "folder":
		return loadOracleSchemaFolder(ctx, db, schemaName, req.FolderType)
	case "object":
		return loadOracleSchemaObject(ctx, db, schemaName, req.ObjectType, req.ObjectName, req.TableName)
	case "search":
		return loadOracleSchemaSearch(ctx, db, schemaName, req.Query)
	case "completion":
		return loadOracleSchema(ctx, db, schemaName, webSQLSchemaRequest{})
	}
	schemaName, err := resolveOracleSchema(ctx, db, schemaName)
	if err != nil {
		return nil, err
	}
	if schemaName == "" {
		return nil, fmt.Errorf("schema 不能为空")
	}
	tableRows, err := db.QueryContext(ctx, `
SELECT o.object_name, o.object_type, NVL(tc.comments, ''), t.num_rows
FROM all_objects o
LEFT JOIN all_tab_comments tc ON tc.owner = o.owner AND tc.table_name = o.object_name
LEFT JOIN all_tables t ON t.owner = o.owner AND t.table_name = o.object_name
WHERE o.owner = :1
  AND o.object_type IN ('TABLE','VIEW')
  AND o.object_name NOT LIKE 'BIN$%'
ORDER BY CASE o.object_type WHEN 'TABLE' THEN 0 WHEN 'VIEW' THEN 1 ELSE 2 END, o.object_name`, schemaName)
	if err != nil {
		return nil, fmt.Errorf("读取Oracle表列表失败: %s", err.Error())
	}
	tableList, viewList, tableIndex, err := readOracleObjectRows(tableRows)
	if err != nil {
		return nil, err
	}
	for tableName, table := range tableIndex {
		columns, err := loadOracleColumnsForTable(ctx, db, schemaName, tableName)
		if err != nil {
			table["column_error"] = err.Error()
			continue
		}
		table["columns"] = columns
		table["columns_loaded"] = true
	}
	indexList, err := loadOracleIndexes(ctx, db, schemaName, tableIndex)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"database": schemaName,
		"schema":   schemaName,
		"tables":   tableList,
		"views":    viewList,
		"indexes":  indexList,
		"schemas": []map[string]interface{}{
			{
				"name":    schemaName,
				"type":    "schema",
				"tables":  tableList,
				"views":   viewList,
				"indexes": indexList,
			},
		},
	}, nil
}

func loadOracleObjectList(ctx context.Context, db *sql.DB, schemaName string, objectType string) ([]map[string]interface{}, error) {
	rows, err := db.QueryContext(ctx, `
SELECT o.object_name, o.object_type, NVL(tc.comments, ''), t.num_rows
FROM all_objects o
LEFT JOIN all_tab_comments tc ON tc.owner = o.owner AND tc.table_name = o.object_name
LEFT JOIN all_tables t ON t.owner = o.owner AND t.table_name = o.object_name
WHERE o.owner = :1
  AND o.object_type = :2
  AND o.object_name NOT LIKE 'BIN$%'
ORDER BY o.object_name`, schemaName, strings.ToUpper(objectType))
	if err != nil {
		return nil, fmt.Errorf("读取Oracle对象列表失败: %s", err.Error())
	}
	tables, views, _, err := readOracleObjectRows(rows)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(objectType, "VIEW") {
		return views, nil
	}
	return tables, nil
}

func loadOracleObjectByName(ctx context.Context, db *sql.DB, schemaName string, objectName string) (map[string]interface{}, error) {
	objectName = normalizeOracleObjectName(objectName)
	rows, err := db.QueryContext(ctx, `
SELECT o.object_name, o.object_type, NVL(tc.comments, ''), t.num_rows
FROM all_objects o
LEFT JOIN all_tab_comments tc ON tc.owner = o.owner AND tc.table_name = o.object_name
LEFT JOIN all_tables t ON t.owner = o.owner AND t.table_name = o.object_name
WHERE o.owner = :1
  AND (o.object_name = :2 OR o.object_name = UPPER(:3))
  AND o.object_type IN ('TABLE','VIEW')
  AND o.object_name NOT LIKE 'BIN$%'
ORDER BY CASE o.object_type WHEN 'TABLE' THEN 0 WHEN 'VIEW' THEN 1 ELSE 2 END`, schemaName, objectName, objectName)
	if err != nil {
		return nil, fmt.Errorf("读取Oracle对象失败: %s", err.Error())
	}
	tables, views, _, err := readOracleObjectRows(rows)
	if err != nil {
		return nil, err
	}
	if len(tables) > 0 {
		return tables[0], nil
	}
	if len(views) > 0 {
		return views[0], nil
	}
	return nil, nil
}

func readOracleObjectRows(rows *sql.Rows) ([]map[string]interface{}, []map[string]interface{}, map[string]map[string]interface{}, error) {
	defer rows.Close()
	tableList := make([]map[string]interface{}, 0)
	viewList := make([]map[string]interface{}, 0)
	tableIndex := map[string]map[string]interface{}{}
	for rows.Next() {
		var objectName string
		var objectType string
		var comment sql.NullString
		var rowCount sql.NullInt64
		if err := rows.Scan(&objectName, &objectType, &comment, &rowCount); err != nil {
			return nil, nil, nil, err
		}
		kind := "table"
		if strings.EqualFold(objectType, "VIEW") {
			kind = "view"
		}
		item := map[string]interface{}{
			"name":           objectName,
			"type":           kind,
			"table_type":     objectType,
			"comment":        comment.String,
			"columns":        []map[string]interface{}{},
			"indexes":        []map[string]interface{}{},
			"columns_loaded": false,
		}
		if rowCount.Valid {
			item["rows"] = rowCount.Int64
		}
		if kind == "view" {
			viewList = append(viewList, item)
		} else {
			tableList = append(tableList, item)
		}
		tableIndex[objectName] = item
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, err
	}
	return tableList, viewList, tableIndex, nil
}

func loadOracleColumnsForTable(ctx context.Context, db *sql.DB, schemaName string, tableName string) ([]map[string]interface{}, error) {
	tableName = normalizeOracleObjectName(tableName)
	rows, err := db.QueryContext(ctx, `
SELECT
  c.column_name,
  c.data_type,
  c.data_length,
  c.data_precision,
  c.data_scale,
  c.nullable,
  NVL(cc.comments, ''),
  c.column_id,
  CASE WHEN pk.column_name IS NULL THEN '' ELSE 'PRI' END
FROM all_tab_columns c
LEFT JOIN all_col_comments cc
  ON cc.owner = c.owner AND cc.table_name = c.table_name AND cc.column_name = c.column_name
LEFT JOIN (
  SELECT acc.column_name
  FROM all_constraints ac
  JOIN all_cons_columns acc
    ON acc.owner = ac.owner AND acc.constraint_name = ac.constraint_name AND acc.table_name = ac.table_name
  WHERE ac.owner = :1 AND ac.table_name = :2 AND ac.constraint_type = 'P'
) pk ON pk.column_name = c.column_name
WHERE c.owner = :3
  AND (c.table_name = :4 OR c.table_name = UPPER(:5))
ORDER BY c.column_id`, schemaName, tableName, schemaName, tableName, tableName)
	if err != nil {
		return nil, fmt.Errorf("读取Oracle字段列表失败: %s", err.Error())
	}
	defer rows.Close()
	columns := make([]map[string]interface{}, 0)
	for rows.Next() {
		var columnName string
		var dataType string
		var dataLength sql.NullInt64
		var dataPrecision sql.NullInt64
		var dataScale sql.NullInt64
		var nullable string
		var comment sql.NullString
		var ordinal sql.NullInt64
		var columnKey sql.NullString
		if err := rows.Scan(&columnName, &dataType, &dataLength, &dataPrecision, &dataScale, &nullable, &comment, &ordinal, &columnKey); err != nil {
			return nil, err
		}
		columns = append(columns, map[string]interface{}{
			"name":     columnName,
			"type":     formatOracleColumnType(dataType, dataLength, dataPrecision, dataScale),
			"nullable": strings.EqualFold(nullable, "Y"),
			"key":      columnKey.String,
			"default":  "",
			"extra":    "",
			"comment":  comment.String,
			"ordinal":  ordinal.Int64,
		})
	}
	return columns, rows.Err()
}

func formatOracleColumnType(dataType string, dataLength sql.NullInt64, dataPrecision sql.NullInt64, dataScale sql.NullInt64) string {
	typ := strings.ToUpper(strings.TrimSpace(dataType))
	switch typ {
	case "CHAR", "NCHAR", "VARCHAR2", "NVARCHAR2", "RAW":
		if dataLength.Valid && dataLength.Int64 > 0 {
			return fmt.Sprintf("%s(%d)", typ, dataLength.Int64)
		}
	case "NUMBER":
		if dataPrecision.Valid && dataPrecision.Int64 > 0 {
			if dataScale.Valid {
				return fmt.Sprintf("NUMBER(%d,%d)", dataPrecision.Int64, dataScale.Int64)
			}
			return fmt.Sprintf("NUMBER(%d)", dataPrecision.Int64)
		}
	case "FLOAT":
		if dataPrecision.Valid && dataPrecision.Int64 > 0 {
			return fmt.Sprintf("FLOAT(%d)", dataPrecision.Int64)
		}
	}
	return typ
}

func loadOracleIndexes(ctx context.Context, db *sql.DB, schemaName string, tableIndex map[string]map[string]interface{}) ([]map[string]interface{}, error) {
	rows, err := db.QueryContext(ctx, `
SELECT i.table_name, i.index_name, i.uniqueness, NVL(i.index_type, ''), ic.column_name, ic.column_position
FROM all_indexes i
LEFT JOIN all_ind_columns ic
  ON ic.index_owner = i.owner AND ic.index_name = i.index_name
WHERE i.owner = :1
ORDER BY i.table_name, i.index_name, ic.column_position`, schemaName)
	if err != nil {
		return nil, fmt.Errorf("读取Oracle索引列表失败: %s", err.Error())
	}
	return readOracleIndexRows(rows, tableIndex)
}

func loadOracleIndexesForTable(ctx context.Context, db *sql.DB, schemaName string, tableName string) ([]map[string]interface{}, error) {
	tableName = normalizeOracleObjectName(tableName)
	rows, err := db.QueryContext(ctx, `
SELECT i.table_name, i.index_name, i.uniqueness, NVL(i.index_type, ''), ic.column_name, ic.column_position
FROM all_indexes i
LEFT JOIN all_ind_columns ic
  ON ic.index_owner = i.owner AND ic.index_name = i.index_name
WHERE i.owner = :1
  AND (i.table_name = :2 OR i.table_name = UPPER(:3))
ORDER BY i.table_name, i.index_name, ic.column_position`, schemaName, tableName, tableName)
	if err != nil {
		return nil, fmt.Errorf("读取Oracle表索引失败: %s", err.Error())
	}
	return readOracleIndexRows(rows, map[string]map[string]interface{}{})
}

func loadOracleSearchIndexes(ctx context.Context, db *sql.DB, schemaName string, query string, tableIndex map[string]map[string]interface{}) ([]map[string]interface{}, error) {
	like := "%" + strings.ToUpper(strings.TrimSpace(query)) + "%"
	rows, err := db.QueryContext(ctx, `
SELECT i.table_name, i.index_name, i.uniqueness, NVL(i.index_type, ''), ic.column_name, ic.column_position
FROM all_indexes i
LEFT JOIN all_ind_columns ic
  ON ic.index_owner = i.owner AND ic.index_name = i.index_name
WHERE i.owner = :1
  AND (UPPER(i.index_name) LIKE :2 OR UPPER(i.table_name) LIKE :3 OR UPPER(ic.column_name) LIKE :4)
ORDER BY i.table_name, i.index_name, ic.column_position`, schemaName, like, like, like)
	if err != nil {
		return nil, fmt.Errorf("搜索Oracle索引失败: %s", err.Error())
	}
	return readOracleIndexRows(rows, tableIndex)
}

func readOracleIndexRows(rows *sql.Rows, tableIndex map[string]map[string]interface{}) ([]map[string]interface{}, error) {
	defer rows.Close()
	indexList := make([]map[string]interface{}, 0)
	indexMap := map[string]map[string]interface{}{}
	for rows.Next() {
		var tableName string
		var indexName string
		var uniqueness sql.NullString
		var indexType sql.NullString
		var columnName sql.NullString
		var columnPosition sql.NullInt64
		if err := rows.Scan(&tableName, &indexName, &uniqueness, &indexType, &columnName, &columnPosition); err != nil {
			return nil, err
		}
		key := tableName + "\x00" + indexName
		item := indexMap[key]
		if item == nil {
			item = map[string]interface{}{
				"name":       indexName,
				"type":       "index",
				"table_name": tableName,
				"unique":     strings.EqualFold(uniqueness.String, "UNIQUE"),
				"index_type": indexType.String,
				"columns":    []string{},
			}
			indexMap[key] = item
			indexList = append(indexList, item)
			if table := tableIndex[tableName]; table != nil {
				indexes, _ := table["indexes"].([]map[string]interface{})
				indexes = append(indexes, item)
				table["indexes"] = indexes
			}
		}
		columns, _ := item["columns"].([]string)
		if columnName.Valid {
			columns = append(columns, columnName.String)
		}
		item["columns"] = columns
		if columnPosition.Valid {
			item["seq_in_index"] = columnPosition.Int64
		}
	}
	return indexList, rows.Err()
}

func loadOracleObjectDetail(ctx context.Context, db *sql.DB, schemaName string, objectType string, objectName string, tableName string) (map[string]interface{}, error) {
	schemaName, err := resolveOracleSchema(ctx, db, schemaName)
	if err != nil {
		return nil, err
	}
	if schemaName == "" {
		return nil, fmt.Errorf("schema 不能为空")
	}
	objectName = normalizeOracleObjectName(objectName)
	tableName = normalizeOracleObjectName(tableName)
	if objectType == "index" {
		return loadOracleIndexDetail(ctx, db, schemaName, objectName, tableName)
	}
	columns, err := loadOracleColumnsForTable(ctx, db, schemaName, objectName)
	if err != nil {
		return nil, err
	}
	ddl, err := readOracleDDL(ctx, db, strings.ToUpper(objectType), objectName, schemaName)
	if err != nil {
		return nil, err
	}
	formattedDDL := formatWebSQLDDL(ddl)
	return map[string]interface{}{
		"database":    schemaName,
		"schema":      schemaName,
		"name":        objectName,
		"type":        objectType,
		"columns":     columns,
		"ddl":         formattedDDL,
		"sql":         formattedDDL,
		"raw_ddl":     ddl,
		"object_type": objectType,
	}, nil
}

func loadOracleIndexDetail(ctx context.Context, db *sql.DB, schemaName string, indexName string, tableName string) (map[string]interface{}, error) {
	indexName = normalizeOracleObjectName(indexName)
	tableName = normalizeOracleObjectName(tableName)
	if tableName == "" {
		var resolvedTable sql.NullString
		if err := db.QueryRowContext(ctx, `
SELECT table_name
FROM all_indexes
WHERE owner = :1 AND (index_name = :2 OR index_name = UPPER(:3))`, schemaName, indexName, indexName).Scan(&resolvedTable); err == nil && resolvedTable.Valid {
			tableName = resolvedTable.String
		}
	}
	ddl, err := readOracleDDL(ctx, db, "INDEX", indexName, schemaName)
	if err != nil {
		return nil, err
	}
	indexes, err := loadOracleSearchIndexes(ctx, db, schemaName, indexName, map[string]map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	columns := []string{}
	unique := false
	indexType := ""
	for _, indexItem := range indexes {
		if !strings.EqualFold(stringValue(indexItem["name"]), indexName) {
			continue
		}
		if tableName != "" && !strings.EqualFold(stringValue(indexItem["table_name"]), tableName) {
			continue
		}
		if rawColumns, ok := indexItem["columns"].([]string); ok {
			columns = rawColumns
		}
		unique = indexItem["unique"] == true
		indexType = stringValue(indexItem["index_type"])
		if tableName == "" {
			tableName = stringValue(indexItem["table_name"])
		}
		break
	}
	formattedDDL := formatWebSQLDDL(ddl)
	return map[string]interface{}{
		"database":    schemaName,
		"schema":      schemaName,
		"name":        indexName,
		"type":        "index",
		"table_name":  tableName,
		"unique":      unique,
		"index_type":  indexType,
		"columns":     columns,
		"ddl":         formattedDDL,
		"sql":         formattedDDL,
		"raw_ddl":     ddl,
		"object_type": "index",
	}, nil
}

func readOracleDDL(ctx context.Context, db *sql.DB, objectType string, objectName string, schemaName string) (string, error) {
	objectName = normalizeOracleObjectName(objectName)
	objectType = strings.ToUpper(strings.TrimSpace(objectType))
	if objectType == "" {
		objectType = "TABLE"
	}
	var ddl sql.NullString
	err := db.QueryRowContext(ctx, "SELECT DBMS_METADATA.GET_DDL(:1, :2, :3) FROM DUAL", objectType, objectName, schemaName).Scan(&ddl)
	if err != nil {
		return "", fmt.Errorf("读取Oracle对象DDL失败: %s", err.Error())
	}
	if ddl.Valid {
		return ddl.String, nil
	}
	return "", fmt.Errorf("DDL为空")
}

func loadSQLiteSchemaRoot(ctx context.Context, db *sql.DB) (map[string]interface{}, error) {
	rows, err := db.QueryContext(ctx, `
SELECT type, COUNT(*)
FROM sqlite_master
WHERE type IN ('table','view','index') AND name NOT LIKE 'sqlite_%'
GROUP BY type`)
	if err != nil {
		return nil, fmt.Errorf("读取SQLite目录数量失败: %s", err.Error())
	}
	defer rows.Close()
	counts := map[string]int64{"table": 0, "view": 0, "index": 0}
	for rows.Next() {
		var typ string
		var count int64
		if err := rows.Scan(&typ, &count); err != nil {
			return nil, err
		}
		counts[typ] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"database": "",
		"schemas": []map[string]interface{}{
			{
				"name": "main",
				"type": "schema",
				"folders": []map[string]interface{}{
					{"name": "tables", "type": "tables", "label": "Tables", "count": counts["table"]},
					{"name": "views", "type": "views", "label": "Views", "count": counts["view"]},
					{"name": "indexes", "type": "indexes", "label": "Indexes", "count": counts["index"]},
				},
			},
		},
	}, nil
}

func loadSQLiteSchemaFolder(ctx context.Context, db *sql.DB, folderType string) (map[string]interface{}, error) {
	switch folderType {
	case "tables", "table":
		items, err := loadSQLiteObjectList(ctx, db, "table")
		return map[string]interface{}{"folder_type": "tables", "items": items, "tables": items}, err
	case "views", "view":
		items, err := loadSQLiteObjectList(ctx, db, "view")
		return map[string]interface{}{"folder_type": "views", "items": items, "views": items}, err
	case "indexes", "index":
		items, err := loadSQLiteIndexes(ctx, db, map[string]map[string]interface{}{})
		return map[string]interface{}{"folder_type": "indexes", "items": items, "indexes": items}, err
	default:
		return nil, fmt.Errorf("folder_type 仅支持 tables/views/indexes")
	}
}

func loadSQLiteObjectList(ctx context.Context, db *sql.DB, objectType string) ([]map[string]interface{}, error) {
	rows, err := db.QueryContext(ctx, `
SELECT name, type
FROM sqlite_master
WHERE type = ? AND name NOT LIKE 'sqlite_%'
ORDER BY name`, objectType)
	if err != nil {
		return nil, fmt.Errorf("读取SQLite对象列表失败: %s", err.Error())
	}
	defer rows.Close()
	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		var name string
		var typ string
		if err := rows.Scan(&name, &typ); err != nil {
			return nil, err
		}
		items = append(items, map[string]interface{}{
			"name":           name,
			"type":           typ,
			"columns":        []map[string]interface{}{},
			"indexes":        []map[string]interface{}{},
			"columns_loaded": false,
		})
	}
	return items, rows.Err()
}

func loadSQLiteSchemaObject(ctx context.Context, db *sql.DB, objectType string, objectName string, tableName string) (map[string]interface{}, error) {
	if objectName == "" {
		return nil, fmt.Errorf("object_name 不能为空")
	}
	if objectType == "index" {
		detail, err := loadSQLiteObjectDetail(ctx, db, objectType, objectName, tableName)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"object": detail}, nil
	}
	columns, err := loadSQLiteColumnsForObject(ctx, db, objectName)
	if err != nil {
		return nil, err
	}
	indexes, err := loadSQLiteIndexesForTable(ctx, db, objectName)
	if err != nil {
		return nil, err
	}
	kind := "table"
	if objectType == "view" {
		kind = "view"
	}
	return map[string]interface{}{
		"object": map[string]interface{}{
			"name":           objectName,
			"type":           kind,
			"columns":        columns,
			"indexes":        indexes,
			"columns_loaded": true,
		},
	}, nil
}

func loadSQLiteSchemaSearch(ctx context.Context, db *sql.DB, query string) (map[string]interface{}, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return loadSQLiteSchemaRoot(ctx, db)
	}
	like := "%" + query + "%"
	rows, err := db.QueryContext(ctx, `
SELECT name, type
FROM sqlite_master
WHERE type IN ('table','view') AND name NOT LIKE 'sqlite_%' AND name LIKE ?
ORDER BY type, name`, like)
	if err != nil {
		return nil, fmt.Errorf("搜索SQLite对象失败: %s", err.Error())
	}
	defer rows.Close()
	tableList := make([]map[string]interface{}, 0)
	viewList := make([]map[string]interface{}, 0)
	tableIndex := map[string]map[string]interface{}{}
	for rows.Next() {
		var name string
		var typ string
		if err := rows.Scan(&name, &typ); err != nil {
			return nil, err
		}
		item := map[string]interface{}{"name": name, "type": typ, "columns": []map[string]interface{}{}, "indexes": []map[string]interface{}{}, "columns_loaded": false}
		if typ == "view" {
			viewList = append(viewList, item)
		} else {
			tableList = append(tableList, item)
		}
		tableIndex[name] = item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	allObjects, err := loadSQLiteObjectList(ctx, db, "table")
	if err != nil {
		return nil, err
	}
	allViews, err := loadSQLiteObjectList(ctx, db, "view")
	if err != nil {
		return nil, err
	}
	allObjects = append(allObjects, allViews...)
	lowerQuery := strings.ToLower(query)
	for _, object := range allObjects {
		name := stringValue(object["name"])
		columns, err := loadSQLiteColumnsForObject(ctx, db, name)
		if err != nil {
			continue
		}
		if item := tableIndex[name]; item != nil && strings.Contains(strings.ToLower(name), lowerQuery) {
			item["columns"] = columns
			item["columns_loaded"] = true
			continue
		}
		matchingColumns := make([]map[string]interface{}, 0)
		for _, column := range columns {
			if strings.Contains(strings.ToLower(stringValue(column["name"])), lowerQuery) ||
				strings.Contains(strings.ToLower(stringValue(column["type"])), lowerQuery) {
				matchingColumns = append(matchingColumns, column)
			}
		}
		if len(matchingColumns) <= 0 {
			continue
		}
		item := tableIndex[name]
		if item == nil {
			item = object
			tableIndex[name] = item
			if stringValue(item["type"]) == "view" {
				viewList = append(viewList, item)
			} else {
				tableList = append(tableList, item)
			}
		}
		item["columns"] = matchingColumns
		item["columns_loaded"] = true
	}
	indexList, err := loadSQLiteSearchIndexes(ctx, db, query, tableIndex)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"database": "",
		"tables":   tableList,
		"views":    viewList,
		"indexes":  indexList,
		"schemas": []map[string]interface{}{
			{
				"name":    "main",
				"type":    "schema",
				"tables":  tableList,
				"views":   viewList,
				"indexes": indexList,
			},
		},
	}, nil
}

func loadSQLiteSchema(ctx context.Context, db *sql.DB, req webSQLSchemaRequest) (map[string]interface{}, error) {
	switch req.Scope {
	case "root":
		return loadSQLiteSchemaRoot(ctx, db)
	case "folder":
		return loadSQLiteSchemaFolder(ctx, db, req.FolderType)
	case "object":
		return loadSQLiteSchemaObject(ctx, db, req.ObjectType, req.ObjectName, req.TableName)
	case "search":
		return loadSQLiteSchemaSearch(ctx, db, req.Query)
	case "completion":
		return loadSQLiteSchema(ctx, db, webSQLSchemaRequest{})
	}

	tableRows, err := db.QueryContext(ctx, `
SELECT name, type
FROM sqlite_master
WHERE type IN ('table','view') AND name NOT LIKE 'sqlite_%'
ORDER BY type, name`)
	if err != nil {
		return nil, fmt.Errorf("读取SQLite表列表失败: %s", err.Error())
	}

	tableList := make([]map[string]interface{}, 0)
	viewList := make([]map[string]interface{}, 0)
	tableIndex := map[string]map[string]interface{}{}
	for tableRows.Next() {
		var tableName string
		var tableType string
		if err := tableRows.Scan(&tableName, &tableType); err != nil {
			return nil, err
		}
		item := map[string]interface{}{
			"name":    tableName,
			"type":    tableType,
			"columns": []map[string]interface{}{},
			"indexes": []map[string]interface{}{},
		}
		if tableType == "view" {
			viewList = append(viewList, item)
		} else {
			tableList = append(tableList, item)
		}
		tableIndex[tableName] = item
	}
	if err := tableRows.Err(); err != nil {
		tableRows.Close()
		return nil, err
	}
	tableRows.Close()

	allObjects := make([]map[string]interface{}, 0, len(tableList)+len(viewList))
	allObjects = append(allObjects, tableList...)
	allObjects = append(allObjects, viewList...)
	for _, table := range allObjects {
		tableName := stringValue(table["name"])
		columnRows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", quoteSQLiteIdentifier(tableName)))
		if err != nil {
			table["column_error"] = err.Error()
			continue
		}
		columns := make([]map[string]interface{}, 0)
		for columnRows.Next() {
			var cid int
			var columnName string
			var columnType string
			var notNull int
			var defaultValue sql.NullString
			var pk int
			if err := columnRows.Scan(&cid, &columnName, &columnType, &notNull, &defaultValue, &pk); err != nil {
				columnRows.Close()
				return nil, err
			}
			columns = append(columns, map[string]interface{}{
				"name":     columnName,
				"type":     columnType,
				"nullable": notNull == 0,
				"key":      map[bool]string{true: "PRI", false: ""}[pk > 0],
				"default":  defaultValue.String,
				"ordinal":  cid + 1,
			})
		}
		if err := columnRows.Err(); err != nil {
			columnRows.Close()
			return nil, err
		}
		columnRows.Close()
		table["columns"] = columns
	}
	indexList, err := loadSQLiteIndexes(ctx, db, tableIndex)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"database": "",
		"tables":   tableList,
		"views":    viewList,
		"indexes":  indexList,
		"schemas": []map[string]interface{}{
			{
				"name":    "main",
				"type":    "schema",
				"tables":  tableList,
				"views":   viewList,
				"indexes": indexList,
			},
		},
	}, nil
}

func loadMySQLIndexes(ctx context.Context, db *sql.DB, databaseName string, tableIndex map[string]map[string]interface{}) ([]map[string]interface{}, error) {
	indexRows, err := db.QueryContext(ctx, `
SELECT table_name, index_name, non_unique, seq_in_index, column_name, index_type
FROM information_schema.statistics
WHERE table_schema = ?
ORDER BY table_name, index_name, seq_in_index`, databaseName)
	if err != nil {
		return nil, fmt.Errorf("读取索引列表失败: %s", err.Error())
	}
	defer indexRows.Close()

	indexList := make([]map[string]interface{}, 0)
	indexMap := map[string]map[string]interface{}{}
	for indexRows.Next() {
		var tableName string
		var indexName string
		var nonUnique int
		var seqInIndex int
		var columnName sql.NullString
		var indexType sql.NullString
		if err := indexRows.Scan(&tableName, &indexName, &nonUnique, &seqInIndex, &columnName, &indexType); err != nil {
			return nil, err
		}
		key := tableName + "\x00" + indexName
		item := indexMap[key]
		if item == nil {
			item = map[string]interface{}{
				"name":       indexName,
				"type":       "index",
				"table_name": tableName,
				"unique":     nonUnique == 0,
				"index_type": indexType.String,
				"columns":    []string{},
			}
			indexMap[key] = item
			indexList = append(indexList, item)
			if table := tableIndex[tableName]; table != nil {
				indexes, _ := table["indexes"].([]map[string]interface{})
				indexes = append(indexes, item)
				table["indexes"] = indexes
			}
		}
		columns, _ := item["columns"].([]string)
		if columnName.Valid {
			columns = append(columns, columnName.String)
		}
		item["columns"] = columns
		item["seq_in_index"] = seqInIndex
	}
	if err := indexRows.Err(); err != nil {
		return nil, err
	}
	return indexList, nil
}

func loadSQLiteIndexes(ctx context.Context, db *sql.DB, tableIndex map[string]map[string]interface{}) ([]map[string]interface{}, error) {
	rows, err := db.QueryContext(ctx, `
SELECT name, tbl_name, COALESCE(sql, '')
FROM sqlite_master
WHERE type = 'index' AND name NOT LIKE 'sqlite_%'
ORDER BY tbl_name, name`)
	if err != nil {
		return nil, fmt.Errorf("读取SQLite索引列表失败: %s", err.Error())
	}

	type sqliteIndexRow struct {
		name      string
		tableName string
		ddl       string
	}
	rawIndexes := make([]sqliteIndexRow, 0)
	for rows.Next() {
		var item sqliteIndexRow
		if err := rows.Scan(&item.name, &item.tableName, &item.ddl); err != nil {
			rows.Close()
			return nil, err
		}
		rawIndexes = append(rawIndexes, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	indexList := make([]map[string]interface{}, 0, len(rawIndexes))
	for _, rawIndex := range rawIndexes {
		item := map[string]interface{}{
			"name":       rawIndex.name,
			"type":       "index",
			"table_name": rawIndex.tableName,
			"ddl":        rawIndex.ddl,
			"columns":    []string{},
		}
		columns, err := loadSQLiteIndexColumns(ctx, db, rawIndex.name)
		if err == nil {
			item["columns"] = columns
		} else {
			item["column_error"] = err.Error()
		}
		indexList = append(indexList, item)
		if table := tableIndex[rawIndex.tableName]; table != nil {
			indexes, _ := table["indexes"].([]map[string]interface{})
			indexes = append(indexes, item)
			table["indexes"] = indexes
		}
	}
	return indexList, nil
}

func loadSQLiteIndexesForTable(ctx context.Context, db *sql.DB, tableName string) ([]map[string]interface{}, error) {
	rows, err := db.QueryContext(ctx, `
SELECT name, tbl_name, COALESCE(sql, '')
FROM sqlite_master
WHERE type = 'index' AND name NOT LIKE 'sqlite_%' AND tbl_name = ?
ORDER BY name`, tableName)
	if err != nil {
		return nil, fmt.Errorf("读取SQLite表索引失败: %s", err.Error())
	}
	type sqliteIndexRow struct {
		name      string
		tableName string
		ddl       string
	}
	rawIndexes := make([]sqliteIndexRow, 0)
	for rows.Next() {
		var item sqliteIndexRow
		if err := rows.Scan(&item.name, &item.tableName, &item.ddl); err != nil {
			rows.Close()
			return nil, err
		}
		rawIndexes = append(rawIndexes, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	indexList := make([]map[string]interface{}, 0, len(rawIndexes))
	for _, rawIndex := range rawIndexes {
		columns, err := loadSQLiteIndexColumns(ctx, db, rawIndex.name)
		if err != nil {
			return nil, err
		}
		indexList = append(indexList, map[string]interface{}{
			"name":       rawIndex.name,
			"type":       "index",
			"table_name": rawIndex.tableName,
			"ddl":        rawIndex.ddl,
			"columns":    columns,
		})
	}
	return indexList, nil
}

func loadSQLiteSearchIndexes(ctx context.Context, db *sql.DB, query string, tableIndex map[string]map[string]interface{}) ([]map[string]interface{}, error) {
	indexes, err := loadSQLiteIndexes(ctx, db, tableIndex)
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(query)
	filtered := make([]map[string]interface{}, 0)
	for _, indexItem := range indexes {
		columns, _ := indexItem["columns"].([]string)
		text := strings.ToLower(stringValue(indexItem["name"]) + " " + stringValue(indexItem["table_name"]) + " " + strings.Join(columns, " "))
		if strings.Contains(text, query) {
			filtered = append(filtered, indexItem)
		}
	}
	return filtered, nil
}

func loadSQLiteColumnsForObject(ctx context.Context, db *sql.DB, objectName string) ([]map[string]interface{}, error) {
	columnRows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", quoteSQLiteIdentifier(objectName)))
	if err != nil {
		return nil, err
	}
	defer columnRows.Close()
	columns := make([]map[string]interface{}, 0)
	for columnRows.Next() {
		var cid int
		var columnName string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := columnRows.Scan(&cid, &columnName, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, map[string]interface{}{
			"name":     columnName,
			"type":     columnType,
			"nullable": notNull == 0,
			"key":      map[bool]string{true: "PRI", false: ""}[pk > 0],
			"default":  defaultValue.String,
			"ordinal":  cid + 1,
		})
	}
	return columns, columnRows.Err()
}

func loadSQLiteIndexColumns(ctx context.Context, db *sql.DB, indexName string) ([]string, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_info(%s)", quoteSQLiteIdentifier(indexName)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make([]string, 0)
	for rows.Next() {
		var seqno int
		var cid int
		var name sql.NullString
		if err := rows.Scan(&seqno, &cid, &name); err != nil {
			return nil, err
		}
		if name.Valid {
			columns = append(columns, name.String)
		}
	}
	return columns, rows.Err()
}

func loadWebSQLObjectDetail(ctx context.Context, db *sql.DB, driverName string, databaseName string, objectType string, objectName string, tableName string) (map[string]interface{}, error) {
	if objectName == "" {
		return nil, fmt.Errorf("object_name 不能为空")
	}
	switch driverName {
	case "mysql":
		return loadMySQLObjectDetail(ctx, db, databaseName, objectType, objectName, tableName)
	case "oracle":
		return loadOracleObjectDetail(ctx, db, databaseName, objectType, objectName, tableName)
	case "sqlite":
		return loadSQLiteObjectDetail(ctx, db, objectType, objectName, tableName)
	default:
		return nil, fmt.Errorf("暂不支持数据库类型: %s", driverName)
	}
}

func loadMySQLObjectDetail(ctx context.Context, db *sql.DB, databaseName string, objectType string, objectName string, tableName string) (map[string]interface{}, error) {
	databaseName, err := resolveMySQLDatabase(ctx, db, databaseName)
	if err != nil {
		return nil, err
	}
	if databaseName == "" {
		return nil, fmt.Errorf("database 不能为空")
	}
	if objectType == "index" {
		return loadMySQLIndexDetail(ctx, db, databaseName, objectName, tableName)
	}
	columns, err := loadMySQLColumnsForTable(ctx, db, databaseName, objectName)
	if err != nil {
		return nil, err
	}
	qualifiedObjectName := quoteMySQLQualifiedIdentifier(databaseName, objectName)
	sqlText := "SHOW CREATE TABLE " + qualifiedObjectName
	if objectType == "view" {
		sqlText = "SHOW CREATE VIEW " + qualifiedObjectName
	}
	rows, err := db.QueryContext(ctx, sqlText)
	if err != nil {
		return nil, fmt.Errorf("读取对象DDL失败: %s", err.Error())
	}
	defer rows.Close()
	ddl, err := readWebSQLCreateStatement(rows)
	if err != nil {
		return nil, err
	}
	formattedDDL := formatWebSQLDDL(ddl)
	return map[string]interface{}{
		"database":    databaseName,
		"name":        objectName,
		"type":        objectType,
		"columns":     columns,
		"ddl":         formattedDDL,
		"sql":         formattedDDL,
		"raw_ddl":     ddl,
		"object_type": objectType,
	}, nil
}

func loadMySQLIndexDetail(ctx context.Context, db *sql.DB, databaseName string, indexName string, tableName string) (map[string]interface{}, error) {
	if tableName == "" {
		return nil, fmt.Errorf("table_name 不能为空")
	}
	rows, err := db.QueryContext(ctx, `
SELECT non_unique, seq_in_index, column_name, index_type
FROM information_schema.statistics
WHERE table_schema = ? AND table_name = ? AND index_name = ?
ORDER BY seq_in_index`, databaseName, tableName, indexName)
	if err != nil {
		return nil, fmt.Errorf("读取索引DDL失败: %s", err.Error())
	}
	defer rows.Close()

	columns := make([]string, 0)
	unique := false
	indexType := ""
	for rows.Next() {
		var nonUnique int
		var seqInIndex int
		var columnName sql.NullString
		var indexTypeValue sql.NullString
		if err := rows.Scan(&nonUnique, &seqInIndex, &columnName, &indexTypeValue); err != nil {
			return nil, err
		}
		unique = nonUnique == 0
		if indexTypeValue.Valid {
			indexType = indexTypeValue.String
		}
		if columnName.Valid {
			columns = append(columns, quoteMySQLIdentifier(columnName.String))
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("索引不存在: %s", indexName)
	}
	prefix := "CREATE INDEX "
	if unique {
		prefix = "CREATE UNIQUE INDEX "
	}
	qualifiedTableName := quoteMySQLQualifiedIdentifier(databaseName, tableName)
	if strings.EqualFold(indexName, "PRIMARY") {
		prefix = "ALTER TABLE " + qualifiedTableName + " ADD PRIMARY KEY "
		ddl := prefix + "(" + strings.Join(columns, ", ") + ");"
		formattedDDL := formatWebSQLDDL(ddl)
		return map[string]interface{}{"database": databaseName, "name": indexName, "type": "index", "table_name": tableName, "ddl": formattedDDL, "sql": formattedDDL, "raw_ddl": ddl}, nil
	}
	using := ""
	if indexType != "" {
		using = " USING " + indexType
	}
	ddl := prefix + quoteMySQLIdentifier(indexName) + " ON " + qualifiedTableName + " (" + strings.Join(columns, ", ") + ")" + using + ";"
	formattedDDL := formatWebSQLDDL(ddl)
	return map[string]interface{}{"database": databaseName, "name": indexName, "type": "index", "table_name": tableName, "ddl": formattedDDL, "sql": formattedDDL, "raw_ddl": ddl}, nil
}

func loadSQLiteObjectDetail(ctx context.Context, db *sql.DB, objectType string, objectName string, tableName string) (map[string]interface{}, error) {
	rows, err := db.QueryContext(ctx, `
SELECT name, type, tbl_name, COALESCE(sql, '')
FROM sqlite_master
WHERE name = ? AND (? = '' OR type = ?)
LIMIT 1`, objectName, objectType, objectType)
	if err != nil {
		return nil, fmt.Errorf("读取SQLite对象SQL失败: %s", err.Error())
	}
	if !rows.Next() {
		rows.Close()
		return nil, fmt.Errorf("对象不存在: %s", objectName)
	}
	var name string
	var typ string
	var tblName string
	var ddl string
	if err := rows.Scan(&name, &typ, &tblName, &ddl); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if tableName == "" {
		tableName = tblName
	}
	columns := []map[string]interface{}{}
	if typ == "table" || typ == "view" {
		loadedColumns, err := loadSQLiteColumnsForObject(ctx, db, name)
		if err != nil {
			return nil, err
		}
		columns = loadedColumns
	}
	formattedDDL := formatWebSQLDDL(ddl)
	return map[string]interface{}{
		"name":        name,
		"type":        typ,
		"table_name":  tableName,
		"columns":     columns,
		"ddl":         formattedDDL,
		"sql":         formattedDDL,
		"raw_ddl":     ddl,
		"object_type": objectType,
	}, nil
}

func readWebSQLCreateStatement(rows *sql.Rows) (string, error) {
	columns, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("读取DDL字段失败: %s", err.Error())
	}
	rawValues := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(columns))
	for i := range rawValues {
		scanArgs[i] = &rawValues[i]
	}
	if !rows.Next() {
		return "", fmt.Errorf("DDL为空")
	}
	if err := rows.Scan(scanArgs...); err != nil {
		return "", fmt.Errorf("读取DDL失败: %s", err.Error())
	}
	for i, name := range columns {
		if i == 0 {
			continue
		}
		if strings.Contains(strings.ToLower(name), "create") && rawValues[i] != nil {
			return string(rawValues[i]), nil
		}
	}
	if len(rawValues) > 1 && rawValues[1] != nil {
		return string(rawValues[1]), nil
	}
	return "", fmt.Errorf("DDL为空")
}

func firstSQLKeyword(sqlText string) string {
	text := strings.TrimSpace(sqlText)
	for {
		if strings.HasPrefix(text, "--") {
			idx := strings.IndexAny(text, "\r\n")
			if idx < 0 {
				return ""
			}
			text = strings.TrimSpace(text[idx+1:])
			continue
		}
		if strings.HasPrefix(text, "/*") {
			idx := strings.Index(text, "*/")
			if idx < 0 {
				return ""
			}
			text = strings.TrimSpace(text[idx+2:])
			continue
		}
		break
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}
	return strings.ToLower(strings.Trim(fields[0], " \t\r\n;"))
}

func isWebSQLQuery(statementType string) bool {
	switch strings.ToLower(statementType) {
	case "select", "show", "desc", "describe", "explain", "with", "pragma":
		return true
	default:
		return false
	}
}

func formatWebSQLDDL(ddl string) string {
	text := compactSQLWhitespace(ddl)
	if text == "" {
		return ""
	}
	upper := strings.ToUpper(text)
	if strings.HasPrefix(upper, "CREATE TABLE ") || strings.HasPrefix(upper, "CREATE TEMPORARY TABLE ") {
		return formatWebSQLDDLWithColumns(text)
	}
	if strings.HasPrefix(upper, "CREATE INDEX ") || strings.HasPrefix(upper, "CREATE UNIQUE INDEX ") {
		return formatWebSQLDDLWithColumns(text)
	}
	if strings.HasPrefix(upper, "ALTER TABLE ") {
		return formatWebSQLGenericDDL(text)
	}
	return ensureSQLSemicolon(uppercaseWebSQLKeywords(text))
}

func formatWebSQLDDLWithColumns(ddl string) string {
	open := findTopLevelChar(ddl, '(')
	if open < 0 {
		return ensureSQLSemicolon(uppercaseWebSQLKeywords(ddl))
	}
	close := findMatchingParen(ddl, open)
	if close < 0 {
		return ensureSQLSemicolon(uppercaseWebSQLKeywords(ddl))
	}
	prefix := strings.TrimSpace(ddl[:open])
	body := strings.TrimSpace(ddl[open+1 : close])
	suffix := strings.TrimSpace(ddl[close+1:])
	parts := splitTopLevelSQLList(body)
	if len(parts) == 0 {
		return ensureSQLSemicolon(uppercaseWebSQLKeywords(ddl))
	}

	var b strings.Builder
	b.WriteString(uppercaseWebSQLKeywords(prefix))
	b.WriteString(" (\n")
	for i, part := range parts {
		b.WriteString("  ")
		b.WriteString(uppercaseWebSQLKeywords(compactSQLWhitespace(part)))
		if i < len(parts)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString(")")
	if suffix != "" && suffix != ";" {
		b.WriteString("\n")
		b.WriteString(uppercaseWebSQLKeywords(strings.TrimSuffix(suffix, ";")))
	}
	return ensureSQLSemicolon(b.String())
}

func formatWebSQLGenericDDL(ddl string) string {
	text := uppercaseWebSQLKeywords(ddl)
	replacements := []struct {
		old string
		new string
	}{
		{" ADD ", "\nADD "},
		{" DROP ", "\nDROP "},
		{" RENAME ", "\nRENAME "},
		{" AS SELECT ", " AS\nSELECT "},
		{" FROM ", "\nFROM "},
		{" WHERE ", "\nWHERE "},
		{" GROUP BY ", "\nGROUP BY "},
		{" ORDER BY ", "\nORDER BY "},
		{" LIMIT ", "\nLIMIT "},
	}
	for _, item := range replacements {
		text = strings.ReplaceAll(text, item.old, item.new)
	}
	return ensureSQLSemicolon(text)
}

func compactSQLWhitespace(input string) string {
	input = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(input, "\r\n", "\n"), "\r", "\n"))
	var b strings.Builder
	b.Grow(len(input))
	inSingle := false
	inDouble := false
	inBacktick := false
	inBracket := false
	lastSpace := false
	for i := 0; i < len(input); i++ {
		ch := input[i]
		if inSingle {
			b.WriteByte(ch)
			if ch == '\'' {
				if i+1 < len(input) && input[i+1] == '\'' {
					i++
					b.WriteByte(input[i])
				} else {
					inSingle = false
				}
			}
			lastSpace = false
			continue
		}
		if inDouble {
			b.WriteByte(ch)
			if ch == '"' {
				if i+1 < len(input) && input[i+1] == '"' {
					i++
					b.WriteByte(input[i])
				} else {
					inDouble = false
				}
			}
			lastSpace = false
			continue
		}
		if inBacktick {
			b.WriteByte(ch)
			if ch == '`' {
				if i+1 < len(input) && input[i+1] == '`' {
					i++
					b.WriteByte(input[i])
				} else {
					inBacktick = false
				}
			}
			lastSpace = false
			continue
		}
		if inBracket {
			b.WriteByte(ch)
			if ch == ']' {
				inBracket = false
			}
			lastSpace = false
			continue
		}
		switch {
		case ch == '\'':
			inSingle = true
			b.WriteByte(ch)
			lastSpace = false
		case ch == '"':
			inDouble = true
			b.WriteByte(ch)
			lastSpace = false
		case ch == '`':
			inBacktick = true
			b.WriteByte(ch)
			lastSpace = false
		case ch == '[':
			inBracket = true
			b.WriteByte(ch)
			lastSpace = false
		case ch == ' ' || ch == '\n' || ch == '\t' || ch == '\f':
			if !lastSpace && b.Len() > 0 {
				b.WriteByte(' ')
				lastSpace = true
			}
		default:
			b.WriteByte(ch)
			lastSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}

func findTopLevelChar(text string, target byte) int {
	return scanSQLTopLevel(text, 0, func(i int, ch byte, depth int) bool {
		return depth == 0 && ch == target
	})
}

func findMatchingParen(text string, open int) int {
	depth := 0
	return scanSQLTopLevel(text, open, func(i int, ch byte, _ int) bool {
		if ch == '(' {
			depth++
			return false
		}
		if ch == ')' {
			depth--
			return depth == 0
		}
		return false
	})
}

func splitTopLevelSQLList(text string) []string {
	parts := make([]string, 0)
	start := 0
	depth := 0
	_ = scanSQLTopLevel(text, 0, func(i int, ch byte, _ int) bool {
		switch ch {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				if part := strings.TrimSpace(text[start:i]); part != "" {
					parts = append(parts, part)
				}
				start = i + 1
			}
		}
		return false
	})
	if part := strings.TrimSpace(text[start:]); part != "" {
		parts = append(parts, part)
	}
	return parts
}

func scanSQLTopLevel(text string, start int, visit func(i int, ch byte, depth int) bool) int {
	inSingle := false
	inDouble := false
	inBacktick := false
	inBracket := false
	depth := 0
	for i := start; i < len(text); i++ {
		ch := text[i]
		if inSingle {
			if ch == '\'' {
				if i+1 < len(text) && text[i+1] == '\'' {
					i++
				} else {
					inSingle = false
				}
			}
			continue
		}
		if inDouble {
			if ch == '"' {
				if i+1 < len(text) && text[i+1] == '"' {
					i++
				} else {
					inDouble = false
				}
			}
			continue
		}
		if inBacktick {
			if ch == '`' {
				if i+1 < len(text) && text[i+1] == '`' {
					i++
				} else {
					inBacktick = false
				}
			}
			continue
		}
		if inBracket {
			if ch == ']' {
				inBracket = false
			}
			continue
		}
		switch ch {
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case '`':
			inBacktick = true
		case '[':
			inBracket = true
		case '(':
			if visit(i, ch, depth) {
				return i
			}
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
			if visit(i, ch, depth) {
				return i
			}
		default:
			if visit(i, ch, depth) {
				return i
			}
		}
	}
	return -1
}

func uppercaseWebSQLKeywords(text string) string {
	keywords := map[string]struct{}{
		"add": {}, "alter": {}, "as": {}, "autoincrement": {}, "by": {}, "cascade": {}, "check": {},
		"collate": {}, "constraint": {}, "create": {}, "default": {}, "delete": {}, "drop": {},
		"exists": {}, "foreign": {}, "from": {}, "generated": {}, "group": {}, "if": {}, "index": {},
		"key": {}, "limit": {}, "not": {}, "null": {}, "on": {}, "order": {}, "primary": {},
		"references": {}, "rename": {}, "restrict": {}, "rowid": {}, "select": {}, "set": {},
		"stored": {}, "table": {}, "temporary": {}, "unique": {}, "update": {}, "using": {},
		"values": {}, "view": {}, "virtual": {}, "where": {}, "without": {},
	}
	return rewriteSQLWords(text, func(word string) string {
		if _, ok := keywords[strings.ToLower(word)]; ok {
			return strings.ToUpper(word)
		}
		return word
	})
}

func rewriteSQLWords(text string, rewrite func(string) string) string {
	var b strings.Builder
	b.Grow(len(text))
	for i := 0; i < len(text); {
		ch := text[i]
		if ch == '\'' || ch == '"' || ch == '`' || ch == '[' {
			end := consumeSQLQuoted(text, i)
			b.WriteString(text[i:end])
			i = end
			continue
		}
		if isSQLWordByte(ch) {
			start := i
			for i < len(text) && isSQLWordByte(text[i]) {
				i++
			}
			b.WriteString(rewrite(text[start:i]))
			continue
		}
		b.WriteByte(ch)
		i++
	}
	return b.String()
}

func consumeSQLQuoted(text string, start int) int {
	quote := text[start]
	endQuote := quote
	if quote == '[' {
		endQuote = ']'
	}
	for i := start + 1; i < len(text); i++ {
		if text[i] != endQuote {
			continue
		}
		if quote != '[' && i+1 < len(text) && text[i+1] == endQuote {
			i++
			continue
		}
		return i + 1
	}
	return len(text)
}

func isSQLWordByte(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}

func ensureSQLSemicolon(text string) string {
	text = strings.TrimSpace(text)
	if text == "" || strings.HasSuffix(text, ";") {
		return text
	}
	return text + ";"
}

func normalizeWebSQLExecutionSQL(driverName string, sqlText string) string {
	text := strings.TrimSpace(sqlText)
	if normalizeWebSQLDriver(driverName) != "oracle" {
		return text
	}
	statementType := firstSQLKeyword(text)
	switch statementType {
	case "begin", "declare":
		return text
	}
	for strings.HasSuffix(text, ";") {
		text = strings.TrimSpace(strings.TrimSuffix(text, ";"))
	}
	return text
}

func quoteSQLiteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func quoteMySQLIdentifier(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

func quoteMySQLQualifiedIdentifier(databaseName string, objectName string) string {
	databaseName = strings.TrimSpace(databaseName)
	if databaseName == "" {
		return quoteMySQLIdentifier(objectName)
	}
	return quoteMySQLIdentifier(databaseName) + "." + quoteMySQLIdentifier(objectName)
}

func webSQLIntValue(raw interface{}, fallback int) int {
	text := stringValue(raw)
	if text == "" {
		return fallback
	}
	var value int
	if _, err := fmt.Sscanf(text, "%d", &value); err != nil {
		return fallback
	}
	return value
}

func minWebSQLInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
