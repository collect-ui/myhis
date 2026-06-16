package plugins

import (
	"encoding/json"
	"fmt"
	"net/http"

	common "github.com/collect-ui/collect/src/collect/common"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"moon/model/base"
)

func RegisterAgentStreamRoutes(r *gin.Engine) {
	r.POST("/template_data/agent/run_stream", handleAgentRunStream)
	r.POST("/template_data/agent/run_cancel", handleAgentRunCancel)
	RegisterAgentArtifactRoutes(r)
}

func writeAgentStreamEvent(c *gin.Context, event string, payload map[string]interface{}) {
	data, _ := json.Marshal(payload)
	_, _ = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data)
	c.Writer.Flush()
}

func buildAgentRunStreamPayload(session *base.AgentSession, run *base.AgentRun, fallbackStatus string) map[string]interface{} {
	payload := map[string]interface{}{}
	if session != nil {
		payload["agent_session_id"] = session.AgentSessionID
		payload["session_key"] = session.SessionKey
		payload["scene_code"] = session.SceneCode
		payload["title"] = session.Title
		payload["status"] = session.Status
		payload["model"] = session.Model
		payload["last_active_time"] = session.LastActiveTime
		payload["create_time"] = session.CreateTime
		payload["modify_time"] = session.ModifyTime
	}
	if run == nil {
		if fallbackStatus != "" {
			payload["status"] = fallbackStatus
		}
		return payload
	}
	status := run.Status
	if status == "" {
		status = fallbackStatus
	}
	payload["agent_session_id"] = run.AgentSessionID
	payload["agent_run_id"] = run.AgentRunID
	payload["request_id"] = run.RequestID
	payload["status"] = status
	payload["current_step"] = run.CurrentStep
	payload["result_json"] = run.ResultJSON
	payload["error_msg"] = run.ErrorMsg
	payload["started_at"] = run.StartedAt
	payload["finished_at"] = run.FinishedAt
	payload["create_time"] = run.CreateTime
	payload["modify_time"] = run.ModifyTime
	return payload
}

func loadAgentRunForStream(agentRunID string, fallback base.AgentRun) base.AgentRun {
	result, err := callAgentLowcodeService("agent.run_query", map[string]interface{}{
		"agent_run_id": agentRunID,
		"to_obj":       true,
	})
	if err != nil || result == nil || result.Data == nil {
		return fallback
	}
	data, err := json.Marshal(result.Data)
	if err != nil {
		return fallback
	}
	var run base.AgentRun
	if err := json.Unmarshal(data, &run); err != nil || run.AgentRunID == "" {
		return fallback
	}
	return run
}

func updateAgentRunCancelledByLowcode(agentRunID string) error {
	for _, service := range []string{"agent.run_cancel_update", "agent.message_cancel_update"} {
		if _, err := callAgentLowcodeService(service, map[string]interface{}{
			"agent_run_id": agentRunID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func handleAgentRunStream(c *gin.Context) {
	ensureAgentRuntime()
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, common.NotOk(err.Error()))
		return
	}
	if params == nil {
		params = map[string]interface{}{}
	}
	params["stream_response"] = true
	if _, ok := params["session_key"]; !ok {
		params["session_key"] = "stream_" + uuid.NewString()
	}

	session, run, _, err := createAgentRun(params)
	if err != nil {
		c.JSON(http.StatusOK, common.NotOk(err.Error()))
		return
	}

	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	writeAgentStreamEvent(c, "start", buildAgentRunStreamPayload(session, run, run.Status))

	err = executeAgentRunWithStream(c.Request.Context(), run.AgentRunID, func(delta string) {
		if c.Request.Context().Err() != nil {
			return
		}
		writeAgentStreamEvent(c, "delta", map[string]interface{}{
			"agent_run_id": run.AgentRunID,
			"delta":        delta,
		})
	}, func(result agentToolResult) {
		if c.Request.Context().Err() != nil {
			return
		}
		writeAgentStreamEvent(c, "tool", map[string]interface{}{
			"agent_run_id": run.AgentRunID,
			"tool_result":  result,
		})
	}, func(event map[string]interface{}) {
		if c.Request.Context().Err() != nil {
			return
		}
		if event == nil {
			event = map[string]interface{}{}
		}
		event["agent_run_id"] = run.AgentRunID
		writeAgentStreamEvent(c, "log", event)
	})
	if err != nil {
		runAfter := loadAgentRunForStream(run.AgentRunID, *run)
		payload := buildAgentRunStreamPayload(session, &runAfter, runStatusFailed)
		payload["msg"] = err.Error()
		if payload["error_msg"] == "" {
			payload["error_msg"] = err.Error()
		}
		writeAgentStreamEvent(c, "done", payload)
		writeAgentStreamEvent(c, "error", payload)
		return
	}

	runAfter := loadAgentRunForStream(run.AgentRunID, *run)
	status := runStatusCompleted
	if runAfter.Status != "" {
		status = runAfter.Status
	}
	writeAgentStreamEvent(c, "done", buildAgentRunStreamPayload(session, &runAfter, status))
}

func handleAgentRunCancel(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, common.NotOk(err.Error()))
		return
	}
	agentRunID, _ := params["agent_run_id"].(string)
	if agentRunID == "" {
		c.JSON(http.StatusOK, common.NotOk("agent_run_id 不能为空"))
		return
	}
	cancelledLive := cancelAgentRun(agentRunID)
	if err := updateAgentRunCancelledByLowcode(agentRunID); err != nil {
		c.JSON(http.StatusOK, common.NotOk(err.Error()))
		return
	}
	c.JSON(http.StatusOK, common.Ok(map[string]interface{}{
		"agent_run_id":   agentRunID,
		"cancelled_live": cancelledLive,
	}, "已发送终止请求"))
}
