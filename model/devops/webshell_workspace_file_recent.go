package devops

const TableNameWebshellWorkspaceFileRecent = "webshell_workspace_file_recent"

type WebshellWorkspaceFileRecent struct {
	RecentID     string  `gorm:"column:recent_id;primaryKey" json:"recent_id"`
	ProjectCode  *string `gorm:"column:project_code" json:"project_code"`
	FilePath     *string `gorm:"column:file_path" json:"file_path"`
	FileName     *string `gorm:"column:file_name" json:"file_name"`
	OpenCount    *int    `gorm:"column:open_count" json:"open_count"`
	LastOpenTime *string `gorm:"column:last_open_time" json:"last_open_time"`
	CreateTime   *string `gorm:"column:create_time" json:"create_time"`
	CreateUser   *string `gorm:"column:create_user" json:"create_user"`
	ModifyTime   *string `gorm:"column:modify_time" json:"modify_time"`
	ModifyUser   *string `gorm:"column:modify_user" json:"modify_user"`
}

func (*WebshellWorkspaceFileRecent) TableName() string {
	return TableNameWebshellWorkspaceFileRecent
}

func (*WebshellWorkspaceFileRecent) PrimaryKey() []string {
	return []string{"recent_id"}
}
