package plugins

import (
	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
)

type ClientIp struct {
	templateService.BaseHandler
}

func (si *ClientIp) HandlerData(template *config.Template, handlerParam *config.HandlerParam, ts *templateService.TemplateService) *common.Result {
	c := ts.GetContext()
	r := common.Ok(c.ClientIP(), "处理参数成功")
	return r
}
