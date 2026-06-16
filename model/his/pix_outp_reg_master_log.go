package his

import "time"

const TableNamePixOutpRegMasterLog = "pix_outp_reg_master_log"

type PixOutpRegMasterLog struct {
	MasterLogID int64      `gorm:"column:master_log_id;primaryKey" json:"master_log_id"`
	MasterID    int64      `gorm:"column:master_id" json:"master_id"`
	LogDate     *time.Time `gorm:"column:log_date" json:"log_date"`
	Operator    *string    `gorm:"column:operator" json:"operator"`
	LogText     *string    `gorm:"column:log_text" json:"log_text"`
}

func (*PixOutpRegMasterLog) TableName() string {
	return TableNamePixOutpRegMasterLog
}

func (*PixOutpRegMasterLog) PrimaryKey() []string {
	return []string{"master_log_id"}
}
