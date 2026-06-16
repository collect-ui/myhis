package plugins

import (
	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	utils "github.com/collect-ui/collect/src/collect/utils"
	"github.com/demdxx/gocast"
)

type GenDocProject struct {
	templateService.BaseHandler
}

func (si *GenDocProject) HandlerData(template *config.Template, handlerParam *config.HandlerParam, ts *templateService.TemplateService) *common.Result {
	params := template.GetParams()
	arr, _ := utils.RenderVarToArrMap(handlerParam.Foreach, params)

	//p := "湖南建设工程造价电子数据标准/项目工程/工程项目标段概况"
	tag := "项目标段信息"

	project := make(map[string]interface{})
	projectRule := map[string]string{
		"项目标段编码": "code",
		"项目标段名称": "name",
		"建设单位":   "company",
		"项目所在地":  "address",
	}

	danRule := map[string]string{
		"项目编码": "code",
		"项目名称": "name",
		"项目特征": "detail",
		"计量单位": "unit",
		"工程量":  "spend",
	}
	danParentRule := map[string]string{
		"序号": "group",
		"名称": "topic",
	}
	danList := make([]map[string]interface{}, 0)
	//<清单项目 序号="1" 项目编码="041001001001" 项目名称="拆除混凝土路面" 项目特征="1.材质: 混凝土路面;&#xA;2.厚度: 20cm;&#xA;3.机械拆除;&#xA;4.拆除 清底 旧料清理成堆:" 工作内容="拆除、清理&#xA;运输" 计算规则="按拆除部位以面积计算&#xA;&#xA;[附注说明]&#xA;注：&#xA;1.拆除路面、人行道及管道清单项目的工作内容均不包括基础及垫层拆除，发生时按本章相应清单项目编码列项。&#xA;2.伐树、挖树蔸应按现行国家标准《园林绿化工程工程量计算规范》GB 50858中相应清单项目编码列项。" 计量单位="m2" 工程量="236.1" 综合单价="8.19" 综合合价="0" 直接费用合价="0" 人工费合价="0" 材料费合价="0" 未计价材料费合价="0" 设备费合价="0" 机械费合价="0" 管理费合价="0" 其他管理费合价="0" 利润合价="0" 暂估合价="0" 是否重点评审清单="false" 备注=""></清单项目>
	arrDict := make(map[string]map[string]interface{})
	for _, item := range arr {
		id := item["id"].(string)
		arrDict[id] = item
	}
	groupIDDict := make(map[string]string)
	count := 0
	groupList := make([]map[string]interface{}, 0)
	groupDict := make(map[string]bool)
	for index, item := range arr {
		tagTmp := item["tag"].(string)
		if tagTmp == tag {
			attributes, _ := utils.RenderVarToArrMap("attributes", item)
			for _, attr := range attributes {
				name := attr["name"].(string)
				value := attr["value"].(string)
				if field, ok := projectRule[name]; ok {
					project[field] = value
				}
			}
		}
		if tagTmp == "清单项目" {
			parentId := item["parent_id"].(string)
			parent := arrDict[parentId]
			danParent := getAttrItem(danParentRule, parent)
			//attributes, _ := utils.RenderVarToArrMap("attributes", item)
			//dan := make(map[string]interface{})
			//for _, attr := range attributes {
			//	name := attr["name"].(string)
			//	value := attr["value"]
			//	if field, ok := danRule[name]; ok {
			//		dan[field] = value
			//	}
			//}
			dan := getAttrItem(danRule, item)
			dan["order_index"] = index
			dan["parent_name"] = danParent["topic"]
			if topic, ok := danParent["topic"]; ok {
				if !utils.IsValueEmpty(topic) {
					group := danParent["group"].(string)
					if utils.IsValueEmpty(group) {
						group = groupIDDict[parentId]
						if utils.IsValueEmpty(group) {
							count += 1
							group = gocast.ToString(count)
							groupIDDict[parentId] = group
						}
					}
					dan["group"] = group
					if _, ok := groupDict[group]; !ok {
						danParent["group"] = group
						groupDict[group] = true
						groupList = append(groupList, danParent)
					}

				}
			}
			danList = append(danList, dan)
		}

	}
	params["project"] = project
	params["danList"] = danList
	params["groupList"] = groupList
	r := common.Ok(nil, "处理参数成功")
	return r
}
func getAttrItem(danRule map[string]string, item map[string]interface{}) map[string]interface{} {
	attributes, _ := utils.RenderVarToArrMap("attributes", item)
	dan := make(map[string]interface{})
	for _, attr := range attributes {
		name := attr["name"].(string)
		value := attr["value"]
		if field, ok := danRule[name]; ok {
			dan[field] = value
		}
	}
	return dan
}
