package base

const TableNameBtnRoleIDList = "btn_role_id_list"

type BtnRoleIDList struct {
	BtnRoleID string `gorm:"column:btn_role_id;primaryKey" json:"btn_role_id"`
	RoleCode  string `gorm:"column:role_code" json:"role_code"`
	BtnCode   string `gorm:"column:btn_code" json:"btn_code"`
	MenuCode  string `gorm:"column:menu_code" json:"menu_code"`
}

// TableName BtnRoleIDList's table name
func (*BtnRoleIDList) TableName() string {
	return TableNameBtnRoleIDList
}

func (*BtnRoleIDList) PrimaryKey() []string {
	return []string{"btn_role_id"}
}
