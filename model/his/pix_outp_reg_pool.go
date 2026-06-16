package his

const TableNamePixOutpRegPool = "pix_outp_reg_pool"

type PixOutpRegPool struct {
	RegPoolID        int64   `gorm:"column:reg_pool_id;primaryKey" json:"reg_pool_id"`
	MasterID         int64   `gorm:"column:master_id" json:"master_id"`
	OutpDurationCode *string `gorm:"column:outp_duration_code" json:"outp_duration_code"`
	SortNo           *int64  `gorm:"column:sort_no" json:"sort_no"`
	IsEnable         *string `gorm:"column:is_enable" json:"is_enable"`
	AppointmentFlag  *string `gorm:"column:appointment_flag" json:"appointment_flag"`
	RegFlag          *string `gorm:"column:reg_flag" json:"reg_flag"`
	IsInternet       *string `gorm:"column:is_internet" json:"is_internet"`
	TimeStamp        *int64  `gorm:"column:time_stamp" json:"time_stamp"`
}

func (*PixOutpRegPool) TableName() string {
	return TableNamePixOutpRegPool
}

func (*PixOutpRegPool) PrimaryKey() []string {
	return []string{"reg_pool_id"}
}
