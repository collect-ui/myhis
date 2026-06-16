package base

const TableNameAgentSession = "agent_session"

type AgentSession struct {
	AgentSessionID string `gorm:"column:agent_session_id;primaryKey" json:"agent_session_id"`
	SessionKey     string `gorm:"column:session_key" json:"session_key"`
	SceneCode      string `gorm:"column:scene_code" json:"scene_code"`
	Title          string `gorm:"column:title" json:"title"`
	Status         string `gorm:"column:status" json:"status"`
	UserID         string `gorm:"column:user_id" json:"user_id"`
	SystemPrompt   string `gorm:"column:system_prompt" json:"system_prompt"`
	Model          string `gorm:"column:model" json:"model"`
	ToolPolicyJSON string `gorm:"column:tool_policy_json" json:"tool_policy_json"`
	McpPolicyJSON  string `gorm:"column:mcp_policy_json" json:"mcp_policy_json"`
	ContextSummary string `gorm:"column:context_summary" json:"context_summary"`
	LastResponseID string `gorm:"column:last_response_id" json:"last_response_id"`
	LastActiveTime string `gorm:"column:last_active_time" json:"last_active_time"`
	ExpireTime     string `gorm:"column:expire_time" json:"expire_time"`
	CreateTime     string `gorm:"column:create_time" json:"create_time"`
	ModifyTime     string `gorm:"column:modify_time" json:"modify_time"`
	CreateUser     string `gorm:"column:create_user" json:"create_user"`
	ModifyUser     string `gorm:"column:modify_user" json:"modify_user"`
	IsDelete       string `gorm:"column:is_delete" json:"is_delete"`
}

func (*AgentSession) TableName() string {
	return TableNameAgentSession
}

func (*AgentSession) PrimaryKey() []string {
	return []string{"agent_session_id"}
}
