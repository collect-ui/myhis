package work_task

const TableNameWorkTaskIssue = "work_task_issue"

// WorkTaskIssue mapped from table <work_task_issue>
type WorkTaskIssue struct {
	TaskIssueID string  `gorm:"column:task_issue_id;primaryKey" json:"task_issue_id"` // ID
	Summary     *string `gorm:"column:summary" json:"summary"`                        // 摘要
	Description *string `gorm:"column:description" json:"description"`                // 描述
	Status      *string `gorm:"column:status" json:"status"`                          // 状态
	CreateTime  *string `gorm:"column:create_time" json:"create_time"`                // 创建时间
	CreateUser  *string `gorm:"column:create_user" json:"create_user"`                // 创建人
	ProjectCode *string `gorm:"column:project_code" json:"project_code"`              // 项目编码
	IssueKey    *string `gorm:"column:issue_key" json:"issue_key"`                    // 问题键
	Assignee    *string `gorm:"column:assignee" json:"assignee"`                      // 负责人
	IssueType   *string `gorm:"column:issue_type" json:"issue_type"`                  // 负责人
	Priority    *string `gorm:"column:priority" json:"priority"`                      // 负责人
	DueDate     *string `gorm:"column:due_date" json:"due_date"`                      // 负责人
	Version     *string `gorm:"column:version" json:"version"`                        // 负责人
}

// TableName WorkTaskIssue's table name
func (*WorkTaskIssue) TableName() string {
	return TableNameWorkTaskIssue
}

func (*WorkTaskIssue) PrimaryKey() []string {
	return []string{"task_issue_id"}
}
