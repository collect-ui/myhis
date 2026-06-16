package work_task

const TableNameWorkTaskVersion = "work_task_version"

// WorkTaskVersion mapped from table <work_task_version>
type WorkTaskVersion struct {
	WorkTaskVersionID string  `gorm:"column:work_task_version_id;primaryKey" json:"work_task_version_id"` // ID
	Name              *string `gorm:"column:name" json:"name"`                                            // 名称
	Code              *string `gorm:"column:code" json:"code"`                                            // 编码
	CreateTime        *string `gorm:"column:create_time" json:"create_time"`                              // 创建时间
	CreateUser        *string `gorm:"column:create_user" json:"create_user"`                              // 创建人
}

// TableName WorkTaskVersion's table name
func (*WorkTaskVersion) TableName() string {
	return TableNameWorkTaskVersion
}

func (*WorkTaskVersion) PrimaryKey() []string {
	return []string{"work_task_version_id"}
}
