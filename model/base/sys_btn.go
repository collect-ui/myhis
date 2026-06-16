package base

const TableNameSysBtn = "sys_btn"

type SysBtn struct {
	SysBtnID   string `gorm:"column:sys_btn_id;primaryKey" json:"sys_btn_id"`
	Name       string `gorm:"column:name" json:"name"`
	Code       string `gorm:"column:code" json:"code"`
	MenuCode   string `gorm:"column:menu_code" json:"menu_code"`
	OrderIndex int    `gorm:"column:order_index" json:"order_index"`
}

// TableName SysBtn's table name
func (*SysBtn) TableName() string {
	return TableNameSysBtn
}

func (*SysBtn) PrimaryKey() []string {
	return []string{"sys_btn_id"}
}
