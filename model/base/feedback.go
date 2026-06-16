package base

const TableNameFeedback = "feedback"

type Feedback struct {
	FeedbackID string `gorm:"column:feedback_id;primaryKey" json:"feedback_id"`
	Contact    string `gorm:"column:contact" json:"contact"`
	Type       string `gorm:"column:type" json:"type"`
	Title      string `gorm:"column:title" json:"title"`
	Content    string `gorm:"column:content" json:"content"`
	Status     string `gorm:"column:status" json:"status"`
	CreateTime string `gorm:"column:create_time" json:"create_time"`
	CreateUser string `gorm:"column:create_user" json:"create_user"`
}

// TableName Feedback's table name
func (*Feedback) TableName() string {
	return TableNameFeedback
}

func (*Feedback) PrimaryKey() []string {
	return []string{"feedback_id"}
}
