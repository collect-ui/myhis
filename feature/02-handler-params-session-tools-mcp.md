# 低代码 `handler_params` 多轮 Session / Tools / MCP 模块设计

## 1. 目标

基于当前仓库的低代码执行链，为“大体量、运行时间长、需要多轮对话、需要外部工具和 MCP”的场景设计一套可长期运行的模块方案。

这份设计的目标不是推翻现有低代码框架，而是：

- 保持 `collect/**/*.yml + plugins/*.go` 的现有模式
- 兼容当前 `handler_params` 的串行参数编排能力
- 增加多轮 `session`、长期运行 `run`、`tools`、`MCP`、恢复与审计能力
- 给后续实现提供明确的模块边界、数据模型、落地顺序、测试办法

本文档默认服务对象是：

- 长链路问答
- 代码/运维/文档类 agent 任务
- 需要跨多次请求保存上下文的会话
- 可能调用本地工具、内部 service、远程 MCP server 的任务
- 可能持续数分钟到数小时的后台执行任务

## 2. 当前 `handler_params` 运行机制梳理

### 2.1 宿主链路

当前仓库的低代码服务执行顺序里，`handler_params` 是前置插件之一：

- `collect/service_router.yml`
  - `handler_req_param`
  - `prevent_duplication`
  - `handler_cache`
  - `handler_params`

对应宿主实现位于替换依赖 `../collect`：

- `../collect/src/collect/service_imp/service_before_plugin.go`
- `../collect/src/collect/service_imp/service_after_plugin.go`

关键结论：

- `handler_params` 是串行执行
- 每个处理器共享同一份 `template.Params`
- 每个处理器可通过 `save_field` 将结果写回参数上下文
- 后续处理器和主模块都读取这份不断增长的上下文
- 任一处理器失败，会直接中断整个服务

### 2.2 `HandlerParam` 的核心能力

`../collect/src/collect/config/router_all.go` 中的 `HandlerParam` 说明了当前低代码处理器的声明式能力：

- `key`：选择处理器
- `enable`：条件启用
- `field` / `fields`
- `foreach` / `item`
- `service`
- `template` / `err_msg`
- `save_field`
- `value` / `value_tpl`
- `operation`
- `second`
- `loop_max`

这意味着现有框架已经具备：

- 参数变换
- 数组遍历
- 条件过滤
- 子服务调用
- 轮询等待
- 结果回填

但它本质上仍然是“单请求内的串行编排器”。

### 2.3 现有插件类型

当前仓库里 `plugins/handler_params_*.go` 已经覆盖三类能力：

- 纯数据处理：`multi_arr`、`rename_field`、`xml2json`
- IO/副作用：`read_file`、`local_file_write`、`to_local_file`
- 外部连接：`shell`、`shell_term`、`sftp`

`collect` 核心里还自带：

- `service2field`
- `filter_arr`
- `update_array`
- `update_array_from_array`
- `session_add`
- `session_get`
- `handler_cache`

### 2.4 当前模式的优点

- 低代码可读性较强
- 子服务复用自然
- 和现有 `collect/**/*.yml` 体系兼容
- 小到中型编排非常高效
- 很适合把“模型调用”看成一个可复用服务节点

### 2.5 当前模式的缺口

对多轮 agent / 长任务场景，当前链路存在以下缺口：

- `session_add/session_get` 只适合存少量键值，不适合保存完整多轮会话历史
- 没有“运行实例 run”的一等实体
- 没有任务恢复、租约、心跳、续跑机制
- 没有消息窗口裁剪、摘要、归档能力
- 没有 tool call / MCP call 的标准审计表
- 没有审批流、人工中断、超时取消机制
- `handler_params` 串行执行适合单次请求，不适合直接承载数小时任务
- `shell_term` 是长连接交互模型，但它依赖 websocket 和单进程内状态，不适合作为通用长期运行框架

## 3. 设计原则

### 3.1 保持低代码优先

本设计不建议把整套 agent 逻辑硬编码到一个大 Go 模块里。

推荐分层：

- 持久化与调度：Go
- 模型 / tools / MCP 适配：Go
- 业务编排：低代码 service + `handler_params`

### 3.2 把“长任务”从“单 HTTP 请求”里拆出来

单次 HTTP 请求只负责：

- 创建会话
- 提交用户输入
- 创建 run
- 查询 run 状态
- 拉取消息和工具调用结果

真正耗时执行应该进入后台 worker。

### 3.3 会话、运行、消息、工具调用分离

至少要拆成四个实体：

- `session`：会话身份与配置
- `message`：多轮消息历史
- `run`：一次任务执行实例
- `tool_call`：工具/MCP 调用审计

### 3.4 幂等、可恢复、可审计优先

对长期运行系统，优先级高于“写起来快”的是：

- 可以重试
- 可以恢复
- 可以追踪
- 可以人工介入

## 4. 总体方案

## 4.1 模块拆分

建议新增一个“Agent Runtime”子域，但仍遵守本仓库模式：

- `model/base/agent_session.go`
- `model/base/agent_message.go`
- `model/base/agent_run.go`
- `model/base/agent_tool_server.go`
- `model/base/agent_tool_call.go`
- `model/base/agent_checkpoint.go`
- `model/base/agent_artifact.go`
- `model/register.go` 中注册模型

Go 模块建议分三层：

- `plugins/module_agent_run.go`
  - 创建 run、入队、取消、恢复
- `plugins/module_agent_worker.go`
  - 后台执行一个 run
- `plugins/module_agent_openai.go`
  - 统一封装 Responses API / stream
- `plugins/module_agent_mcp.go`
  - 统一封装 MCP server 调用与审批

低代码入口建议新建：

- `collect/agent/session/index.yml`
- `collect/agent/run/index.yml`
- `collect/agent/tool_server/index.yml`
- `collect/agent/message/index.yml`

前端或上游系统只打这些 service，不直接碰底层 OpenAI/MCP 实现。

## 4.2 关键实体设计

### `agent_session`

表示一个长期存在的会话。

核心字段建议：

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
- `context_summary`
- `last_response_id`
- `last_active_time`
- `expire_time`
- `create_time`
- `modify_time`
- `is_delete`

说明：

- `last_response_id` 用于对接 Responses API 多轮上下文
- `context_summary` 用于历史压缩
- `scene_code` 区分不同业务场景

### `agent_message`

表示会话中的输入输出消息。

核心字段建议：

- `agent_message_id`
- `agent_session_id`
- `run_id`
- `role`
- `message_type`
- `content_text`
- `content_json`
- `seq_no`
- `source`
- `token_count`
- `status`
- `create_time`

`message_type` 可区分：

- `user`
- `assistant`
- `system`
- `summary`
- `tool_call`
- `tool_result`
- `mcp_approval_request`
- `mcp_approval_response`

### `agent_run`

表示一次后台执行。

核心字段建议：

- `agent_run_id`
- `agent_session_id`
- `request_id`
- `trigger_type`
- `status`
- `current_step`
- `worker_id`
- `lease_expire_time`
- `heartbeat_time`
- `retry_count`
- `max_retry`
- `error_msg`
- `started_at`
- `finished_at`
- `create_time`

`status` 建议固定枚举：

- `queued`
- `running`
- `waiting_approval`
- `waiting_tool`
- `completed`
- `failed`
- `cancelled`
- `expired`

### `agent_tool_server`

存放可被会话引用的工具源配置。

核心字段建议：

- `agent_tool_server_id`
- `server_label`
- `server_type`
- `base_url`
- `auth_type`
- `auth_config_json`
- `tool_filter_json`
- `approval_policy`
- `timeout_second`
- `enabled`
- `create_time`

`server_type`：

- `openai_builtin`
- `function`
- `mcp`
- `internal_service`

### `agent_tool_call`

记录工具调用审计。

核心字段建议：

- `agent_tool_call_id`
- `agent_session_id`
- `agent_run_id`
- `server_label`
- `tool_name`
- `tool_type`
- `request_json`
- `response_json`
- `status`
- `approval_status`
- `started_at`
- `finished_at`
- `error_msg`

### `agent_checkpoint`

用于长任务恢复。

核心字段建议：

- `agent_checkpoint_id`
- `agent_run_id`
- `checkpoint_no`
- `checkpoint_type`
- `state_json`
- `create_time`

## 4.3 执行链路

推荐链路：

1. 前端调用 `agent.session_upsert`，创建或获取会话
2. 前端调用 `agent.run_create`，提交本轮用户消息
3. 服务写入 `agent_message(user)`，创建 `agent_run(queued)`
4. 后台 worker 抢占 `queued` run，更新为 `running`
5. worker 装配上下文窗口、工具集、MCP server、模型参数
6. 调用 OpenAI Responses API
7. 如发生 tool / MCP 调用：
   - 记录 `agent_tool_call`
   - 必要时进入 `waiting_approval`
   - 批准后继续执行
8. 产生 assistant 输出后，写回 `agent_message`
9. 更新 `agent_session.last_response_id`
10. run 结束为 `completed` / `failed`

## 4.4 与现有 `handler_params` 的结合方式

本设计不建议让单个 `handler_params` 直接包住整个长任务，而建议让 `handler_params` 只负责编排单步 service。

建议新增以下低代码 service：

- `agent.session_query`
- `agent.session_upsert`
- `agent.message_query`
- `agent.run_create`
- `agent.run_query`
- `agent.run_cancel`
- `agent.run_retry`
- `agent.tool_server_query`

然后在业务 service 里用 `handler_params` 做轻量编排，例如：

- 先查 session
- 再补默认 prompt / model
- 再创建 run
- 再回填结果字段

适合新增的 `handler_params` 插件：

- `agent_session_get_or_create`
- `agent_message_append`
- `agent_context_window_build`
- `agent_summary_compact`
- `agent_run_enqueue`
- `agent_run_wait`
- `agent_tool_registry_load`

其中 `agent_run_wait` 只适合短轮询，不负责真正执行任务。

## 5. 多轮 Session 设计

## 5.1 为什么不能只用现有 HTTP Session

当前 `session_add/session_get` 适合：

- 保存 `user_id`
- 保存当前页面轻量状态
- 保存少量标志位

不适合：

- 长期保存对话历史
- 保存几十到几百轮消息
- 跨设备/跨进程恢复
- 任务审计
- 长任务恢复

因此多轮会话必须落库。

## 5.2 会话窗口策略

推荐三层上下文：

- 热窗口：最近 N 轮用户/助手消息
- 工具摘要：最近几次关键 tool 结果摘要
- 会话摘要：对更早历史进行 compact summary

实际组装给模型时按顺序：

1. system prompt
2. 会话摘要
3. 最近消息窗口
4. 当前用户输入

## 5.3 `previous_response_id` 策略

建议同时保留两种能力：

- 默认使用 `previous_response_id` 维持 OpenAI 原生多轮链路
- 在需要重建上下文时，使用本地消息重放

原因：

- 只依赖 `previous_response_id`，迁移和恢复弹性不足
- 只依赖本地消息重放，成本更高

推荐做法：

- 正常轮次：优先使用 `last_response_id`
- 当检测到链路丢失/过期/切模型：退回本地窗口重建

## 6. Tools 设计

## 6.1 工具分类

建议统一抽象为四类：

- OpenAI built-in tools
- 内部 service tool
- 本地 function tool
- 远程 MCP tool

### OpenAI built-in

例如：

- `web_search`
- `file_search`

### 内部 service tool

把当前低代码 service 暴露给 agent，例如：

- `webshell.workspace_project_query`
- `project.server_dir_tree_shell`

这类最适合本仓库复用。

### 本地 function tool

用于少量高价值 Go 逻辑。

### 远程 MCP tool

对接标准 MCP server。

## 6.2 工具注册方式

建议不要把工具注册写死在代码里。

推荐：

- 默认工具集放 DB 表 `agent_tool_server`
- 场景级白名单放 `agent_session.tool_policy_json`
- 运行时拼装 `tools`

## 6.3 内部 service tool 适配

这是最值得优先落地的部分。

做法：

- 增加一个 `internal_service` 类型工具
- 输入 schema 映射到 `collect` service 参数
- worker 执行时调用 `ts.ResultInner(...)`

好处：

- 直接复用现有低代码能力
- 权限边界清晰
- 不必把所有东西改成 MCP

## 7. MCP 设计

## 7.1 定位

MCP 在这里的角色不是替代内部 service，而是：

- 连接外部系统
- 连接通用文档/知识源
- 连接标准化远程工具

## 7.2 MCP 接入层

建议单独实现 `module_agent_mcp.go`，负责：

- 列工具
- 调工具
- 处理 approval
- 标准化返回
- 错误与超时处理

不要把 MCP 细节散落到每个业务 service。

## 7.3 MCP 运行策略

每个 MCP server 建议支持配置：

- `server_label`
- `base_url`
- `headers`
- `approval_policy`
- `read_only_only`
- `timeout_second`
- `retry_policy`

工具调用时记录：

- 传入参数
- 返回结果
- 审批请求
- 审批结果
- 重试次数

## 7.4 MCP 审批流

对于写操作或高风险工具，建议进入：

- `waiting_approval`

由前端或人工接口提交：

- `approve`
- `reject`

被拒绝时：

- 写入 `agent_message`
- 更新 `agent_tool_call`
- run 继续或失败，由配置决定

## 8. 长期运行设计

## 8.1 Worker 模型

建议后台 worker 独立于 HTTP 请求。

最小实现可以先用本进程 goroutine + DB 抢锁，后续再演进成独立 worker 进程。

worker 循环逻辑：

1. 抢占 `queued` run
2. 写入 `worker_id`
3. 更新 `lease_expire_time`
4. 周期性 heartbeat
5. 每完成一步落 checkpoint
6. 完成后释放 run

## 8.2 租约与恢复

若 worker 崩溃：

- 超过 `lease_expire_time` 的 `running` run 可被回收
- 回收器将其转为 `queued` 或 `failed`
- 如存在 checkpoint，从最近 checkpoint 恢复

## 8.3 超时控制

至少三层超时：

- HTTP 请求超时
- 单次模型调用超时
- 单次 tool / MCP 调用超时
- run 总超时

## 8.4 结果输出策略

建议同时保留：

- 最终结果：落 `agent_message`
- 过程日志：落 `agent_checkpoint` / `agent_tool_call`
- 前端进度查询：查 `agent_run`

如果后续做流式输出，再单独补：

- websocket/sse 推送层

但核心数据仍要先落库。

## 8.5 清理与归档

长期运行必须有清理策略：

- 完成超过 N 天的 run 归档
- 大消息内容可转 `artifact`
- checkpoint 保留最近若干份
- 失败 run 定期统计与告警

## 9. 推荐落地顺序

## Phase 1: 最小可用版

目标：

- 支持多轮会话
- 支持后台 run
- 支持 OpenAI Responses API
- 不做 MCP 审批

落地内容：

- 表：`agent_session`、`agent_message`、`agent_run`
- 服务：`session_upsert`、`message_query`、`run_create`、`run_query`
- Go：`module_agent_openai`、`module_agent_run`
- 使用 `previous_response_id`

## Phase 2: 工具化

目标：

- 支持内部 service tool
- 支持 OpenAI built-in tools

落地内容：

- 表：`agent_tool_server`、`agent_tool_call`
- Go：`module_agent_tool_registry`
- 增加 `internal_service` tool adapter

## Phase 3: MCP

目标：

- 支持远程 MCP server
- 支持审批流

落地内容：

- Go：`module_agent_mcp`
- 服务：`tool_call_approve`、`tool_call_reject`
- 状态：`waiting_approval`

## Phase 4: 长期运行增强

目标：

- 支持 checkpoint
- 支持恢复
- 支持会话摘要压缩

落地内容：

- 表：`agent_checkpoint`
- 插件：`agent_summary_compact`
- 回收器 / lease sweeper

## 10. 建议文件布局

建议最终代码布局：

```text
model/base/
  agent_session.go
  agent_message.go
  agent_run.go
  agent_tool_server.go
  agent_tool_call.go
  agent_checkpoint.go

plugins/
  module_agent_openai.go
  module_agent_run.go
  module_agent_worker.go
  module_agent_mcp.go
  handler_params_agent_session_get_or_create.go
  handler_params_agent_context_window_build.go
  handler_params_agent_run_enqueue.go
  handler_params_agent_summary_compact.go

collect/agent/session/index.yml
collect/agent/run/index.yml
collect/agent/message/index.yml
collect/agent/tool_server/index.yml

feature/
  02-handler-params-session-tools-mcp.md
```

## 11. 测试设计

## 11.1 单元测试

优先给纯逻辑部分补测试：

- 会话窗口裁剪
- 消息摘要压缩
- tool schema 组装
- MCP 请求/响应转换
- run 状态机流转
- lease 过期判定

建议命令：

```bash
go test ./...
```

如果拆出独立包，至少覆盖：

- `./plugins/...`
- `./model/...`

## 11.2 集成测试

至少做三类：

### A. OpenAI 假服务集成

用本地 mock server 模拟：

- 正常回答
- tool call
- 超时
- 429 / 500

### B. MCP 假服务集成

模拟：

- list tools
- call tool
- approval request
- 失败和重试

### C. 数据库恢复测试

流程：

1. 创建 run
2. worker 执行到中间步骤
3. 人工终止进程
4. 重启后执行回收器
5. 验证 run 能恢复或标记失败

## 11.3 低代码回归测试

要补一组低代码 service 验收用例：

- 创建 session
- 连续两轮对话
- message 查询顺序正确
- run 状态轮询正确
- tool call 审计正确
- MCP 审批前后状态正确

## 11.4 长稳测试

这是本需求的重点。

至少要做：

- 1000 次短 run 压测
- 100 次带 tool 调用 run
- 20 次带 MCP 调用 run
- 进程中途重启恢复测试
- 数据表增长与清理策略测试

关注指标：

- run 完成率
- 平均耗时
- 超时率
- 重试率
- 僵尸 run 数量
- lease 回收成功率

## 12. 运维与长期运行建议

## 12.1 必须记录的日志

- session 创建/关闭
- run 状态变更
- tool call 开始/结束
- MCP approval 请求/结果
- checkpoint 保存
- worker 抢锁与续租

## 12.2 告警

建议最少监控：

- `running` 超过阈值未 heartbeat
- `failed` run 日增长异常
- MCP 超时率异常
- OpenAI 429/5xx 异常

## 12.3 数据清理

建议定时任务：

- 清理过期 session
- 归档旧消息
- 删除旧 checkpoint
- 汇总 tool call 指标

## 13. 风险与规避

### 风险 1：把所有业务都塞进一个大模块

后果：

- 不可复用
- 不可测试
- 和低代码体系脱节

规避：

- runtime、tool、MCP、业务编排分层

### 风险 2：继续依赖单 HTTP 请求承载长任务

后果：

- 超时
- 断连即丢状态
- 难恢复

规避：

- 必须引入 `run + worker + checkpoint`

### 风险 3：只存 `previous_response_id` 不存本地消息

后果：

- 无法恢复
- 无法审计

规避：

- 本地消息落库是硬要求

### 风险 4：MCP 直接放开写能力

后果：

- 高风险副作用
- 难审计

规避：

- 默认审批
- 按 server / tool 白名单控制

## 14. 本次设计结论

结合当前仓库现状，最合理的路线不是“新写一个巨大对话模块顶替低代码”，而是：

- 保持现有 `handler_params` 作为声明式编排层
- 新增持久化会话与运行时子域
- 让耗时任务进入后台 worker
- 把工具调用统一抽象为 `tool registry`
- 对外部标准工具统一走 MCP 适配层
- 用数据库记录多轮消息、run、tool call、checkpoint

简化成一句话：

当前 `handler_params` 适合“编排一次任务”，接下来要补的是“让一次任务可以演进成一个可恢复、可审计、可长期运行的 agent runtime”。

## 15. 后续实现入口

建议按下面顺序开工：

1. 建表与 model 注册
2. 做 `session_upsert / run_create / run_query / message_query`
3. 做 `module_agent_openai.go`
4. 做 worker 与 lease
5. 做内部 service tool
6. 最后接 MCP 与审批流

不要一开始就把 MCP、流式输出、长稳恢复全部一起写完，否则风险过高。
