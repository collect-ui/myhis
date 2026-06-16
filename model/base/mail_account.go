package base

const TableNameMailAccount = "mail_account"

type MailAccount struct {
	MailAccountID                string `gorm:"column:mail_account_id;primaryKey" json:"mail_account_id"`
	OrderIndex                   int32  `gorm:"column:order_index" json:"order_index"`
	EmailName                    string `gorm:"column:email_name" json:"email_name"`
	Password                     string `gorm:"column:password" json:"password"`
	GUIDCode                     string `gorm:"column:guid_code" json:"guid_code"`
	RecoveryCode                 string `gorm:"column:recovery_code" json:"recovery_code"`
	RawText                      string `gorm:"column:raw_text" json:"raw_text"`
	IsCurrentRunning             string `gorm:"column:is_current_running" json:"is_current_running"`
	CurrentRunMarkTime           string `gorm:"column:current_run_mark_time" json:"current_run_mark_time"`
	ProtonRegistered             string `gorm:"column:proton_registered" json:"proton_registered"`
	ProtonEmail                  string `gorm:"column:proton_email" json:"proton_email"`
	ProtonPassword               string `gorm:"column:proton_password" json:"proton_password"`
	CodexDeviceCode              string `gorm:"column:codex_device_code" json:"codex_device_code"`
	CodexDeviceAuthID            string `gorm:"column:codex_device_auth_id" json:"codex_device_auth_id"`
	CodexUserCode                string `gorm:"column:codex_user_code" json:"codex_user_code"`
	CodexAuthorizationCode       string `gorm:"column:codex_authorization_code" json:"codex_authorization_code"`
	CodexCodeVerifier            string `gorm:"column:codex_code_verifier" json:"codex_code_verifier"`
	CodexVerificationURI         string `gorm:"column:codex_verification_uri" json:"codex_verification_uri"`
	CodexVerificationURIComplete string `gorm:"column:codex_verification_uri_complete" json:"codex_verification_uri_complete"`
	CodexInterval                string `gorm:"column:codex_interval" json:"codex_interval"`
	CodexExpiresIn               string `gorm:"column:codex_expires_in" json:"codex_expires_in"`
	CodexAccessToken             string `gorm:"column:codex_access_token" json:"codex_access_token"`
	CodexRefreshToken            string `gorm:"column:codex_refresh_token" json:"codex_refresh_token"`
	CodexIDToken                 string `gorm:"column:codex_id_token" json:"codex_id_token"`
	CodexTokenType               string `gorm:"column:codex_token_type" json:"codex_token_type"`
	CodexScope                   string `gorm:"column:codex_scope" json:"codex_scope"`
	CodexExpiresAt               string `gorm:"column:codex_expires_at" json:"codex_expires_at"`
	CodexAccountID               string `gorm:"column:codex_account_id" json:"codex_account_id"`
	CodexUsageJSON               string `gorm:"column:codex_usage_json" json:"codex_usage_json"`
	CodexUsagePlanType           string `gorm:"column:codex_usage_plan_type" json:"codex_usage_plan_type"`
	CodexUsageAllowed            string `gorm:"column:codex_usage_allowed" json:"codex_usage_allowed"`
	CodexUsageLimitReached       string `gorm:"column:codex_usage_limit_reached" json:"codex_usage_limit_reached"`
	CodexUsageUsedPercent        string `gorm:"column:codex_usage_used_percent" json:"codex_usage_used_percent"`
	CodexUsageResetAt            string `gorm:"column:codex_usage_reset_at" json:"codex_usage_reset_at"`
	CodexUsageLastQueryTime      string `gorm:"column:codex_usage_last_query_time" json:"codex_usage_last_query_time"`
	CodexUsageMsg                string `gorm:"column:codex_usage_msg" json:"codex_usage_msg"`
	CodexAuthJSON                string `gorm:"column:codex_auth_json" json:"codex_auth_json"`
	CodexAuthStatus              string `gorm:"column:codex_auth_status" json:"codex_auth_status"`
	CodexAuthMsg                 string `gorm:"column:codex_auth_msg" json:"codex_auth_msg"`
	CodexLastAuthTime            string `gorm:"column:codex_last_auth_time" json:"codex_last_auth_time"`
	CreateTime                   string `gorm:"column:create_time" json:"create_time"`
	CreateUser                   string `gorm:"column:create_user" json:"create_user"`
	IsDelete                     string `gorm:"column:is_delete" json:"is_delete"`
	IsDisabled                   string `gorm:"column:is_disabled" json:"is_disabled"`
}

func (*MailAccount) TableName() string {
	return TableNameMailAccount
}

func (*MailAccount) PrimaryKey() []string {
	return []string{"mail_account_id"}
}
