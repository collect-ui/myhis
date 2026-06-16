package plugins

import (
	"encoding/json"
	"fmt"
	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	"io"
	"strings"
)

type MailAccountAuthJSONImport struct {
	templateService.BaseHandler
}

type mailAccountAuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	AccountID    string `json:"account_id"`
}

type mailAccountAuthPayload struct {
	AuthMode     string                `json:"auth_mode"`
	OpenAIAPIKey interface{}           `json:"OPENAI_API_KEY"`
	LastRefresh  string                `json:"last_refresh"`
	Tokens       mailAccountAuthTokens `json:"tokens"`
}

func (si *MailAccountAuthJSONImport) HandlerData(template *config.Template, handlerParam *config.HandlerParam, ts *templateService.TemplateService) *common.Result {
	if ts.File == nil {
		return common.NotOk("上传 JSON 文件不能为空")
	}

	content, err := io.ReadAll(ts.File)
	if err != nil {
		return common.NotOk(fmt.Sprintf("读取上传文件失败: %s", err.Error()))
	}

	raw := strings.TrimSpace(string(content))
	if raw == "" {
		return common.NotOk("上传 JSON 文件内容不能为空")
	}

	payload := mailAccountAuthPayload{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return common.NotOk(fmt.Sprintf("JSON 解析失败: %s", err.Error()))
	}

	accessToken := strings.TrimSpace(payload.Tokens.AccessToken)
	refreshToken := strings.TrimSpace(payload.Tokens.RefreshToken)
	idToken := strings.TrimSpace(payload.Tokens.IDToken)
	accountID := strings.TrimSpace(payload.Tokens.AccountID)

	if accessToken == "" || refreshToken == "" || idToken == "" || accountID == "" {
		rootMap := map[string]interface{}{}
		if err := json.Unmarshal([]byte(raw), &rootMap); err == nil {
			accessToken = firstNonEmptyString(accessToken, mapString(rootMap["access_token"]))
			refreshToken = firstNonEmptyString(refreshToken, mapString(rootMap["refresh_token"]))
			idToken = firstNonEmptyString(idToken, mapString(rootMap["id_token"]))
			accountID = firstNonEmptyString(accountID, mapString(rootMap["account_id"]))
		}
	}

	if accessToken == "" || refreshToken == "" || idToken == "" || accountID == "" {
		return common.NotOk("JSON 中缺少 tokens.access_token / refresh_token / id_token / account_id")
	}

	result := map[string]interface{}{
		"codex_access_token":  accessToken,
		"codex_refresh_token": refreshToken,
		"codex_id_token":      idToken,
		"codex_account_id":    accountID,
		"codex_auth_json":     raw,
		"codex_auth_status":   "token_ready",
		"codex_auth_msg":      "uploaded_from_json",
	}
	if strings.TrimSpace(payload.LastRefresh) != "" {
		result["codex_last_auth_time"] = strings.TrimSpace(payload.LastRefresh)
	}

	return common.Ok(result, "处理参数成功")
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func mapString(value interface{}) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}
