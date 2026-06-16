package his

const TableNamePixOutpRegScheduleAtime = "pix_outp_reg_schedule_atime"

type PixOutpRegScheduleAtime struct {
	ScheduleAtimeID int64   `gorm:"column:schedule_atime_id;primaryKey" json:"schedule_atime_id"`
	ScheduleID      int64   `gorm:"column:schedule_id" json:"schedule_id"`
	MasterID        int64   `gorm:"column:master_id" json:"master_id"`
	TimeQuantum     *string `gorm:"column:time_quantum" json:"time_quantum"`
	SortNo          *int64  `gorm:"column:sort_no" json:"sort_no"`
}

func (*PixOutpRegScheduleAtime) TableName() string {
	return TableNamePixOutpRegScheduleAtime
}

func (*PixOutpRegScheduleAtime) PrimaryKey() []string {
	return []string{"schedule_atime_id"}
}
