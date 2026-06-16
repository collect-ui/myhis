package devops

const TableNameWebSQLRecentSQL = "websql_recent_sql"

type WebSQLRecentSQL struct {
	RecentSQLID    string `gorm:"column:recent_sql_id;primaryKey" json:"recent_sql_id"`
	ProjectCode    string `gorm:"column:project_code;index" json:"project_code"`
	RecentSQLHash  string `gorm:"column:recent_sql_hash;uniqueIndex" json:"recent_sql_hash"`
	Driver         string `gorm:"column:driver;index" json:"driver"`
	ConnectionID   string `gorm:"column:connection_id;index" json:"connection_id"`
	ConnectionName string `gorm:"column:connection_name" json:"connection_name"`
	StatementType  string `gorm:"column:statement_type" json:"statement_type"`
	SQLText        string `gorm:"column:sql_text;type:text" json:"sql_text"`
	ExecuteStatus  string `gorm:"column:execute_status" json:"execute_status"`
	ErrorText      string `gorm:"column:error_text;type:text" json:"error_text"`
	RowCount       int64  `gorm:"column:row_count" json:"row_count"`
	RowsAffected   int64  `gorm:"column:rows_affected" json:"rows_affected"`
	DurationMs     int64  `gorm:"column:duration_ms" json:"duration_ms"`
	ExecuteCount   int    `gorm:"column:execute_count" json:"execute_count"`
	LastExecutedAt string `gorm:"column:last_executed_at;index" json:"last_executed_at"`
	CreateTime     string `gorm:"column:create_time" json:"create_time"`
	CreateUser     string `gorm:"column:create_user" json:"create_user"`
	ModifyTime     string `gorm:"column:modify_time" json:"modify_time"`
	ModifyUser     string `gorm:"column:modify_user" json:"modify_user"`
}

func (*WebSQLRecentSQL) TableName() string {
	return TableNameWebSQLRecentSQL
}

func (*WebSQLRecentSQL) PrimaryKey() []string {
	return []string{"recent_sql_id"}
}
