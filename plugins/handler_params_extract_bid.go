package plugins

import (
	"encoding/json"
	"fmt"
	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	utils "github.com/collect-ui/collect/src/collect/utils"

	"os"
	"os/exec"
	"path/filepath"
	"time"

	"regexp"
	"strings"
)

type ExtractBid struct {
	templateService.BaseHandler
}

// 将 JSON 字符串转换为 map[string]interface{}
func jsonStringToMap(jsonStr string) (map[string]interface{}, error) {
	// 创建一个 map 用于存储解析结果
	var result map[string]interface{}

	// 解析 JSON 字符串
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func getValue(text string, reg string) string {
	nameRegex := regexp.MustCompile(reg)
	nameMatch := nameRegex.FindStringSubmatch(text)
	if len(nameMatch) > 1 {
		return strings.TrimSpace(nameMatch[1])
	}
	return ""
}

// 提取招标项目名称、工期和质量要求
func extractBiddingInfo(text string, fields []map[string]interface{}) map[string]interface{} {
	data := make(map[string]interface{})
	for _, field := range fields {
		key := field["key"].(string)
		reg := field["reg"]
		if utils.IsValueEmpty(reg) {
			continue
		}
		regTmp := reg.(string)
		// 匹配多个正则，在最前面优先匹配
		if strings.Contains(regTmp, "||") {
			for _, regItem := range strings.Split(regTmp, "||") {
				value := getValue(text, regItem)
				if !utils.IsValueEmpty(value) {
					data[key] = value
					break
				}
			}
		} else {
			data[key] = getValue(text, regTmp)
		}

		//nameRegex := regexp.MustCompile(reg.(string))
		//nameMatch := nameRegex.FindStringSubmatch(text)
		//if len(nameMatch) > 1 {
		//	data[key] = strings.TrimSpace(nameMatch[1])
		//}

	}
	return data

}
func (si *ExtractBid) HandlerData(template *config.Template, handlerParam *config.HandlerParam, ts *templateService.TemplateService) *common.Result {
	params := template.GetParams()
	pdfPath := utils.RenderVar(handlerParam.Path, params).(string)
	fields, _ := utils.RenderVarToArrMap(handlerParam.Foreach, params)
	// 起始页
	startPage := 1
	// 结束页
	endPage := 30

	// 生成随机文件名
	outputPath := filepath.Join("./", fmt.Sprintf("output_%d.txt", time.Now().UnixNano()))
	defer os.Remove(outputPath) // 确保文件最终被删除

	pdftotextPath, err := exec.LookPath("pdftotext")
	if err != nil {
		return common.NotOk("请先安装poppler-utils。 yum install poppler-utils")
	}

	cmd := exec.Command(pdftotextPath, "-f", fmt.Sprintf("%d", startPage), "-l", fmt.Sprintf("%d", endPage), pdfPath, "-")
	content, err := cmd.Output()
	if err != nil {
		return common.NotOk(err.Error())
	}

	// 解析标书字段
	result := extractBiddingInfo(string(content), fields)
	for _, field := range fields {
		key := field["key"].(string)
		dataFrom := field["data_from"]
		if utils.IsValueEmpty(dataFrom) {
			continue
		}
		if !utils.IsRenderVar(dataFrom.(string)) {
			continue
		}
		value := utils.RenderVar(dataFrom.(string), params)
		result[key] = value
	}

	r := common.Ok(result, "处理参数成功")
	return r
}
