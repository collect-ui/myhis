package plugins

import (
	"encoding/json"
	"fmt"
	"strings"

	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	"github.com/demdxx/gocast"
	"github.com/google/uuid"

	"moon/model/base"
)

type AgentRunService struct {
	templateService.BaseHandler
}

func createAgentRun(params map[string]interface{}) (*base.AgentSession, *base.AgentRun, map[string]interface{}, error) {
	inputText := strings.TrimSpace(gocast.ToString(params["input_text"]))
	if inputText == "" {
		return nil, nil, nil, fmt.Errorf("input_text 不能为空")
	}

	session, err := getOrCreateAgentSession(params)
	if err != nil {
		return nil, nil, nil, err
	}

	requestData := map[string]interface{}{
		"input_text": inputText,
	}
	if mockResponse := strings.TrimSpace(gocast.ToString(params["mock_response"])); mockResponse != "" {
		requestData["mock_response"] = mockResponse
	}
	if delaySecond := gocast.ToInt64(params["simulate_delay_second"]); delaySecond > 0 {
		requestData["simulate_delay_second"] = delaySecond
	}
	if streamResponse := gocast.ToBool(params["stream_response"]) || gocast.ToBool(params["stream"]); streamResponse {
		requestData["stream_response"] = true
	}
	if gocast.ToBool(params["run_sync"]) {
		requestData["run_sync"] = true
	}
	if toolPolicyJSON := strings.TrimSpace(gocast.ToString(params["tool_policy_json"])); toolPolicyJSON != "" {
		requestData["tool_policy_json"] = toolPolicyJSON
	}
	if mcpPolicyJSON := strings.TrimSpace(gocast.ToString(params["mcp_policy_json"])); mcpPolicyJSON != "" {
		requestData["mcp_policy_json"] = mcpPolicyJSON
	}
	requestJSON, _ := json.Marshal(requestData)
	createTime := agentParamNow(params, "create_time")
	modifyTime := agentParamText(params, "modify_time", createTime)

	run := base.AgentRun{
		AgentRunID:     "agent_run_" + uuid.NewString(),
		AgentSessionID: session.AgentSessionID,
		RequestID:      "req_" + uuid.NewString(),
		TriggerType:    defaultTriggerType,
		Status:         runStatusQueued,
		CurrentStep:    "queued",
		MaxRetry:       1,
		RequestJSON:    string(requestJSON),
		CreateTime:     createTime,
		ModifyTime:     modifyTime,
		CreateUser:     gocast.ToString(params["session_user_id"]),
		ModifyUser:     gocast.ToString(params["session_user_id"]),
		IsDelete:       "0",
	}
	if triggerType := strings.TrimSpace(gocast.ToString(params["trigger_type"])); triggerType != "" {
		run.TriggerType = triggerType
	}
	if _, err := callAgentLowcodeService("agent.run_save", map[string]interface{}{
		"agent_run_id":      run.AgentRunID,
		"agent_session_id":  run.AgentSessionID,
		"request_id":        run.RequestID,
		"trigger_type":      run.TriggerType,
		"status":            run.Status,
		"current_step":      run.CurrentStep,
		"worker_id":         run.WorkerID,
		"lease_expire_time": run.LeaseExpireTime,
		"heartbeat_time":    run.HeartbeatTime,
		"retry_count":       run.RetryCount,
		"max_retry":         run.MaxRetry,
		"error_msg":         run.ErrorMsg,
		"request_json":      run.RequestJSON,
		"result_json":       run.ResultJSON,
		"started_at":        run.StartedAt,
		"finished_at":       run.FinishedAt,
		"create_time":       run.CreateTime,
		"modify_time":       run.ModifyTime,
		"create_user":       run.CreateUser,
		"modify_user":       run.ModifyUser,
		"is_delete":         run.IsDelete,
	}); err != nil {
		return nil, nil, nil, err
	}
	if _, err := appendAgentMessage(session.AgentSessionID, run.AgentRunID, "user", "user", inputText, string(requestJSON), run.CreateUser, agentParamText(params, "message_create_time", createTime)); err != nil {
		return nil, nil, nil, err
	}
	if isAgentStreamResponse(requestData) {
		if _, err := ensureStreamingAssistantMessage(session.AgentSessionID, run.AgentRunID, session.UserID); err != nil {
			return nil, nil, nil, err
		}
	}
	return session, &run, requestData, nil
}

func (s *AgentRunService) Result(template *config.Template, ts *templateService.TemplateService) *common.Result {
	ensureAgentRuntime()
	params := template.GetParams()
	session, run, _, err := createAgentRun(params)
	if err != nil {
		return common.NotOk(err.Error())
	}

	if gocast.ToBool(params["run_sync"]) {
		if err := executeAgentRun(run.AgentRunID); err != nil {
			return common.NotOk(err.Error())
		}
		if runAfter, ok, err := queryAgentRun(map[string]interface{}{"agent_run_id": run.AgentRunID}); err == nil && ok {
			run = runAfter
		}
	} else {
		go func(id string) {
			_ = executeAgentRun(id)
		}(run.AgentRunID)
	}

	return common.Ok(map[string]interface{}{
		"agent_session_id": session.AgentSessionID,
		"session_key":      session.SessionKey,
		"title":            session.Title,
		"agent_run_id":     run.AgentRunID,
		"request_id":       run.RequestID,
		"status":           run.Status,
	}, "任务已创建")
}
