package base

const TableNameUserChangeHistory = "user_change_history"

// UserChangeHistory mapped from table <user_change_history>
type UserChangeHistory struct {
	ChangeID   string  `gorm:"column:change_id;primaryKey" json:"change_id"` // 变更ID
	Before     *string `gorm:"column:before" json:"before"`                  // 变更前值
	After      *string `gorm:"column:after" json:"after"`                    // 变更后值
	Field      *string `gorm:"column:field" json:"field"`                    // 变更字段
	Value      *string `gorm:"column:value" json:"value"`                    // 变更值
	Name       *string `gorm:"column:name" json:"name"`                      // 名称
	UserID     *string `gorm:"column:user_id" json:"user_id"`                // 用户ID
	CreateUser *string `gorm:"column:create_user" json:"create_user"`        // 创建人
	CreateTime *string `gorm:"column:create_time" json:"create_time"`        // 创建时间
}

// TableName UserChangeHistory's table name
func (*UserChangeHistory) TableName() string {
	return TableNameUserChangeHistory
}

// PrimaryKey 返回主键字段名
func (*UserChangeHistory) PrimaryKey() []string {
	return []string{"change_id"}
}
