
# 需求
这个是我ai agent 
http://192.168.232.130:8015/collect-ui#/collect-ui/framework/agent_regression
但是这个不好用，需要重构，后端可以接着用，只要前端体验感不好

# 要求
- 认真理解直接文件
- 美化界面，操作方便，页面类似web 版本deepseek
- 先出原型设计，以html 形式输出输出/data/project/sport/feature/原型设计/agent
- 页面说明到/data/project/sport/feature/原型设计/agent/readme.md
- 以尽量以低代码形式实现agent,实在实现不了，允许自定组件，自定义模块
- 支持左右分屏、上下分屏，支持多tab 会话session ,并且支持左右和上下
- 支持图片上传
- 之前切换模型
- 右上角支持显示session 的概要
- 使用低代码技能/data/project/sport/.codex/skills/lowcode-json/SKILL.md
先备份json
# 开发流程
- 梳理功能架构、低代码开发模式
- 出原型设计，出完自己验证一下，是否美观、操作简单方便、基本功能都有
- 开发 ，功能拆解、研发
- 出测试计划、验收计划
- 验证和测试 ，测试回归
- 分屏左右和上下http://127.0.0.1:8009/collect-ui#/collect-ui/framework/webshell 参考这个，paneList

# 开发要求
- 架构梳理到本文，不要覆盖本部的东西，只能添加
- 改动什么文件必须写日志，到文本
#低代码改造检查：

- 先定位 initStore、表单、action group、list/tabs 绑定。
- 保持 store key 一致。
- 不要把 reload 绑到高频字段更新。
- 优先复用现有 action 链。
- 修改后用 jq 校验 JSON，并用页面真实操作验证打开、查询、保存、删除、刷新。
## Vibe Coding 上下文压缩

保留这些信息：

- 用户真实诉求：
- 已经确认的入口：
- 已经修改的文件：
- 已经通过的验证：
- 已知失败和原因：

丢弃这些噪音：

- 无关日志全文。
- 重复的命令输出。
- 已排除的猜测。

恢复工作时：

1. 先核对最新用户消息。

## 测试：无头浏览器

验证要求：

1. 打开真实 URL。
2. 等待页面资源加载完成。
3. 按用户路径点击、输入、保存。
4. 监听 console error、pageerror、requestfailed。
5. 保存 JSON 报告和关键截图。

断言内容：

- 页面可打开。
- 目标控件可见。
- 操作结果正确。
- 数据保存后可回读。
- 无前端错误和失败请求。

---

## 实施记录 2026-05-13

### 用户真实诉求

当前 `agent_regression` 页面后端能力可继续使用，主要问题是前端体验不好；目标是重构成接近 Web 版 DeepSeek 的聊天工作台，操作更直观，保留查询、发送、刷新、删除、回读能力。

### 已经确认的入口

- 真实页面：`http://192.168.232.130:8015/collect-ui#/collect-ui/framework/agent_regression`
- 前端低代码配置：`collect/frontend/page_data/data/system/agent_regression.json`
- 页面路由注册：`collect/frontend/page_data/index.yml` 中 `agent_regression`
- 后端服务链：`agent.session_query`、`agent.session_upsert`、`agent.run_create`、`agent.message_query`、`agent.run_query`、`agent.session_delete`

### 功能架构梳理

- Store：继续使用 `agentRegressionStore`，核心 key 保持为 `pageForm`、`sessionInfo`、`runInfo`、`sessionList`、`messageList`、`activeSessionId`、`sendLoading`、`refreshLoading`。
- 表单：主表单为 `agent-regression-form`，负责 `session_key`、`title`、`scene_code`、`model`、`system_prompt`、`input_text`；新增会话查询表单 `agent-session-search-form`，只负责历史会话查询条件。
- Action group：新增 `agent-send-message`、`agent-refresh-current`、`agent-reload-session-list`，发送按钮和 `Ctrl+Enter / Cmd+Enter` 复用同一条发送链，避免两套逻辑漂移。
- List 绑定：历史会话继续绑定 `sessionList` + `agent_session_id`；消息流继续绑定 `messageList` + `agent_message_id`；快捷提示绑定新增 `quickPromptList` + `prompt_id`。
- 低代码边界：后端接口、数据表和 Go 插件不变；本次只重构页面 JSON、原型和说明文档。

### 已经修改的文件

- `collect/frontend/page_data/data/system/agent_regression.json`：重构 UI、增加会话查询、快捷提示、Action group、DeepSeek 风格布局和响应式样式。
- `f/data/project/sport/feature/原型设计/agent/code.html`：输出静态原型 HTML。
- `feature/原型设计/agent/readme.md`：补充页面说明、低代码映射和验证重点。
- `feature/原型设计/agent/prototype-check.json`：静态原型无头浏览器检查报告。
- `/data/project/sport/feature/原型设计/agent/screen.png`：静态原型桌面视口截图。
- `feature/原型设计/agent/backups/agent_regression.before-20260513.json`：修改前 JSON 备份。
- `feature/13-agent.md`：追加本实施记录、架构梳理、测试计划和验收计划。

## 原型我已经调好了
请按照这个标准做
- /data/project/sport/feature/原型设计/agent/screen.png
- /data/project/sport/feature/原型设计/agent/code.html

### 测试计划

- JSON 静态校验：`jq empty collect/frontend/page_data/data/system/agent_regression.json`
- 原型验证：打开 `feature/原型设计/agent/index.html`，检查左侧历史、主聊天区、快捷提示、底部输入在桌面宽度下布局完整。
- 页面冒烟：打开真实 URL，监听 `console error`、`pageerror`、`requestfailed`。
- 交互回归：查询会话、重置查询、点击快捷提示、输入多行消息、点击发送消息、刷新当前对话、删除会话。
- 数据回读：发送后检查 `Session ID`、`Run ID`、用户消息、Agent 回复、历史会话高亮。

### 验收计划

- 页面可打开并能看到 `Agent Chat 回归`、`查询会话`、`高级设置`、`发送消息`。
- `Enter` 能换行，`Ctrl+Enter / Cmd+Enter` 与点击发送按钮走相同发送动作组。
- 发送后能回读消息列表、运行状态和会话列表，当前会话保持高亮。
- 查询会话只刷新左侧历史，不绑定到高频输入更新。
- 删除当前会话后清空当前上下文；删除非当前会话不影响当前消息流。
- 无前端 console error、pageerror、失败请求。

### 已经通过的验证

- `jq empty collect/frontend/page_data/data/system/agent_regression.json` 通过。
- `go test ./...` 通过。
- 静态原型无头浏览器检查通过：桌面视口 `1440x980` 下 `Agent Chat 回归`、`查询会话`、`今天要让 Agent 做什么？`、`快捷开始`、`发送消息` 均可见，且无 console error / pageerror。

### 已知失败和原因

- 暂无。真实页面无头浏览器回归待服务启动后执行。

---

## 实施记录 2026-05-13 23:35

### 本轮架构落地

- 低代码范围：只改 `collect/frontend/page_data/data/system/agent_regression.json`，后端 agent 服务、Go 插件、数据表不变。
- Store：保留兼容 key `pageForm`、`sessionInfo`、`runInfo`、`sessionList`、`messageList`、`activeSessionId`、`sendLoading`、`refreshLoading`；新增 `paneAForm`、`paneBForm`、`sessionInfoA/B`、`messageListA/B`、`runListA/B`、`activePane`、`splitDirection`、`uploadedImagesA/B`。
- 表单：拆成 `agent-pane-a-form`、`agent-pane-b-form`、`agent-session-search-form`，发送链分别读取当前 Pane 表单，避免 A/B 会话互相覆盖。
- 分屏：使用低代码 `panel-group`、`panel`、`panel-resize`，`splitDirection` 支持 `horizontal` 左右分屏和 `vertical` 上下分屏。
- 多 Tab：每个 Pane 使用 `tabs`，A Pane 包含 `Session A` / `Trace Log`，B Pane 包含 `Session B` / `Docs`。
- 图片上传：每个 Pane 的输入区使用 `upload`，调用 `sport.upload_course_attachment`，上传完成后把图片路径写入对应 Pane 的 `uploadedImagesA/B`，发送时附加到 `input_text`。
- 模型切换：每个 Pane 头部保留 `model` 下拉，支持 `gpt-5-mini`、`gpt-5.4-mini`、`gpt-5.4`。
- Session 概要：工作台右上区域和 Pane 头部展示 `context_summary` / `last_response_id` / 当前操作结果。
- 外层框架：页面级 CSS 在存在 `.agent-chat-page` 时隐藏 collect 顶栏和框架二级页签，使真实页面贴近已确认原型。

### 本轮修改文件日志

- `collect/frontend/page_data/data/system/agent_regression.json`：重构为双 Pane、多 Tab、分屏切换、图片上传、模型切换、会话概要和真实发送/刷新/查询链。
- `test/lowcode-page/scripts/frontend/agent_regression_page_check.js`：更新无头浏览器回归脚本，覆盖真实页面打开、左右/上下切换、快捷提示、A Pane 发送、B Pane 输入、会话查询和刷新。
- `feature/原型设计/agent/readme.md`：同步低代码落地状态和验证重点。
- `feature/原型设计/agent/backups/agent_regression.before-implementation-20260513-225846.json`：本轮修改前 JSON 备份。
- `test/lowcode-page/results/latest/agent-regression-validation/agent-regression-result.json`：真实页面回归 JSON 报告。
- `test/lowcode-page/results/latest/agent-regression-validation/agent-regression-page.png`：真实页面关键截图。
- `test/lowcode-page/results/latest/agent-regression-validation/agent-regression-console-errors.log`：浏览器错误日志，当前为空。
- `feature/13-agent.md`：追加本轮架构、改动、测试和验收记录。

### 本轮测试计划

- 静态校验：`jq empty collect/frontend/page_data/data/system/agent_regression.json`。
- 编译回归：`go test ./...`。
- 服务验证：按仓库约定执行 `./linux-shutdown`、端口检查、`./linux-startup`、端口检查、`curl --noproxy '*' http://127.0.0.1:8015/collect-ui/`。
- 页面服务验证：POST `frontend.agent_regression`，确认返回 `success=true`、`tag=layout-fit`、`storeName=agentRegressionStore`。
- 浏览器回归：运行 `node test/lowcode-page/scripts/frontend/agent_regression_page_check.js`，监听 `console error`、`pageerror`、`requestfailed`，保存截图和 JSON 报告。

### 本轮验收计划

- 页面打开后直接进入无外层 collect chrome 的 Agent 工作台。
- 左侧历史会话可查询、可重置、可点击加载到当前激活 Pane。
- A/B 两个 Pane 同屏显示，能在左右/上下分屏之间切换。
- 每个 Pane 有独立 Tab、模型选择、快捷提示、图片上传入口、输入区和发送按钮。
- 发送后能回读用户消息和 Agent 回复；刷新后不产生前端错误。
- 浏览器回归报告中 `consoleErrors=[]`、`pageErrors=[]`、`failedRequests=[]`。

### 本轮已通过验证

- `jq empty collect/frontend/page_data/data/system/agent_regression.json` 通过。
- `go test ./...` 通过。
- 服务已重启：`8015` 端口监听进程 `run-dev-main`；`/collect-ui/` 返回 `200`。
- `frontend.agent_regression` 页面服务返回 `success=true`，`tag=layout-fit`，`storeName=agentRegressionStore`。
- 无头浏览器回归通过：`paneCount=2`、`activePaneCount=1`、`verticalDirection=vertical`、`horizontalDirection=horizontal`、`hasSplitControls=true`。
- 回归脚本输出 `consoleErrors=[]`、`pageErrors=[]`、`failedRequests=[]`。

### 本轮已知失败和原因

- 根路径 `http://127.0.0.1:8015/` 返回 `301`，这是应用根路径跳转行为；实际静态入口 `/collect-ui/` 返回 `200`，不影响本页面验收。
