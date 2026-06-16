package devops

const TableNameWebsqlFavorite = "websql_favorite"

type WebsqlFavorite struct {
	WebsqlFavoriteID string `gorm:"column:websql_favorite_id;primaryKey" json:"websql_favorite_id"`
	ProjectCode      string `gorm:"column:project_code;index" json:"project_code"`
	ItemType         string `gorm:"column:item_type;index" json:"item_type"`
	Name             string `gorm:"column:name" json:"name"`
	Path             string `gorm:"column:path;index" json:"path"`
	Folder           string `gorm:"column:folder;index" json:"folder"`
	SqlText          string `gorm:"column:sql_text;type:text" json:"sql_text"`
	ConnectionID     string `gorm:"column:connection_id" json:"connection_id"`
	ConnectionName   string `gorm:"column:connection_name" json:"connection_name"`
	Driver           string `gorm:"column:driver" json:"driver"`
	IsDelete         string `gorm:"column:is_delete;index" json:"is_delete"`
	CreateTime       string `gorm:"column:create_time" json:"create_time"`
	CreateUser       string `gorm:"column:create_user" json:"create_user"`
	ModifyTime       string `gorm:"column:modify_time" json:"modify_time"`
	ModifyUser       string `gorm:"column:modify_user" json:"modify_user"`
}

func (*WebsqlFavorite) TableName() string {
	return TableNameWebsqlFavorite
}

func (*WebsqlFavorite) PrimaryKey() []string {
	return []string{"websql_favorite_id"}
}
