package plugins

import (
	"encoding/json"
	"fmt"
	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	utils "github.com/collect-ui/collect/src/collect/utils"
	"strings"
)

type FixJson struct {
	templateService.BaseHandler
}

// extractJSON 从文本中提取 JSON 部分（兼容 ```json 或纯 JSON）
func extractJSON(input string) (string, error) {
	// 尝试查找 ```json 标记
	start := strings.Index(input, "```json")
	if start != -1 {
		start += len("```json")
		end := strings.Index(input[start:], "```")
		if end == -1 {
			return "", fmt.Errorf("unclosed JSON code block")
		}
		jsonStr := strings.TrimSpace(input[start : start+end])
		return jsonStr, nil
	}

	// 如果没有 ```json 标记，尝试直接解析整个文本是否为 JSON
	var js map[string]interface{}
	if err := json.Unmarshal([]byte(input), &js); err == nil {
		return input, nil // 直接返回原始 JSON
	}

	// 如果都不是，尝试提取可能的 JSON 部分（例如，在文本中嵌入的 JSON）
	start = strings.Index(input, "{")
	if start == -1 {
		return "", fmt.Errorf("no JSON found")
	}
	end := strings.LastIndex(input, "}")
	if end == -1 {
		return "", fmt.Errorf("malformed JSON")
	}
	jsonStr := strings.TrimSpace(input[start : end+1])
	return jsonStr, nil
}
func (si *FixJson) HandlerData(template *config.Template, handlerParam *config.HandlerParam, ts *templateService.TemplateService) *common.Result {
	params := template.GetParams()
	value := utils.RenderVar(handlerParam.Value, params).(string)
	input, err := extractJSON(value)
	if err != nil {
		return common.NotOk(err.Error())
	}
	r := common.Ok(input, "处理参数成功")
	return r
}
