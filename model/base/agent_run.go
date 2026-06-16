package base

const TableNameAgentRun = "agent_run"

type AgentRun struct {
	AgentRunID      string `gorm:"column:agent_run_id;primaryKey" json:"agent_run_id"`
	AgentSessionID  string `gorm:"column:agent_session_id" json:"agent_session_id"`
	RequestID       string `gorm:"column:request_id" json:"request_id"`
	TriggerType     string `gorm:"column:trigger_type" json:"trigger_type"`
	Status          string `gorm:"column:status" json:"status"`
	CurrentStep     string `gorm:"column:current_step" json:"current_step"`
	WorkerID        string `gorm:"column:worker_id" json:"worker_id"`
	LeaseExpireTime string `gorm:"column:lease_expire_time" json:"lease_expire_time"`
	HeartbeatTime   string `gorm:"column:heartbeat_time" json:"heartbeat_time"`
	RetryCount      int64  `gorm:"column:retry_count" json:"retry_count"`
	MaxRetry        int64  `gorm:"column:max_retry" json:"max_retry"`
	ErrorMsg        string `gorm:"column:error_msg" json:"error_msg"`
	RequestJSON     string `gorm:"column:request_json" json:"request_json"`
	ResultJSON      string `gorm:"column:result_json" json:"result_json"`
	StartedAt       string `gorm:"column:started_at" json:"started_at"`
	FinishedAt      string `gorm:"column:finished_at" json:"finished_at"`
	CreateTime      string `gorm:"column:create_time" json:"create_time"`
	ModifyTime      string `gorm:"column:modify_time" json:"modify_time"`
	CreateUser      string `gorm:"column:create_user" json:"create_user"`
	ModifyUser      string `gorm:"column:modify_user" json:"modify_user"`
	IsDelete        string `gorm:"column:is_delete" json:"is_delete"`
}

func (*AgentRun) TableName() string {
	return TableNameAgentRun
}

func (*AgentRun) PrimaryKey() []string {
	return []string{"agent_run_id"}
}
