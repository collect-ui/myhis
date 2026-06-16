package his

const TableNamePixOutpRegSchedule = "pix_outp_reg_schedule"

type PixOutpRegSchedule struct {
	ScheduleID         int64   `gorm:"column:schedule_id;primaryKey" json:"schedule_id"`
	MasterID           int64   `gorm:"column:master_id" json:"master_id"`
	AreaCode           *string `gorm:"column:area_code" json:"area_code"`
	OutpSpecialID      *int64  `gorm:"column:outp_special_id" json:"outp_special_id"`
	SpecialClinicCode  *string `gorm:"column:special_clinic_code" json:"special_clinic_code"`
	ResideDeptCode     *string `gorm:"column:reside_dept_code" json:"reside_dept_code"`
	DoctorCode         *string `gorm:"column:doctor_code" json:"doctor_code"`
	OutpDurationCode   *string `gorm:"column:outp_duration_code" json:"outp_duration_code"`
	ScheduleDate       *string `gorm:"column:schedule_date" json:"schedule_date"`
	RegistrationLimits *int64  `gorm:"column:registration_limits" json:"registration_limits"`
	AppointmentLimits  *int64  `gorm:"column:appointment_limits" json:"appointment_limits"`
	ReserveLimits      *int64  `gorm:"column:reserve_limits" json:"reserve_limits"`
	InternetLimits     *int64  `gorm:"column:internet_limits" json:"internet_limits"`
	AtimeFlag          *string `gorm:"column:atime_flag" json:"atime_flag"`
	RegistrationType   *string `gorm:"column:registration_type" json:"registration_type"`
	AverageTime        *int64  `gorm:"column:average_time" json:"average_time"`
	RegFeeID           *int64  `gorm:"column:reg_fee_id" json:"reg_fee_id"`
	OutpTypeCode       *string `gorm:"column:outp_type_code" json:"outp_type_code"`
	TitleCode          *string `gorm:"column:title_code" json:"title_code"`
	BeginDate          *string `gorm:"column:begin_date" json:"begin_date"`
	EndDate            *string `gorm:"column:end_date" json:"end_date"`
	DayOfWeek          *string `gorm:"column:day_of_week" json:"day_of_week"`
	IsEnable           *string `gorm:"column:is_enable" json:"is_enable"`
	DeleteFlag         *string `gorm:"column:delete_flag" json:"delete_flag"`
	ScheduleStatus     *string `gorm:"column:schedule_status" json:"schedule_status"`
}

func (*PixOutpRegSchedule) TableName() string {
	return TableNamePixOutpRegSchedule
}

func (*PixOutpRegSchedule) PrimaryKey() []string {
	return []string{"schedule_id"}
}
