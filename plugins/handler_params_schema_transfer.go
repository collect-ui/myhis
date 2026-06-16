package plugins

import (
	"encoding/json"
	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	utils "github.com/collect-ui/collect/src/collect/utils"
	"github.com/demdxx/gocast"
)

type SchemaTransfer struct {
	templateService.BaseHandler
}

func handlerRuleTransfer(rule map[string]interface{}, params map[string]interface{}) []map[string]interface{} {
	ruleType := rule["type"].(string)
	ruleKey := rule["key"].(string)
	targetList := make([]map[string]interface{}, 0)
	// 处理后台 data attr 规则
	if dataAttr, ok := rule["data_attr"].(string); ok {
		var js map[string]interface{}
		if err := json.Unmarshal([]byte(dataAttr), &js); err == nil {
			ignore := gocast.ToBool(js["ignore"])
			if ignore {
				return targetList
			}
		}
	}
	if ruleType == "string" {
		target := make(map[string]interface{})
		target["key"] = ruleKey
		target["value"] = utils.RenderVar("["+ruleKey+"]", params)
		target["index"] = 0
		if orderIndex, ok := rule["order_index"]; ok {
			target["index"] = orderIndex
		}
		target["parent_key"] = ""
		targetList = append(targetList, target)
	} else if ruleType == "list" {
		list, _ := utils.RenderVarToArrMap(ruleKey, params)
		ruleList, _ := utils.RenderVarToArrMap("[children]", rule)
		//获取多条记录
		for index, item := range list {
			for _, childRule := range ruleList {
				childTarget := handlerRuleTransfer(childRule, item)[0]
				childTarget["index"] = index
				childTarget["parent_key"] = ruleKey
				if childTarget["value"] != nil {
					childTarget["value"] = childTarget["value"].(string)
				} else {
					childTarget["value"] = nil
				}

				targetList = append(targetList, childTarget)
			}

		}
	}

	return targetList
}

func handlerRuleRestore(transferList []map[string]interface{}, ruleDict map[string]interface{}, ruleList []map[string]interface{}) map[string]interface{} {
	targetData := make(map[string]interface{})
	var last int32
	lastItem := make(map[string]interface{})
	last = -1
	// 按数据转换字段
	lastParentKey := ""
	for _, item := range transferList {
		key, ok := item["key"].(string)
		if !ok {
			continue
		}
		value := item["value"]
		ruleKey := getKey(item)
		ruleTypeTmp, ok := ruleDict[getRuleKey(ruleKey)]
		if !ok {
			continue
		}
		ruleType := ruleTypeTmp.(string)
		if ruleKey != "string" {
			value = utils.CastValue(value, ruleType)
		}

		if ruleDict[ruleKey] == true { // 简单字段
			parentKey := getParentKey(item)
			index := gocast.ToInt32(item["index"])
			// 处理换了parent
			if parentKey != lastParentKey {
				lastParentKey = parentKey
				last = -1
			}
			newRow := false
			if last != index {
				last = index
				lastItem = make(map[string]interface{})
				newRow = true
			}
			if _, ok := targetData[parentKey]; !ok {
				subList := make([]map[string]interface{}, 0)
				targetData[parentKey] = subList
			}
			lastItem[key] = value
			if newRow {
				lastItem["index"] = index
				targetData[parentKey] = append(targetData[parentKey].([]map[string]interface{}), lastItem)
			}
		} else {
			targetData[key] = value
		}

	}

	//还是不能补充，否则不太好处理默认值，后续看看是否有控制开关。目前开无法区分是新增，还是修改
	// 根据规则补充字段，如果数据没有
	for _, item := range ruleList {
		parentKey := item["parent_key"].(string)
		key := item["key"].(string)
		varType := item["type"].(string)
		if utils.IsValueEmpty(parentKey) { // 没有父字段
			if _, ok := targetData[key]; !ok {
				if varType == "list" {
					targetData[key] = make([]map[string]interface{}, 0)
				} else {
					targetData[key] = nil
				}

			}
		}

	}

	return targetData
}

func getParentKey(rule map[string]interface{}) string {
	parentKey := ""
	if parentKeyStr, ok := rule["parent_key"]; ok {
		parentKey = parentKeyStr.(string)
	}
	return parentKey
}
func getKey(rule map[string]interface{}) string {

	parentKey := getParentKey(rule)
	ruleKey := rule["key"].(string) + "#" + parentKey
	return ruleKey
}
func getRuleKey(ruleKey string) string {
	return ruleKey + "_rule"
}
func getRuleDict(ruleList []map[string]interface{}) map[string]interface{} {
	ruleDict := make(map[string]interface{})
	for _, item := range ruleList {
		parentKey := getParentKey(item)
		ruleKey := getKey(item)
		if utils.IsValueEmpty(parentKey) {
			ruleDict[ruleKey] = false
		} else {
			ruleDict[ruleKey] = true
		}
		t := item["type"]
		if utils.IsValueEmpty(t) {
			t = "string"
		}
		ruleDict[getRuleKey(ruleKey)] = t

	}
	return ruleDict
}

func (si *SchemaTransfer) HandlerData(template *config.Template, handlerParam *config.HandlerParam, ts *templateService.TemplateService) *common.Result {
	params := template.GetParams()
	//field := handlerParam.Field
	//fieldValue := utils.RenderVar(field, params)
	ruleList, _ := utils.RenderVarToArrMap(handlerParam.Foreach, params)
	//saveOriginal := handlerParam.SaveOriginal

	if handlerParam.Operation == "transfer" {
		dataList := make([]map[string]interface{}, 0)
		for _, item := range ruleList {
			key := item["key"].(string)
			//如果前端没有传这个字段就不更新
			if _, ok := params[key]; !ok {
				continue
			}
			target := handlerRuleTransfer(item, params)
			dataList = append(dataList, target...)
		}
		r := common.Ok(dataList, "处理参数成功")
		return r
	} else if handlerParam.Operation == "restore" {
		transferList, _ := utils.RenderVarToArrMap(handlerParam.Field, params)
		// 将规则转字典
		ruleDict := getRuleDict(ruleList)
		targetData := handlerRuleRestore(transferList, ruleDict, ruleList)
		r := common.Ok(targetData, "处理参数成功")
		return r
	} else if handlerParam.Operation == "restore_list" {
		arrList, _ := utils.RenderVarToArrMap(handlerParam.Field, params)
		targetIDList := make([]string, 0)
		list := make([]map[string]interface{}, 0)
		for _, item := range arrList {
			belongID := utils.RenderVar("[belong_id]", item).(string)
			if !utils.StringArrayContain(targetIDList, belongID) {
				targetIDList = append(targetIDList, belongID)
			}
		}
		// 将规则转字典
		ruleDict := getRuleDict(ruleList)
		for _, belongID := range targetIDList {
			transferList := make([]map[string]interface{}, 0)
			for _, item := range arrList {
				if utils.RenderVar("[belong_id]", item).(string) == belongID {
					transferList = append(transferList, item)
				}
			}
			targetData := handlerRuleRestore(transferList, ruleDict, ruleList)
			targetData["belong_id"] = belongID
			list = append(list, targetData)
		}

		r := common.Ok(list, "处理参数成功")
		return r
	} else if handlerParam.Operation == "restore_config" {
		// 将规则转字典
		ruleDict := getRuleDict(ruleList)
		targetData := handlerRuleRestore(ruleList, ruleDict, ruleList)
		r := common.Ok(targetData, "处理参数成功")
		return r
	}

	return common.Ok(nil, "处理参数成功")
}
