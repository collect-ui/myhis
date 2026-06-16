package base

const TableNameAgentMessage = "agent_message"

type AgentMessage struct {
	AgentMessageID string `gorm:"column:agent_message_id;primaryKey" json:"agent_message_id"`
	AgentSessionID string `gorm:"column:agent_session_id" json:"agent_session_id"`
	AgentRunID     string `gorm:"column:agent_run_id" json:"agent_run_id"`
	Role           string `gorm:"column:role" json:"role"`
	MessageType    string `gorm:"column:message_type" json:"message_type"`
	ContentText    string `gorm:"column:content_text" json:"content_text"`
	ContentJSON    string `gorm:"column:content_json" json:"content_json"`
	SeqNo          int64  `gorm:"column:seq_no" json:"seq_no"`
	Source         string `gorm:"column:source" json:"source"`
	TokenCount     int64  `gorm:"column:token_count" json:"token_count"`
	Status         string `gorm:"column:status" json:"status"`
	CreateTime     string `gorm:"column:create_time" json:"create_time"`
	CreateUser     string `gorm:"column:create_user" json:"create_user"`
	IsDelete       string `gorm:"column:is_delete" json:"is_delete"`
}

func (*AgentMessage) TableName() string {
	return TableNameAgentMessage
}

func (*AgentMessage) PrimaryKey() []string {
	return []string{"agent_message_id"}
}
