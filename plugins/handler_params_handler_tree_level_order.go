package plugins

import (
	"fmt"
	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	utils "github.com/collect-ui/collect/src/collect/utils"
	"strconv"
)

type HandlerTreeLevelOrder struct {
	templateService.BaseHandler
}

func (si *HandlerTreeLevelOrder) HandlerData(template *config.Template, handlerParam *config.HandlerParam, ts *templateService.TemplateService) *common.Result {
	params := template.GetParams()
	arr, errMsg := utils.RenderVarToArrMap(handlerParam.Foreach, params)
	if !utils.IsValueEmpty(errMsg) {
		return common.NotOk(errMsg)
	}
	target := buildNumberedTopics(arr)
	r := common.Ok(target, "处理参数成功")
	return r
}

func buildNumberedTopics(contents []map[string]interface{}) []map[string]interface{} {
	// 首先构建树形结构
	tree := contents

	var result []map[string]interface{}

	// 递归添加序号
	for i, root := range tree {
		// 创建第一级节点的map（复制原始数据）
		rootMap := root
		rootMap["seq"] = fmt.Sprintf("%s、 ", toChineseNumber(i+1))
		rootMap["level"] = 0
		//result = append(result, rootMap)

		// 处理子节点
		children, _ := root["children"].([]map[string]interface{})
		for j, child := range children {
			// 创建第二级节点的map
			childMap := child
			childMap["seq"] = fmt.Sprintf("%d. ", j+1)
			childMap["level"] = 1
			//result = append(result, childMap)

			// 处理第三级节点
			grandChildren, _ := child["children"].([]map[string]interface{})
			for k, grandChild := range grandChildren {
				// 创建第三级节点的map
				grandChildMap := grandChild
				grandChildMap["seq"] = fmt.Sprintf("%c) ", 'a'+k)
				grandChildMap["level"] = 2
				//result = append(result, grandChildMap)
			}
		}
	}
	return result
}

// 数字转中文（支持1-99）
func toChineseNumber(num int) string {
	if num < 1 {
		return strconv.Itoa(num)
	}

	digits := []string{"", "一", "二", "三", "四", "五", "六", "七", "八", "九"}

	if num <= 10 {
		// 1-10特殊处理
		chineseNumbers := []string{"一", "二", "三", "四", "五", "六", "七", "八", "九", "十"}
		return chineseNumbers[num-1]
	} else if num < 20 {
		// 11-19特殊处理（"十一"到"十九"）
		return "十" + digits[num%10]
	} else if num < 100 {
		// 20-99
		ten := num / 10
		one := num % 10
		if one == 0 {
			return digits[ten] + "十" // 例如"二十"、"三十"
		}
		return digits[ten] + "十" + digits[one] // 例如"二十一"、"三十九"
	}

	// 如果超过99，返回阿拉伯数字
	return strconv.Itoa(num)
}
