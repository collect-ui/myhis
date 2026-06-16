package base

const TableNameRegistration = "registration"

type Registration struct {
	RegistrationID string `gorm:"column:registration_id;primaryKey" json:"registration_id"`
	Username       string `gorm:"column:username;not null" json:"username"`
	Nick           string `gorm:"column:nick;not null" json:"nick"`
	Company        string `gorm:"column:company;not null" json:"company"`
	Phone          string `gorm:"column:phone;not null" json:"phone"`
	Status         string `gorm:"column:status;not null" json:"status"`
	Password       string `gorm:"column:password;not null" json:"password"`
	CreateTime     string `gorm:"column:create_time;not null" json:"create_time"`
}

// TableName Registration's table name
func (*Registration) TableName() string {
	return TableNameRegistration
}

func (*Registration) PrimaryKey() []string {
	return []string{"registration_id"}
}
