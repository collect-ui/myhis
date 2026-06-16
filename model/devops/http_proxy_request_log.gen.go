package devops

const TableNameHttpProxyRequestLog = "http_proxy_request_log"

// HttpProxyRequestLog mapped from table <http_proxy_request_log>
type HttpProxyRequestLog struct {
	HttpProxyRequestLogID string `gorm:"column:http_proxy_request_log_id;primaryKey" json:"http_proxy_request_log_id"`
	CreateTime            string `gorm:"column:create_time" json:"create_time"`
	ProjectCode           string `gorm:"column:project_code" json:"project_code"`
	CookieScope           string `gorm:"column:cookie_scope" json:"cookie_scope"`
	RequestMethod         string `gorm:"column:request_method" json:"request_method"`
	RequestURL            string `gorm:"column:request_url" json:"request_url"`
	RequestHeaderText     string `gorm:"column:request_header_text" json:"request_header_text"`
	RequestDataText       string `gorm:"column:request_data_text" json:"request_data_text"`
	ResponseStatusCode    int    `gorm:"column:response_status_code" json:"response_status_code"`
	ResponseStatusText    string `gorm:"column:response_status_text" json:"response_status_text"`
	ResponseText          string `gorm:"column:response_text" json:"response_text"`
	ErrorText             string `gorm:"column:error_text" json:"error_text"`
	DurationMs            int64  `gorm:"column:duration_ms" json:"duration_ms"`
}

// TableName HttpProxyRequestLog's table name
func (*HttpProxyRequestLog) TableName() string {
	return TableNameHttpProxyRequestLog
}

func (*HttpProxyRequestLog) PrimaryKey() []string {
	return []string{"http_proxy_request_log_id"}
}
