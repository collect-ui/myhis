package devops

const TableNameWebSQLCommitEvent = "websql_commit_event"

type WebSQLCommitEvent struct {
	WebSQLCommitEventID string `gorm:"column:websql_commit_event_id;primaryKey" json:"websql_commit_event_id"`
	Status              string `gorm:"column:status" json:"status"`
	CommitMode          string `gorm:"column:commit_mode" json:"commit_mode"`
	Driver              string `gorm:"column:driver" json:"driver"`
	DatabaseName        string `gorm:"column:database_name" json:"database_name"`
	ConnectionID        string `gorm:"column:connection_id" json:"connection_id"`
	ConnectionName      string `gorm:"column:connection_name" json:"connection_name"`
	StatementType       string `gorm:"column:statement_type" json:"statement_type"`
	SQLText             string `gorm:"column:sql_text;type:text" json:"sql_text"`
	RowsAffected        int64  `gorm:"column:rows_affected" json:"rows_affected"`
	LastInsertID        int64  `gorm:"column:last_insert_id" json:"last_insert_id"`
	CreateTime          string `gorm:"column:create_time" json:"create_time"`
	ExpireTime          string `gorm:"column:expire_time" json:"expire_time"`
	FinishTime          string `gorm:"column:finish_time" json:"finish_time"`
	DurationMs          int64  `gorm:"column:duration_ms" json:"duration_ms"`
	ErrorText           string `gorm:"column:error_text;type:text" json:"error_text"`
}

func (*WebSQLCommitEvent) TableName() string {
	return TableNameWebSQLCommitEvent
}

func (*WebSQLCommitEvent) PrimaryKey() []string {
	return []string{"websql_commit_event_id"}
}
