package his

const TableNamePixOutpRegMasterDetail = "pix_outp_reg_master_detail"

type PixOutpRegMasterDetail struct {
	MasterDetailID  int64   `gorm:"column:master_detail_id;primaryKey" json:"master_detail_id"`
	MasterAtimeID   int64   `gorm:"column:master_atime_id" json:"master_atime_id"`
	MasterID        int64   `gorm:"column:master_id" json:"master_id"`
	TimeQuantum     *string `gorm:"column:time_quantum" json:"time_quantum"`
	StageBeginTime  *string `gorm:"column:stage_begin_time" json:"stage_begin_time"`
	StageEndTime    *string `gorm:"column:stage_end_time" json:"stage_end_time"`
	OutpDate        *string `gorm:"column:outp_date" json:"outp_date"`
	SortNo          *int64  `gorm:"column:sort_no" json:"sort_no"`
	IsEnable        *string `gorm:"column:is_enable" json:"is_enable"`
	AppointmentFlag *string `gorm:"column:appointment_flag" json:"appointment_flag"`
	RegFlag         *string `gorm:"column:reg_flag" json:"reg_flag"`
	IsInternet      *string `gorm:"column:is_internet" json:"is_internet"`
	TimeStamp       *int64  `gorm:"column:time_stamp" json:"time_stamp"`
}

func (*PixOutpRegMasterDetail) TableName() string {
	return TableNamePixOutpRegMasterDetail
}

func (*PixOutpRegMasterDetail) PrimaryKey() []string {
	return []string{"master_detail_id"}
}
