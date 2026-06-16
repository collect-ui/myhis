package plugins

import (
	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
)

type AgentSessionService struct {
	templateService.BaseHandler
}

func (s *AgentSessionService) Result(template *config.Template, ts *templateService.TemplateService) *common.Result {
	ensureAgentRuntime()
	session, err := getOrCreateAgentSession(template.GetParams())
	if err != nil {
		return common.NotOk(err.Error())
	}
	return common.Ok(session, "会话处理成功")
}
