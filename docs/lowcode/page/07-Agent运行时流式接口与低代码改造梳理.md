# Agent 运行时流式接口与低代码改造梳理

更新时间：2026-05-27

本文梳理 `plugins.RegisterAgentStreamRoutes(r)` 注册出来的接口、调用流程、低代码服务依赖，以及当前代码如何和大模型交互。目标是给后续低代码化改造提供边界和拆分依据。

## 1. 总览结论

`plugins.RegisterAgentStreamRoutes(r)` 直接注册 3 个路由路径，按 HTTP method 计是 4 个接口：

| 序号 | Method | Path | Handler | 作用 |
| --- | --- | --- | --- | --- |
| 1 | POST | `/template_data/agent/run_stream` | `handleAgentRunStream` | 创建 Agent run，并以 SSE 方式实时返回模型 delta、工具调用、日志和结束事件 |
| 2 | POST | `/template_data/agent/run_cancel` | `handleAgentRunCancel` | 终止正在运行的 Agent run，同时把 run/message 状态更新为 cancelled |
| 3 | GET | `/template_data/agent/artifact` | `handleAgentArtifact` | 按安全白名单读取工具生成的图片 artifact，供 Markdown 图片展示 |
| 4 | HEAD | `/template_data/agent/artifact` | `handleAgentArtifact` | 与 GET 同 handler，用于图片资源探测 |

注册位置：

- `main.go:367` 调用 `plugins.RegisterAgentStreamRoutes(r)`
- `plugins/agent_stream.go:15-19` 注册 `run_stream`、`run_cancel`，并继续注册 artifact
- `plugins/agent_artifact.go:16-19` 注册 artifact 的 GET/HEAD

关键判断：

- `/template_data/agent/run_stream` 不是低代码 `/template_data/data?service=...` 服务，而是 Gin 直连路由。
- 运行状态、会话、消息落库已经大量使用 `collect/agent/**/index.yml` 的低代码服务。
- 真正不能简单配置化的部分在 Go 里：SSE 长连接、大模型 Responses 请求、SSE 解析、工具调用循环、取消上下文、文件/命令/浏览器工具的安全执行。

## 2. 相关入口文件

| 文件 | 作用 |
| --- | --- |
| `main.go` | Gin 主路由注册，挂载 `/template_data/data`、Agent 流式接口、WebSocket |
| `plugins/agent_stream.go` | 直接 HTTP 接口层，负责 `run_stream`、`run_cancel`、SSE 输出 |
| `plugins/agent_artifact.go` | artifact 图片文件访问 |
| `plugins/module_agent_run.go` | 低代码 `module: agent_run`，创建 Agent run，支持同步/异步非流式执行 |
| `plugins/module_agent_session.go` | 低代码 `module: agent_session`，创建或更新 session |
| `plugins/agent_runtime.go` | Agent 核心运行时：队列、状态机、provider 调用、SSE 解析、工具循环、落库 |
| `plugins/agent_tools.go` | 大模型可调用的本地工具定义和执行 |
| `collect/agent/service.yml` | Agent 低代码服务聚合，挂载 session/message/run |
| `collect/agent/session/index.yml` | 会话查询、保存、更新、删除 |
| `collect/agent/run/index.yml` | run 创建、查询、状态更新、抢占、心跳、取消 |
| `collect/agent/message/index.yml` | 消息查询、保存、流式内容更新、失败/取消更新 |
| `/data/project/sport-ui/src/action/agent-run-stream.tsx` | 前端自定义 action，直接请求 `/template_data/agent/run_stream` 并解析 SSE |
| `/data/project/sport-ui/src/main.tsx` | 注册 `agent-run-stream`、`agent-run-cancel` 等 action |

## 3. 直接 Gin 接口清单

### 3.1 POST `/template_data/agent/run_stream`

职责：

- 接收前端发来的聊天请求。
- 创建或更新 `agent_session`。
- 创建 `agent_run`，保存用户消息，创建流式 assistant 占位消息。
- 通过 SSE 向前端实时推送：
  - `start`：任务创建成功，返回 session/run 基本信息。
  - `delta`：模型输出增量文本。
  - `tool`：工具调用结果，包括真实工具结果和 `model_request_debug` 调试记录。
  - `log`：provider 重试、auth reload、超时等运行日志。
  - `done`：任务结束，返回最终 run 状态。
  - `error`：任务失败时附带错误信息。当前实现失败时会先发 `done` 再发 `error`。

主要请求参数：

| 参数 | 说明 |
| --- | --- |
| `input_text` | 必填，用户输入内容 |
| `agent_session_id` | 可选，传入则复用该 session |
| `session_key` | 可选，未传时 handler 默认生成 `stream_` 前缀 key |
| `scene_code` | 场景码，默认 `default`，回归页常用 `agent_regression_page` |
| `title` | 会话标题，可为空，后端会根据输入生成 |
| `system_prompt` | 系统提示词，进入大模型 `instructions` |
| `model` | 模型名，默认 `gpt-5-mini` |
| `tool_policy_json` | 工具策略，控制读文件、搜索、编辑、命令、浏览器、图片检查等能力 |
| `mcp_policy_json` | 目前只保存和传递，核心运行时未看到实质执行 |
| `mock_response` | 有值时不请求真实模型，直接返回 mock |
| `simulate_delay_second` | 测试用延迟 |
| `stream_response` | handler 会强制设为 true |

返回形式：

- `Content-Type: text/event-stream; charset=utf-8`
- 每个事件格式：

```text
event: delta
data: {"agent_run_id":"...","delta":"..."}
```

核心调用链：

1. `handleAgentRunStream`
2. `ensureAgentRuntime`
3. `createAgentRun`
4. `writeAgentStreamEvent(start, ...)`
5. `executeAgentRunWithStream`
6. `callAgentProvider`
7. `callAgentProviderWithTools` 或直接 `callChatGPTCodexProvider` / `callPlatformResponsesProvider`
8. `callResponsesProviderWithAuthReloadRetry`
9. `callResponsesProviderOnce`
10. `readAgentProviderSSEWithIdleTimeout`
11. 状态落库，发送 `done` 或 `error`

### 3.2 POST `/template_data/agent/run_cancel`

职责：

- 终止某个 `agent_run_id`。
- 如果该 run 在当前进程内正在执行，则调用内存里的 `context.CancelFunc`。
- 不管是否命中内存中的运行任务，都会尝试把低代码表状态改成 cancelled。

请求参数：

| 参数 | 说明 |
| --- | --- |
| `agent_run_id` | 必填，要终止的 run |

核心流程：

1. `handleAgentRunCancel` 解析 JSON。
2. `cancelAgentRun(agentRunID)` 从 `agentRunCancelMap` 找 cancel 函数并调用。
3. `updateAgentRunCancelledByLowcode` 依次调用：
   - `agent.run_cancel_update`
   - `agent.message_cancel_update`
4. 返回：

```json
{
  "agent_run_id": "...",
  "cancelled_live": true
}
```

`cancelled_live=true` 表示当前进程内命中了正在执行的上下文；`false` 表示只做了状态更新，可能任务已经结束、在其他进程、或当前内存没有记录。

### 3.3 GET/HEAD `/template_data/agent/artifact`

职责：

- 给 Agent 工具生成的图片文件提供安全访问入口。
- 主要被 `browser_check`、`inspect_image` 等工具返回的 Markdown 图片使用。

请求参数：

| 参数 | 说明 |
| --- | --- |
| `path` | 必填，本地图片路径 |

安全限制：

- `path` 必须落在 `defaultAgentWorkspaceRoots()` 允许的目录内。
- 只允许图片扩展名：`.png`、`.jpg`、`.jpeg`、`.gif`、`.webp`。
- 目录不允许访问。
- 响应加 `Cache-Control: no-store, no-cache, must-revalidate`。

## 4. 配套低代码 HTTP 服务

`collect/service_router.yml` 挂载：

- `key: agent`
- `path: agent/service.yml`

`collect/agent/service.yml` 再挂载：

- `session/index.yml`
- `message/index.yml`
- `run/index.yml`

对外 `http: true` 的 Agent 低代码服务共有 6 个：

| 服务 | 文件 | 用途 |
| --- | --- | --- |
| `agent.session_upsert` | `collect/agent/session/index.yml` | 低代码页面创建或更新 session，内部使用 `module: agent_session` |
| `agent.session_query` | `collect/agent/session/index.yml` | 查询 session 列表或单个 session |
| `agent.session_delete` | `collect/agent/session/index.yml` | 逻辑删除 session |
| `agent.run_create` | `collect/agent/run/index.yml` | 非流式创建 run，内部使用 `module: agent_run`，可同步或异步执行 |
| `agent.run_query` | `collect/agent/run/index.yml` | 查询 run 列表或单个 run |
| `agent.message_query` | `collect/agent/message/index.yml` | 查询消息列表 |

这些服务都走通用入口：

- `POST /template_data/data?service=agent.xxx`
- `main.go:364-366` 调用 `templateService.HandlerRequest(c)`

## 5. 运行时依赖的内部低代码服务

`collect/agent` 下总计 27 个服务，其中 6 个对外 HTTP，21 个主要供 Go 运行时内部调用。

### 5.1 session 服务

| 服务 | 对外 HTTP | 用途 |
| --- | --- | --- |
| `agent.session_upsert` | 是 | 创建或更新 session 的外部入口 |
| `agent.session_query` | 是 | 查询 session |
| `agent.session_save` | 否 | 保存新 session |
| `agent.session_update` | 否 | 更新已有 session |
| `agent.session_last_response_update` | 否 | 保存最新 provider response id 和活跃时间 |
| `agent.session_delete` | 是 | 逻辑删除 session |

### 5.2 run 服务

| 服务 | 对外 HTTP | 用途 |
| --- | --- | --- |
| `agent.run_create` | 是 | 创建 run 的低代码入口 |
| `agent.run_query` | 是 | 查询 run |
| `agent.run_queued_query` | 否 | 后台 worker 查询 queued run |
| `agent.run_expired_query` | 否 | 查询租约过期的 running run |
| `agent.run_save` | 否 | 保存新 run |
| `agent.run_claim_update` | 否 | 抢占 queued run，改成 running |
| `agent.run_heartbeat_update` | 否 | running 期间续租 |
| `agent.run_fail_update` | 否 | 标记失败 |
| `agent.run_fail_result_update` | 否 | 标记失败，同时保存 result_json |
| `agent.run_complete_update` | 否 | 标记完成并保存 result_json |
| `agent.run_expired_fail_update` | 否 | 把过期 running run 标记失败 |
| `agent.run_cancel_update` | 否 | 把 queued/running run 标记取消 |

### 5.3 message 服务

| 服务 | 对外 HTTP | 用途 |
| --- | --- | --- |
| `agent.message_query` | 是 | 查询消息 |
| `agent.message_max_seq_query` | 否 | 计算下一条消息 seq_no |
| `agent.message_latest_assistant_query` | 否 | 查询某 run 最新 assistant 消息 |
| `agent.message_save` | 否 | 保存 user/assistant 消息 |
| `agent.message_stream_start_update` | 否 | 把 assistant 消息置为 running |
| `agent.message_content_update` | 否 | 更新 assistant 文本和状态 |
| `agent.message_content_json_update` | 否 | 更新 assistant 文本、content_json 和状态 |
| `agent.message_run_failed_update` | 否 | run 过期失败时更新 running assistant 消息 |
| `agent.message_cancel_update` | 否 | run 取消时更新 running assistant 消息 |

## 6. run_stream 详细流程

### 6.1 创建阶段

`handleAgentRunStream`：

1. 解析 JSON。
2. 设置 `stream_response=true`。
3. 如果没有 `session_key`，生成 `stream_ + uuid`。
4. 调用 `createAgentRun(params)`。

`createAgentRun`：

1. 校验 `input_text`。
2. 调用 `getOrCreateAgentSession`。
3. 组装 `request_json`，包含：
   - `input_text`
   - `mock_response`
   - `simulate_delay_second`
   - `stream_response`
   - `run_sync`
   - `tool_policy_json`
   - `mcp_policy_json`
4. 生成 `agent_run_id`、`request_id`，初始状态为 `queued`。
5. 调用 `agent.run_save` 保存 run。
6. 调用 `appendAgentMessage` 保存 user 消息。
7. 如果是流式响应，调用 `ensureStreamingAssistantMessage` 创建 running assistant 占位消息。

### 6.2 执行阶段

`executeAgentRunWithStream`：

1. `claimAgentRun` 调用 `agent.run_claim_update`，只有 queued 状态能抢占成功。
2. 注册取消函数到 `agentRunCancelMap`。
3. 启动心跳，每 15 秒调用 `agent.run_heartbeat_update`，租约延长 2 分钟。
4. 查询 session：`agent.session_query`。
5. 查询当前 run 的 user 消息：`agent.message_query`。
6. 如果是流式，准备 `onDelta`：
   - 累加文本。
   - 每 250ms 或新增 64 字节刷新一次 `agent.message_content_update`。
   - 同时把 delta 推给 HTTP SSE 客户端。
7. 调用 `callAgentProvider` 请求大模型。
8. 成功后：
   - assistant 消息写入最终文本和 `content_json`。
   - session 写入 `last_response_id`。
   - run 写入 `completed` 和 `result_json`。
9. 失败后：
   - 取消类错误写入 cancelled。
   - 普通错误写入 failed。
   - 如果已有工具调试结果，会写入 `run_fail_result_update`。

### 6.3 后台 worker

`ensureAgentRuntime` 只初始化一次，并启动 `startAgentRunWorker`。

worker 每 2 秒执行：

1. `markExpiredAgentRuns`
   - 查 `agent.run_expired_query`
   - 调 `agent.run_expired_fail_update`
   - 调 `agent.message_run_failed_update`
2. `processQueuedRuns`
   - 查 `agent.run_queued_query`
   - 跳过 `run_sync=true`
   - 跳过 `stream_response=true`
   - 对其他 queued run 调 `executeAgentRun`

注意：`run_stream` 创建的 run 会立即在当前 HTTP 请求内执行，并且 `stream_response=true`，不会由后台 worker 再处理。

## 7. 大模型交互链路

核心入口是 `callAgentProvider`。

### 7.1 凭据来源

优先级：

1. 环境变量 `OPENAI_API_KEY`
   - mode: `platform_api_key`
   - account: `OPENAI_ACCOUNT_ID`
2. `~/.codex/auth.json` 里的 `OPENAI_API_KEY`
   - mode: `platform_api_key`
3. `~/.codex/auth.json` 里的 `tokens.access_token`
   - mode: `chatgpt_access_token`
   - account: `tokens.account_id`
4. 没有凭据时走本地 mock：`Mock assistant: <input_text>`

### 7.2 Base URL 和模型

`resolveAgentBaseURL`：

| 条件 | Base URL |
| --- | --- |
| `OPENAI_BASE_URL` 有值 | 使用环境变量，去掉末尾 `/` |
| `chatgpt_access_token` | `https://chatgpt.com/backend-api/codex` |
| 其他 | `https://api.openai.com/v1` |

最终请求地址都是：

```text
{baseURL}/responses
```

模型：

- 默认模型：`gpt-5-mini`
- ChatGPT access token 模式下，如果模型为空或默认值，会映射为 `gpt-5.4-mini`

### 7.3 请求体构造

`buildAgentResponsesRequestBody` 构造 Responses 请求：

| 字段 | 说明 |
| --- | --- |
| `model` | 解析后的模型 |
| `input` | 用户输入、历史消息列表、或工具输出 |
| `previous_response_id` | 平台 API 模式下用于续接上下文 |
| `stream` | 有 `onDelta` 或 ChatGPT access token 模式时为 true |
| `tools` | 工具定义列表 |
| `tool_choice` | 有工具时为 `auto` |
| `parallel_tool_calls` | 有工具时为 true |
| `instructions` | session 的 `system_prompt` 或默认提示词 |
| `prompt_cache_key` | 按 mode/model/scene/instructions/tools 计算 |
| `store` | ChatGPT access token 模式下为 false |
| `include` | ChatGPT access token 模式下为空数组 |
| `client_metadata` | ChatGPT access token 模式下带 `x-codex-installation-id` |

上下文策略：

- 平台 API 模式：优先用 `previous_response_id` 续接上一轮。
- ChatGPT access token 模式：构造完整 `input` 消息列表，包含当前 session 下历史 user/assistant 消息；会过滤 provider 失败、429、调试 dump 这类不适合再次发给模型的 assistant 文本。

### 7.4 HTTP 和 SSE

`callResponsesProviderOnce`：

1. `POST {baseURL}/responses`
2. Header:
   - `Authorization: Bearer ***`
   - `Content-Type: application/json`
   - stream 时 `Accept: text/event-stream`
   - ChatGPT token 模式附加 Codex 风格 header，例如 `originator`、`session-id`、`thread-id`、`x-codex-installation-id`
   - account id 对应 `ChatGPT-Account-ID` 或 `OpenAI-Account-ID`
3. stream 响应走 `readAgentProviderSSEWithIdleTimeout`
4. 非 stream 响应直接 JSON decode

SSE 解析关注事件：

| Provider event type | 处理 |
| --- | --- |
| `response.output_text.delta` | 累加输出文本并触发 `onDelta` |
| `response.output_text.done` | 记录完整 done 文本 |
| `response.output_item.done` | 如果是 `function_call`，转成工具调用 |
| `response.completed` | 如果没有 delta，用 done 文本补齐 |

超时：

- 响应头超时默认 30 秒。
- SSE idle 超时默认 45 秒。
- 非 stream 请求总超时默认 45 秒。
- 可通过 `AGENT_PROVIDER_STREAM_HEADER_TIMEOUT`、`AGENT_PROVIDER_RESPONSE_HEADER_TIMEOUT` 覆盖部分超时。

### 7.5 重试和 auth reload

`callResponsesProviderWithAuthReloadRetry` 默认最多 2 次：

- 401、403、429 且凭据来自 `~/.codex/auth.json` 时，会强制重新读取 auth 文件。
- 408、429、5xx、网关错误、超时、EOF、连接重置等临时错误会重试。
- 如果已经收到 delta，则失败后不再重试，避免输出混乱。
- 重试、失败、auth reload 都会通过 `log` SSE 事件发给前端。

## 8. 工具调用系统

工具是否启用由 `tool_policy_json` 和 scene 决定。

默认规则：

- 如果 `scene_code=agent_regression_page`，且 policy enabled，会默认打开 Codex CLI 工具和验证工具。
- `tool_policy_json` 可指定 allowed roots、读写能力、命令能力、浏览器能力、图片检查能力、最大轮次和输出限制。

当前可暴露给大模型的工具：

| 工具名 | 执行函数 | 作用 |
| --- | --- | --- |
| `read_project_file` | `executeAgentReadProjectFile` | 读取允许根目录内 UTF-8 文本文件，支持行范围 |
| `glob` | `executeAgentCodexCLIGlob` | 按 glob 查文件/目录 |
| `grep` | `executeAgentCodexCLIGrep` | 搜索文件内容，支持 regex、上下文行、文件类型和 glob |
| `edit` | `executeAgentCodexCLIEdit` | 小范围编辑文本文件，支持 dry run、replace、insert、range 操作 |
| `delete_file` | `executeAgentCodexCLIDeleteFile` | 删除单个非关键文件，默认需要确认短语 |
| `run_command` | `executeAgentRunCommand` | 在允许目录内运行非交互命令，带超时和输出上限 |
| `browser_check` | `executeAgentBrowserCheck` | 内置 Playwright Chromium 检查页面、截图、console/page/request 错误 |
| `inspect_image` | `executeAgentInspectImage` | 检查截图/图片尺寸、空白、颜色集中度和 sha256 |
| `model_request_debug` | Go 内部合成 | 不给模型调用，用于把每次 provider 请求的脱敏摘要作为 tool result 展示给前端 |

工具循环：

1. `callAgentProviderWithTools` 先按 prompt 判断是否要执行预读工具。
2. 发起 provider 请求，携带工具定义。
3. 如果模型返回 `function_call`：
   - `executeAgentToolCall` 执行工具。
   - 工具结果通过 SSE `tool` 发给前端。
   - 工具结果再作为 `function_call_output` 回传给模型。
4. 重复直到模型不再请求工具，或达到 `max_tool_rounds`。
5. 达到上限时，再发一条限制提示，让模型基于已有结果收尾。

## 9. 数据状态模型

### 9.1 `agent_session`

Go 模型：`model/base/agent_session.go`

关键字段：

- `agent_session_id`
- `session_key`
- `scene_code`
- `title`
- `status`
- `user_id`
- `system_prompt`
- `model`
- `tool_policy_json`
- `mcp_policy_json`
- `last_response_id`
- `last_active_time`

作用：

- 表示一段对话上下文。
- 保存系统提示词、模型、工具策略。
- 保存上一轮 provider response id，用于平台 API 续接上下文。

### 9.2 `agent_run`

Go 模型：`model/base/agent_run.go`

关键字段：

- `agent_run_id`
- `agent_session_id`
- `request_id`
- `trigger_type`
- `status`
- `current_step`
- `worker_id`
- `lease_expire_time`
- `heartbeat_time`
- `request_json`
- `result_json`
- `error_msg`
- `started_at`
- `finished_at`

状态：

- `queued`
- `running`
- `completed`
- `failed`
- `cancelled`

作用：

- 表示一次用户输入到模型输出的任务。
- `request_json` 保存本次请求的输入和运行参数。
- `result_json` 保存 provider 输出、usage、tool results、debug requests。

### 9.3 `agent_message`

Go 模型：`model/base/agent_message.go`

关键字段：

- `agent_message_id`
- `agent_session_id`
- `agent_run_id`
- `role`
- `message_type`
- `content_text`
- `content_json`
- `seq_no`
- `source`
- `status`

作用：

- 保存 user 和 assistant 消息。
- 流式输出时 assistant 消息先 running，再持续更新 content_text，最终写入 content_json。

## 10. 前端调用链路

页面入口：

- `collect/frontend/page_data/index.yml` 注册 `frontend.agent_regression`
- 页面文件：`collect/frontend/page_data/data/system/agent_regression.json`

自定义 action 注册位置：

- `/data/project/sport-ui/src/main.tsx:123-129`

注册项：

- `agent-run-stream`
- `agent-run-cancel`
- `agent-tools-sync`
- `agent-workspace-sync`
- `agent-workspace-select`
- `agent-ui-init`

### 10.1 页面 JSON 中怎么请求

页面里不是直接通过普通 `ajax` 请求 `/template_data/agent/run_stream`，而是先走低代码 action 链：

1. `update-store`
   - 从当前 pane 的富文本 form 字段 `input_text` 提取纯文本。
   - A pane 写入 `pendingInputTextA` 和通用 `pendingInputText`。
   - B pane 写入 `pendingInputTextB` 和通用 `pendingInputText`。
   - 这一步会处理 `<p>`、`<br>`、`<li>`、HTML 实体、空白行等，把富文本压成后端要的文本。

2. `check`
   - 校验 `pendingInputTextA/B` 非空。

3. `submit-form`
   - 提交当前 pane 的 form。
   - A pane form: `agent-pane-a-form`
   - B pane form: `agent-pane-b-form`

4. `ajax -> agent.session_upsert`
   - API: `post:/template_data/data?service=agent.session_upsert`
   - `appendFormFields` 带上 pane form 中的 `session_key`、`title`、`scene_code`、`model`、`system_prompt`、`input_text`、`tool_policy_json`、`mcp_policy_json` 等字段。
   - `appendFields.agent_session_id` 从当前 pane 的 session store 里取。
   - `data.input_text` 使用 `pendingInputTextA/B`。
   - `adapt` 回填 `sessionInfoA/B`、`activeSessionIdA/B` 和通用 `sessionInfo`、`activeSessionId`。

5. `agent-run-stream`
   - 这是 sport-ui 自定义 action，不是 collect-ui 内置 ajax。
   - A pane 典型配置：

```json
{
  "tag": "agent-run-stream",
  "pane": "A",
  "formName": "agent-pane-a-form",
  "inputText": "${pendingInputTextA}",
  "uploadedImages": "${uploadedImagesA}",
  "sessionId": "${sessionInfoA.agent_session_id||activeSessionIdA||''}"
}
```

   - B pane 只把 `pane/formName/inputText/uploadedImages/sessionId` 换成 B。
   - 页面里还有顶层快捷配置，带 `enable: "${activePane==='A'}"` 或 `enable: "${activePane==='B'}"`，用于按当前 active pane 触发。

6. 发送完成后，页面里保留了一组 `enable: "${false}"` 的旧刷新链路：
   - `agent.message_query`
   - `agent.run_query`
   - `agent.session_query`
   - 这些现在不在发送后立即执行，流式 action 会自己维护消息、run、工具和 token usage 的 store。

终止按钮配置：

```json
{
  "tag": "agent-run-cancel",
  "pane": "A"
}
```

B pane 同理。按钮可见性由 `sendLoadingA/B` 控制。

### 10.2 agent-run-stream action 参数解析

实现文件：

- `/data/project/sport-ui/src/action/agent-run-stream.tsx`

默认导出函数读取低代码 action 配置：

| action 字段 | 默认值 | 用途 |
| --- | --- | --- |
| `pane` | `activePane` 或 `A` | 决定更新 A/B 哪套 store |
| `formName` | A 为 `agent-pane-a-form`，B 为 `agent-pane-b-form` | 读取当前 form 值 |
| `inputText` | `form.input_text` | 用户输入，支持富文本转纯文本 |
| `uploadedImages` | `uploadedImagesA/B` | 图片附件路径，会拼到输入后面 |
| `sessionId` | `activeSessionIdA/B` | 复用已有 session |

函数最后调用 `runAgentStream({ store, useApp, pane, formName, inputText, uploadedImages, sessionId })`，并立即 `return utils.getResult(true)`。也就是说低代码 action 本身不会等待流式任务结束，后续 UI 变化由异步函数持续写 store。

### 10.3 真正发起 HTTP 请求的位置

`runAgentStream` 内部执行：

1. 读取 form：

```ts
const form = getFormValue(store, formName);
const input = extractPromptText(cfg.inputText || form.input_text);
```

2. 拼图片附件：

```ts
const fullInput = input + (images.length ? `\n\n图片附件：${images.join("\n")}` : "");
```

3. 创建本地临时消息：

- user 临时消息：`agent_message_user_stream_<timestamp>`
- assistant 临时消息：`agent_message_assistant_stream_<timestamp>`
- assistant 状态先置为 `running`
- `agent_stream_live: true` 用于 UI 样式

4. 请求前先更新 UI store：

- `pendingInputTextA/B`
- `sendLoadingA/B`
- `messageListA/B`
- `toolCallListA/B`
- `agentPlanListA/B`
- `operationResult`
- 清空 form 的 `input_text`
- 清空 `uploadedImagesA/B`

5. 构造 payload：

```ts
const payload = {
  ...form,
  agent_session_id: sessionId,
  input_text: fullInput,
  stream_response: true
};
```

6. 发起请求：

```ts
const response = await fetch("/template_data/agent/run_stream", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(payload),
  signal: controller.signal
});
```

这里 `controller` 来自 `new AbortController()`，并保存到模块级变量 `activeRuns[pane]`，供终止按钮调用。

### 10.4 前端传数据契约

前端实际向两个后端入口传数据：

1. 先请求低代码服务 `agent.session_upsert`，用于准备 session。
2. 再请求 Gin 流式接口 `/template_data/agent/run_stream`，用于创建 run 并开始 SSE。

#### 10.4.1 `agent.session_upsert` 请求数据

请求方式：

```text
POST /template_data/data?service=agent.session_upsert
Content-Type: application/json
```

主要数据来源：

| 字段 | 来源 | 说明 |
| --- | --- | --- |
| `service` | URL query | 固定为 `agent.session_upsert` |
| `agent_session_id` | `sessionInfoA/B.agent_session_id` 或 `activeSessionIdA/B` | 有值时更新已有 session |
| `session_key` | 当前 pane form | 会话 key，可为空 |
| `title` | 当前 pane form | 会话标题 |
| `scene_code` | 当前 pane form | 常见为 `agent_regression_page` |
| `model` | 当前 pane form | 模型名 |
| `system_prompt` | 当前 pane form | 后续进入 provider `instructions` |
| `tool_policy_json` | 当前 pane form | 工具能力策略 |
| `mcp_policy_json` | 当前 pane form | 当前主要透传保存 |
| `input_text` | `pendingInputTextA/B` | 当前发送文本，用于后端生成标题和更新时间 |

典型 body 形态：

```json
{
  "agent_session_id": "agent_session_xxx",
  "session_key": "",
  "title": "新会话 A",
  "scene_code": "agent_regression_page",
  "model": "gpt-5-mini",
  "system_prompt": "你是 Codex CLI 风格的工程代理...",
  "tool_policy_json": "{\"enable\":true,\"codexcli\":true}",
  "mcp_policy_json": "",
  "input_text": "用户本次输入"
}
```

返回后前端主要接收 `data`，写入：

- `sessionInfoA/B`
- `sessionInfo`
- `activeSessionIdA/B`
- `activeSessionId`
- `operationResult`

#### 10.4.2 `/template_data/agent/run_stream` 请求数据

请求方式：

```text
POST /template_data/agent/run_stream
Content-Type: application/json
Accept: 由 fetch 自动处理，前端没有手动写 Accept
```

真实 body 由 `agent-run-stream` 构造：

```ts
const payload = {
  ...form,
  agent_session_id: sessionId,
  input_text: fullInput,
  stream_response: true
};
```

其中：

- `form` 来自 `store.getFormValue(formName)`。
- `sessionId` 来自低代码 action 配置的 `sessionId`，通常是 `sessionInfoA/B.agent_session_id || activeSessionIdA/B`。
- `fullInput = input + 图片附件`。
- 如果有图片，格式是：

```text
用户输入

图片附件：/path/a.png
/path/b.png
```

主要 body 字段：

| 字段 | 来源 | 后端用途 |
| --- | --- | --- |
| `agent_session_id` | action `sessionId` | 复用已有 session |
| `session_key` | form | 没有时后端会生成 `stream_` 前缀 key |
| `title` | form | 会话标题 |
| `scene_code` | form | 决定默认工具策略等场景行为 |
| `model` | form | 选择大模型 |
| `system_prompt` | form | 进入 provider `instructions` |
| `tool_policy_json` | form | 控制本地工具定义和执行权限 |
| `mcp_policy_json` | form | 当前保存/透传 |
| `input_text` | `fullInput` | 用户消息正文，必填 |
| `stream_response` | action 强制 true | 告诉后端创建流式 assistant 占位消息并走 SSE |

典型 body 形态：

```json
{
  "agent_session_id": "agent_session_xxx",
  "session_key": "",
  "title": "新会话 A",
  "scene_code": "agent_regression_page",
  "model": "gpt-5-mini",
  "system_prompt": "你是 Codex CLI 风格的工程代理...",
  "tool_policy_json": "{\"enable\":true,\"codexcli\":true}",
  "mcp_policy_json": "",
  "input_text": "用户本次输入",
  "stream_response": true
}
```

前端同时把 `AbortController.signal` 传给 fetch，后续 `agent-run-cancel` 可以中断这条流。

### 10.5 后端发送 SSE event 契约

后端统一用 `writeAgentStreamEvent` 发送 SSE：

```go
fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data)
c.Writer.Flush()
```

所以每个后端 event 都是：

```text
event: <event_name>
data: <json object>

```

#### 10.5.1 `start`

发送时机：

- `createAgentRun` 成功后立即发送。
- 表示后端已经创建真实 session/run。

发送位置：

- `plugins/agent_stream.go:119`

payload 字段：

| 字段 | 说明 |
| --- | --- |
| `agent_session_id` | 真实 session id |
| `session_key` | session key |
| `scene_code` | 场景码 |
| `title` | 会话标题 |
| `model` | 模型 |
| `agent_run_id` | 真实 run id |
| `request_id` | 请求 id |
| `status` | run 状态，通常是 `queued` 或刚创建时状态 |
| `current_step` | 当前步骤 |
| `result_json` | 创建时通常为空 |
| `error_msg` | 创建时通常为空 |
| `started_at` | 创建时通常为空，claim 后才有 |
| `finished_at` | 创建时通常为空 |
| `create_time` | run 创建时间 |
| `modify_time` | run 修改时间 |
| `last_active_time` | session 活跃时间，若 session 字段未被 run 覆盖则存在 |

示例：

```text
event: start
data: {"agent_session_id":"agent_session_xxx","session_key":"sess_xxx","agent_run_id":"agent_run_xxx","request_id":"req_xxx","status":"queued","current_step":"queued"}
```

#### 10.5.2 `delta`

发送时机：

- provider SSE 解析到模型文本增量时。

发送位置：

- `plugins/agent_stream.go:125-128`

payload：

```json
{
  "agent_run_id": "agent_run_xxx",
  "delta": "模型新增文本"
}
```

#### 10.5.3 `tool`

发送时机：

- 模型返回 function call 后，后端执行一个工具并得到结果时。
- 内部合成的 `model_request_debug` 也按 tool event 发给前端。

发送位置：

- `plugins/agent_stream.go:133-136`

payload：

```json
{
  "agent_run_id": "agent_run_xxx",
  "tool_result": {
    "call_id": "call_xxx",
    "name": "grep",
    "arguments": "{\"pattern\":\"run_stream\"}",
    "output": "{\"success\":true,\"count\":1}",
    "error": "",
    "duration_ms": 12
  }
}
```

`tool_result.name` 可能是：

- `read_project_file`
- `glob`
- `grep`
- `edit`
- `delete_file`
- `run_command`
- `browser_check`
- `inspect_image`
- `model_request_debug`

#### 10.5.4 `log`

发送时机：

- provider 请求重试、auth reload、失败、取消、等待重试等运行日志。

发送位置：

- `plugins/agent_stream.go:137-145`

payload 是运行时动态 map，后端会强制追加 `agent_run_id`。常见字段：

| 字段 | 说明 |
| --- | --- |
| `agent_run_id` | 当前 run |
| `type` | 日志类型，例如 `provider_attempt`、`auth_reload_retry`、`provider_failed`、`provider_retry_wait` |
| `message` | 给用户看的日志文本 |
| `create_time` | 日志创建时间 |
| `attempt` | 当前尝试次数 |
| `max_attempts` | 最大尝试次数 |
| `mode` | provider 凭据模式 |
| `source` | 凭据来源 |
| `elapsed_ms` | 总耗时 |
| `elapsed_text` | 总耗时文本 |
| `error` | 错误详情，失败时可能存在 |
| `error_summary` | 错误摘要 |
| `provider_status` | HTTP 状态摘要 |

示例：

```text
event: log
data: {"agent_run_id":"agent_run_xxx","type":"provider_retry_wait","message":"等待 10秒 后继续重试","attempt":2,"max_attempts":2}
```

#### 10.5.5 `done`

发送时机：

- 成功时：模型执行完成、消息和 run 状态已经落库后发送。
- 失败时：后端会先发送一个失败状态的 `done`，再发送 `error`。

发送位置：

- 成功：`plugins/agent_stream.go:164`
- 失败：`plugins/agent_stream.go:154`

payload 来自最新 `agent.run_query` 结果，字段与 `start` 类似，但通常会包含最终字段：

| 字段 | 说明 |
| --- | --- |
| `agent_session_id` | session id |
| `agent_run_id` | run id |
| `request_id` | 请求 id |
| `status` | `completed`、`failed` 或 `cancelled` |
| `current_step` | `completed`、`failed` 或 `cancelled` |
| `result_json` | 成功时包含 provider 结果、usage、tool_results、debug_requests |
| `error_msg` | 失败或取消原因 |
| `started_at` | 开始时间 |
| `finished_at` | 完成时间 |
| `create_time` | 创建时间 |
| `modify_time` | 修改时间 |

#### 10.5.6 `error`

发送时机：

- `executeAgentRunWithStream` 返回错误。
- 当前实现会先发 `done`，再发 `error`，两个 payload 基本相同。

发送位置：

- `plugins/agent_stream.go:155`

额外字段：

| 字段 | 说明 |
| --- | --- |
| `msg` | `err.Error()` |
| `error_msg` | 如果原 run 没有错误信息，会填入 `err.Error()` |

示例：

```text
event: error
data: {"agent_run_id":"agent_run_xxx","status":"failed","msg":"Agent provider 请求失败...","error_msg":"Agent provider 请求失败..."}
```

### 10.6 前端接收 SSE event 契约

前端没有使用 `EventSource`，而是用 `fetch + response.body.getReader()` 读取 POST SSE。

接收入口：

- `/data/project/sport-ui/src/action/agent-run-stream.tsx:1094-1206`

读取逻辑：

1. 拿 reader 和 decoder：

```ts
const reader = response.body.getReader();
const decoder = new TextDecoder();
let buffer = "";
```

2. 循环读取 chunk：

```ts
const result = await reader.read();
buffer += decoder.decode(result.value, { stream: true });
```

3. 按空行分包：

```ts
let index = buffer.indexOf("\n\n");
while (index >= 0) {
  const parsed = parseSSEBlock(buffer.slice(0, index));
  if (parsed) {
    handleEvent(parsed.event, parsed.data);
  }
  buffer = buffer.slice(index + 2);
  index = buffer.indexOf("\n\n");
}
```

4. 流结束后，如果 `buffer.trim()` 还有残留，再解析一次。

`parseSSEBlock` 的规则：

- 默认 event 是 `message`。
- 遍历每行：
  - `event:` 开头取事件名。
  - `data:` 开头拼接 JSON 字符串。
- 最后 `JSON.parse(raw)`。
- 解析失败直接返回 `null`，不会抛到外层。

### 10.7 前端 event 回调怎么更新 UI

所有事件统一进入 `handleEvent(event, data)`。

| SSE event | 前端处理 |
| --- | --- |
| `start` | 后端已创建真实 session/run。前端用返回的 `agent_session_id`、`agent_run_id` 替换临时 id，更新 `activeSessionIdA/B`、`sessionInfoA/B`、`runInfoA/B`、`runListA/B`，初始化工具列表和计划状态 |
| `delta` | 把 `data.delta` 追加到 assistant 临时消息 `content_text`；调用 `ingestPlan` 从模型输出中解析计划；刷新 `messageListA/B` 并滚动到底部 |
| `tool` | 取 `data.tool_result`，合并进当前 run 的 `result_json.tool_results`；调用 `syncTools` 更新工具浮窗；调用 `onToolPlan` 推进计划；更新 `operationResult` 为当前工具名 |
| `log` | 把 provider 日志追加进 `agentProviderLogListA/B`，最多保留 30 条；同时把 `[运行日志] xxx` 追加到 assistant 消息里 |
| `done` | 把 assistant 状态置为 completed；合并最终 run 数据；更新 `runInfo/runList`、工具列表、token usage；调用 `finishPlan` 完成或失败计划；关闭 loading |
| `error` | 抛出异常，进入 catch；catch 会把 assistant 状态置为 failed，工具浮窗置为 failed/cancelled，计划置为失败或终止 |

重要细节：

- `done` 事件只更新前端当前临时 assistant 消息的状态，不会主动再拉一次 `agent.message_query`。
- 如果 `done` 没带 `result_json`，前端会沿用当前 run 里已经缓存的 `result_json`。
- `tool` 事件会实时更新工具浮窗，因此工具结果不必等最终 `done`。
- `log` 事件既进日志列表，也进聊天正文，所以用户能看到 auth reload、provider retry、超时等信息。
- 如果 SSE 正常结束但没有收到 `done`，前端兜底把 assistant 标记为 completed，并用缓存工具结果同步工具浮窗。

### 10.8 错误和取消处理

错误分支：

1. 如果 `fetch` 返回 `application/json`，说明后端没有进入 SSE，前端读取 JSON；`success=false` 时抛出 `body.msg`。
2. 如果 HTTP 非 2xx 或没有 `response.body`，抛出 `流式请求失败: <status>`。
3. `event=error` 也会抛出异常。
4. catch 中判断是否 `AbortError` 或 `controller.signal.aborted`：
   - abort：文本为 `用户已终止本次请求`
   - 其他：文本为 `请求失败：xxx`

catch 更新：

- assistant 消息 status 置为 `failed`
- `messageListA/B` 更新失败消息
- `syncTools` 状态置为 `cancelled` 或 `failed`
- `finishPlan` 状态置为 `计划已终止` 或 `计划执行失败`
- `sendLoadingA/B` 关闭
- `operationResult` 写入失败/终止提示

终止 action：

1. `agent-run-cancel` 先从 `activeRuns[pane]` 找 `AbortController` 并执行 `abort()`，前端本地流立即中断。
2. 再取当前 run id：
   - 优先 `active.runId`
   - 其次 `runInfoA/B.agent_run_id`
3. POST `/template_data/agent/run_cancel`：

```ts
await fetch("/template_data/agent/run_cancel", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ agent_run_id: runId })
});
```

4. 即使这个 POST 失败，前端仍认为本地 abort 生效，会关闭 loading 并提示已终止。

### 10.9 前端相关 store 字段

按 pane 分 A/B，同时维护一份通用字段给当前激活 pane 使用。

| 字段 | 作用 |
| --- | --- |
| `activeSessionIdA/B`、`activeSessionId` | 当前 session id |
| `sessionInfoA/B`、`sessionInfo` | 当前 session 信息 |
| `runInfoA/B`、`runInfo` | 当前 run 信息 |
| `runListA/B`、`runList` | run 列表 |
| `messageListA/B`、`messageList` | 聊天消息列表 |
| `sendLoadingA/B`、`sendLoading` | 当前 pane / 全局发送中状态 |
| `pendingInputTextA/B`、`pendingInputText` | 发送中的输入快照 |
| `uploadedImagesA/B` | 图片附件 |
| `toolCallListA/B`、`toolCallList` | 工具调用展示列表 |
| `toolCallStatusA/B`、`toolCallStatus` | 工具调用状态 |
| `toolCallVisibleA/B`、`toolCallVisible` | 工具浮窗是否显示 |
| `toolCallRunIdA/B`、`toolCallRunId` | 工具结果对应的 run |
| `agentPlanListA/B`、`agentPlanList` | 从模型输出中解析的执行计划 |
| `agentProviderLogListA/B`、`agentProviderLogList` | provider 重试/auth/超时日志 |
| `agentTokenUsageTextA/B`、`agentTokenUsageText` | 从 run.result_json 中解析的 token 用量 |
| `operationResult` | 当前操作提示 |

### 10.10 对低代码化的影响

如果后续希望把前端也进一步低代码化，必须替代当前 `agent-run-stream` 自定义 action 的这些能力：

- POST SSE 读取。普通 `ajax` 只能等完整响应，不具备 chunk 增量处理。
- `AbortController` 和 `/run_cancel` 的双重取消。
- `start/delta/tool/log/done/error` 六类事件到多个 store 的增量映射。
- 工具结果的实时合并、展示格式化、浮窗同步。
- 计划解析和计划状态推进。
- 富文本输入转纯文本、图片附件拼接。
- A/B pane 双状态和通用状态同步。

## 11. 低代码化改造边界

### 11.1 已经适合保留为低代码的部分

这些部分已经是低代码服务，可以继续保留或细化：

- session 查询、保存、更新、删除。
- run 查询、保存、抢占、心跳、完成、失败、取消。
- message 查询、保存、流式内容更新、失败/取消更新。
- 页面 schema 和普通 CRUD action。

### 11.2 可以进一步低代码化的部分

这些逻辑目前在 Go 里，但可以拆成更配置化的服务编排：

- `createAgentRun` 里的参数归一化、run/message 初始保存。
- `getOrCreateAgentSession` 的 upsert 编排。
- run 状态机的服务名称和状态流转配置。
- tool policy 的默认值和 scene 绑定规则。
- provider 参数模板，例如 model、instructions、base_url、stream、tools、prompt_cache_key 的可配置化。

### 11.3 不建议纯配置化的部分

这些能力建议保留 Go 插件或专用 handler，再由低代码服务调用：

- SSE 长连接输出和 flush。
- provider 的 SSE 读取、idle timeout、响应头超时和错误分类。
- in-memory cancel map，也就是 `agentRunCancelMap`。
- 本地文件读取、编辑、删除、命令执行、浏览器执行、图片检查的安全边界。
- 大模型 tool call 多轮循环。
- artifact 文件访问的路径白名单和文件类型限制。

### 11.4 建议改造方向

建议按三层拆：

1. HTTP/SSE 传输层
   - 保留 `/template_data/agent/run_stream` 作为薄 handler。
   - 只负责 JSON 解析、SSE header、事件输出、请求取消。

2. 低代码编排层
   - 把 session/run/message 的状态流转尽量落在 `collect/agent/**/index.yml`。
   - 新增必要的低代码服务，例如 `agent.run_stream_prepare`、`agent.run_stream_finish`、`agent.provider_request_log_save`。
   - 让状态流转服务更集中，减少 Go 中散落的 `callAgentLowcodeService("agent.xxx")`。

3. Go 能力插件层
   - 保留 provider executor 和 tool executor。
   - 可以抽成低代码 module，例如：
     - `module: agent_provider_execute`
     - `module: agent_tool_execute`
   - module 输入/输出固定，低代码负责传参和落库。

## 12. 关键改造点

后续要改造时，优先看这些点：

1. `plugins/agent_stream.go`
   - 这是直接接口层。
   - 如果要把接口变薄，先从这里拆。

2. `plugins/module_agent_run.go`
   - `createAgentRun` 是流式和非流式共用的创建逻辑。
   - 低代码化时要避免复制两份创建流程。

3. `plugins/agent_runtime.go`
   - `callAgentProvider` 决定 mock、工具模式、ChatGPT token 模式、平台 API 模式。
   - `buildAgentResponsesRequestBody` 决定最终发给大模型的请求体。
   - `callResponsesProviderOnce` 是真实 HTTP 请求点。
   - `callAgentProviderWithTools` 是多轮工具调用核心。
   - `executeAgentRunWithStream` 是状态机和落库主流程。

4. `plugins/agent_tools.go`
   - 所有本地工具和安全限制都在这里。
   - 低代码页面只应配置 tool policy，不应绕过这些安全限制。

5. `/data/project/sport-ui/src/action/agent-run-stream.tsx`
   - 当前前端流式体验依赖这个自定义 action。
   - 若后续改成纯低代码 action，需要先解决 SSE 读取、增量更新 store、工具浮窗更新、取消请求这几件事。

## 13. 当前接口数量汇总

按不同层级统计：

| 层级 | 数量 | 说明 |
| --- | --- | --- |
| `RegisterAgentStreamRoutes` 直接注册路径 | 3 | `run_stream`、`run_cancel`、`artifact` |
| `RegisterAgentStreamRoutes` 按 method 计接口 | 4 | artifact 同时注册 GET 和 HEAD |
| Agent 低代码对外 HTTP 服务 | 6 | 走 `/template_data/data?service=agent.xxx` |
| Agent 低代码内部服务 | 21 | Go runtime 通过 `callAgentLowcodeService` 调用 |
| Agent 低代码服务总数 | 27 | session 6、run 12、message 9 |
| 可暴露给模型的业务工具 | 8 | read/glob/grep/edit/delete/run_command/browser/inspect_image |
| 内部调试工具结果 | 1 | `model_request_debug` |
