package base

const TableNameFeedbackAttachment = "feedback_attachment"

type FeedbackAttachment struct {
	AttachmentID string `gorm:"column:attachment_id;primaryKey" json:"attachment_id"`
	FeedbackID   string `gorm:"column:feedback_id" json:"feedback_id"`
	Name         string `gorm:"column:name" json:"name"`
	URL          string `gorm:"column:url" json:"url"`
}

// TableName FeedbackAttachment's table name
func (*FeedbackAttachment) TableName() string {
	return TableNameFeedbackAttachment
}

func (*FeedbackAttachment) PrimaryKey() []string {
	return []string{"attachment_id"}
}
