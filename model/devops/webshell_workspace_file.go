package devops

const TableNameWebshellWorkspaceFile = "webshell_workspace_file"

type WebshellWorkspaceFile struct {
	FileID      string  `gorm:"column:file_id;primaryKey" json:"file_id"`
	ProjectCode *string `gorm:"column:project_code" json:"project_code"`
	Name        *string `gorm:"column:name" json:"name"`
	Path        *string `gorm:"column:path" json:"path"`
	ParentID    *string `gorm:"column:parent_id" json:"parent_id"`
	IsDir       *string `gorm:"column:is_dir" json:"is_dir"`
	OrderIndex  *int32  `gorm:"column:order_index" json:"order_index"`
	IsDelete    *string `gorm:"column:is_delete" json:"is_delete"`
	CreateTime  *string `gorm:"column:create_time" json:"create_time"`
	CreateUser  *string `gorm:"column:create_user" json:"create_user"`
	ModifyTime  *string `gorm:"column:modify_time" json:"modify_time"`
	ModifyUser  *string `gorm:"column:modify_user" json:"modify_user"`
}

func (*WebshellWorkspaceFile) TableName() string {
	return TableNameWebshellWorkspaceFile
}

func (*WebshellWorkspaceFile) PrimaryKey() []string {
	return []string{"file_id"}
}
