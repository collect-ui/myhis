package his

const TableNamePixOutpRegMaster = "pix_outp_reg_master"

type PixOutpRegMaster struct {
	MasterID                 int64   `gorm:"column:master_id;primaryKey" json:"master_id"`
	AreaCode                 *string `gorm:"column:area_code" json:"area_code"`
	OutpDate                 *string `gorm:"column:outp_date" json:"outp_date"`
	OutpSpecialID            *int64  `gorm:"column:outp_special_id" json:"outp_special_id"`
	ResideDeptCode           *string `gorm:"column:reside_dept_code" json:"reside_dept_code"`
	SpecialClinicCode        *string `gorm:"column:special_clinic_code" json:"special_clinic_code"`
	OutpTitleCode            *string `gorm:"column:outp_title_code" json:"outp_title_code"`
	DoctorCode               *string `gorm:"column:doctor_code" json:"doctor_code"`
	OutpTypeCode             *string `gorm:"column:outp_type_code" json:"outp_type_code"`
	OutpDurationCode         *string `gorm:"column:outp_duration_code" json:"outp_duration_code"`
	RegistrationLimits       *int64  `gorm:"column:registration_limits" json:"registration_limits"`
	AppointmentLimits        *int64  `gorm:"column:appointment_limits" json:"appointment_limits"`
	CurrentLimits            *int64  `gorm:"column:current_limits" json:"current_limits"`
	AppointmentCurrentLimits *int64  `gorm:"column:appointment_current_limits" json:"appointment_current_limits"`
	AtimeFlag                *string `gorm:"column:atime_flag" json:"atime_flag"`
	IsEnable                 *string `gorm:"column:is_enable" json:"is_enable"`
	UploadFlag               *string `gorm:"column:upload_flag" json:"upload_flag"`
	ModifyFlag               *string `gorm:"column:modify_flag" json:"modify_flag"`
	CurrentNo                *int64  `gorm:"column:current_no" json:"current_no"`
	SortNo                   *int64  `gorm:"column:sort_no" json:"sort_no"`
	TransferNo               *int64  `gorm:"column:transfer_no" json:"transfer_no"`
	AliasName                *string `gorm:"column:alias_name" json:"alias_name"`
	MasterStatus             *string `gorm:"column:master_status" json:"master_status"`
	Descn                    *string `gorm:"column:descn" json:"descn"`
	OldDoctorCode            *string `gorm:"column:old_doctor_code" json:"old_doctor_code"`
	RegFeeID                 *int64  `gorm:"column:reg_fee_id" json:"reg_fee_id"`
	RegistrationType         *string `gorm:"column:registration_type" json:"registration_type"`
	InternetLimits           *string `gorm:"column:internet_limits" json:"internet_limits"`
	AverageTime              *int64  `gorm:"column:average_time" json:"average_time"`
}

func (*PixOutpRegMaster) TableName() string {
	return TableNamePixOutpRegMaster
}

func (*PixOutpRegMaster) PrimaryKey() []string {
	return []string{"master_id"}
}
