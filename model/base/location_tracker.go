package base

const TableNameLocationTracker = "location_tracker"

type LocationTracker struct {
	LocationTrackerID string `gorm:"column:location_tracker_id;primaryKey" json:"location_tracker_id"`
	Path              string `gorm:"column:path" json:"path"`
	MenuCode          string `gorm:"column:menu_code" json:"menu_code"`
	ExtraData         string `gorm:"column:extra_data" json:"-"`
	CreateTime        string `gorm:"column:create_time" json:"create_time"`
	CreateUser        string `gorm:"column:create_user" json:"create_user"`
	ClientIP          string `gorm:"column:client_ip" json:"client_ip"`
	FinishTime        string `gorm:"column:finish_time" json:"finish_time"`
}

// TableName LocationTracker's table name
func (*LocationTracker) TableName() string {
	return TableNameLocationTracker
}

func (*LocationTracker) PrimaryKey() []string {
	return []string{"location_tracker_id"}
}
