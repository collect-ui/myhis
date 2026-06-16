package base

const TableNameSchemaPage = "schema_page"

type SchemaPage struct {
	SchemaPageID  string `gorm:"column:schema_page_id;primaryKey" json:"schema_page_id"`
	Code          string `gorm:"column:code" json:"code"`
	ParentID      string `gorm:"column:parent_id" json:"parent_id"`
	Index         int    `gorm:"column:index" json:"index"`
	Name          string `gorm:"column:name" json:"name"`
	CreateTime    string `gorm:"column:create_time" json:"create_time"`
	CreateUser    string `gorm:"column:create_user" json:"create_user"`
	OrderIndex    string `gorm:"column:order_index" json:"order_index"`
	BelongProject string `gorm:"column:belong_project" json:"belong_project"`
}

// TableName SchemaPage's table name
func (*SchemaPage) TableName() string {
	return TableNameSchemaPage
}

func (*SchemaPage) PrimaryKey() []string {
	return []string{"schema_page_id"}
}
