package devops

const TableNameWebshellWorkspaceProject = "webshell_workspace_project"

type WebshellWorkspaceProject struct {
	WebshellWorkspaceProjectID string  `gorm:"column:webshell_workspace_project_id;primaryKey" json:"webshell_workspace_project_id"`
	ProjectName                *string `gorm:"column:project_name" json:"project_name"`
	ProjectCode                *string `gorm:"column:project_code" json:"project_code"`
	OrderID                    *int    `gorm:"column:order_id" json:"order_id"`
	ShowHome                   *string `gorm:"column:show_home" json:"show_home"`
	ServerID                   *string `gorm:"column:server_id" json:"server_id"`
	ServerOsUsersID            *string `gorm:"column:server_os_users_id" json:"server_os_users_id"`
	ProjectDir                 *string `gorm:"column:project_dir" json:"project_dir"`
	GitRepoURL                 *string `gorm:"column:git_repo_url" json:"git_repo_url"`
	ProjectType                *string `gorm:"column:project_type" json:"project_type"`
	CollectSourceRoot          *string `gorm:"column:collect_source_root" json:"collect_source_root"`
	PythonPkgPath              *string `gorm:"column:python_pkg_path" json:"python_pkg_path"`
	ExcludeDirs                *string `gorm:"column:exclude_dirs" json:"exclude_dirs"`
	IsDelete                   *string `gorm:"column:is_delete" json:"is_delete"`
	CreateTime                 *string `gorm:"column:create_time" json:"create_time"`
	CreateUser                 *string `gorm:"column:create_user" json:"create_user"`
	ModifyTime                 *string `gorm:"column:modify_time" json:"modify_time"`
	ModifyUser                 *string `gorm:"column:modify_user" json:"modify_user"`
}

func (*WebshellWorkspaceProject) TableName() string {
	return TableNameWebshellWorkspaceProject
}

func (*WebshellWorkspaceProject) PrimaryKey() []string {
	return []string{"webshell_workspace_project_id"}
}
