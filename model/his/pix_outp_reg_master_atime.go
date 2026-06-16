package his

const TableNamePixOutpRegMasterAtime = "pix_outp_reg_master_atime"

type PixOutpRegMasterAtime struct {
	MasterAtimeID            int64   `gorm:"column:master_atime_id;primaryKey" json:"master_atime_id"`
	MasterID                 int64   `gorm:"column:master_id" json:"master_id"`
	SortNo                   *int64  `gorm:"column:sort_no" json:"sort_no"`
	TimeStatus               *string `gorm:"column:time_status" json:"time_status"`
	CurrentTimeLimits        *int64  `gorm:"column:current_time_limits" json:"current_time_limits"`
	CurrentAppointmentLimits *int64  `gorm:"column:current_appointment_limits" json:"current_appointment_limits"`
	CurrentReserveLimits     *int64  `gorm:"column:current_reserve_limits" json:"current_reserve_limits"`
	CurrentInternetLimits    *int64  `gorm:"column:current_internet_limits" json:"current_internet_limits"`
	OutpDate                 *string `gorm:"column:outp_date" json:"outp_date"`
	TimeQuantum              *string `gorm:"column:time_quantum" json:"time_quantum"`
}

func (*PixOutpRegMasterAtime) TableName() string {
	return TableNamePixOutpRegMasterAtime
}

func (*PixOutpRegMasterAtime) PrimaryKey() []string {
	return []string{"master_atime_id"}
}
