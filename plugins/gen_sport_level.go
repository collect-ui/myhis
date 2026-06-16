package plugins

import (
	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	utils "github.com/collect-ui/collect/src/collect/utils"
	"github.com/demdxx/gocast"
)

type GenSportLevel struct {
	templateService.BaseHandler
}

func (si *GenSportLevel) HandlerData(template *config.Template, handlerParam *config.HandlerParam, ts *templateService.TemplateService) *common.Result {
	params := template.GetParams()
	arr, errMsg := utils.RenderVarToArrMap(handlerParam.Foreach, params)
	if !utils.IsValueEmpty(errMsg) {
		return common.NotOk(errMsg)
	}
	for _, item := range arr {
		for _, field := range handlerParam.Fields {
			target := utils.RenderVar(field.Field, item)
			if target == nil {
				continue
			}
			score := gocast.ToInt(target)
			levelName := ""
			level := ""
			if score < 60 {
				levelName = "不合格"
				level = "not_passed"
			} else if score >= 60 && score < 80 {
				levelName = "合格"
				level = "passed"
			} else if score >= 80 && score < 90 {
				levelName = "良好"
				level = "good"
			} else {
				levelName = "优秀"
				level = "excellent"
			}
			item[field.To+"_name"] = levelName
			item[field.To] = level
		}
	}
	r := common.Ok(arr, "处理参数成功")
	return r
}
