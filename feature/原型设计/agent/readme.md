# Agent 测试回归 PC 原型说明

## 目标

第四版原型已经作为低代码页面落地。真实页面按最新要求移除 collect 顶部导航、二级页签等外层框架，只展示 `agent_regression` 页面自身内容，并保持接近 Web 版 DeepSeek 的双 Pane 工作台体验。

## 页面结构

- 外层框架：不展示 collect 主导航、系统管理菜单和 `Agent测试回归` 页签。
- 左侧：会话历史改成单行列表，每行展示标题、会话 key、最近响应和状态，减少卡片堆叠占用。
- 左侧折叠：历史区支持收起，收起后只保留窄条入口，点击可展开。
- 历史行 hover：鼠标经过会话行时显示 `编辑`、`删除` 操作提示，默认只展示状态。
- 最近消息：不展示 response id，改成用户可读的最近消息摘要。
- 右侧上部：只保留当前会话、Session ID、模型切换、刷新和 run 结果提示。
- 模型切换：在当前会话栏右侧显式展示 `gpt-5-mini`、`gpt-5.4-mini`、`gpt-5.4`，当前模型高亮。
- 消息区：消息按时间从上到下连续展示，用户和 Agent 仍保留角色区分，但不做复杂装饰。
- 底部：保留多行输入框、发送按钮和三个快捷 chip。
- 移除项：高级设置表单不在精简原型主界面展示，避免页面过重；如需配置，后续可以作为折叠抽屉或弹窗。

## 低代码实现映射草案

- 页面入口：`/collect-ui#/collect-ui/framework/agent_regression`。
- 页面 JSON：`collect/frontend/page_data/data/system/agent_regression.json`。
- Store：继续使用 `agentRegressionStore`，保留 `pageForm`、`sessionInfo`、`runInfo`、`sessionList`、`messageList`、`activeSessionId`；新增 `paneAForm`、`paneBForm`、`sessionInfoA/B`、`messageListA/B`、`runListA/B`、`activePane`、`splitDirection`、`uploadedImagesA/B`。
- Action：A/B Pane 分别使用独立表单和发送链；`Ctrl+Enter / Cmd+Enter` 根据 `activePane` 发送当前 Pane。
- 分屏：使用 `panel-group`、`panel`、`panel-resize`，支持左右和上下切换。
- 多 Tab：A Pane 为 `Session A` / `Trace Log`，B Pane 为 `Session B` / `Docs`。
- 图片上传：每个 Pane 输入区使用 `upload` 调用 `sport.upload_course_attachment`，上传路径随发送内容附加。
- 后端服务：继续使用 `agent.session_query`、`agent.session_upsert`、`agent.run_create`、`agent.message_query`、`agent.run_query`、`agent.session_delete`。

## 落地验证

- JSON 校验：`jq empty collect/frontend/page_data/data/system/agent_regression.json` 通过。
- Go 回归：`go test ./...` 通过。
- 页面服务：`frontend.agent_regression` 返回 `success=true`、`tag=layout-fit`、`storeName=agentRegressionStore`。
- 浏览器回归：`node test/lowcode-page/scripts/frontend/agent_regression_page_check.js` 通过。
- 证据文件：
  - `test/lowcode-page/results/latest/agent-regression-validation/agent-regression-result.json`
  - `test/lowcode-page/results/latest/agent-regression-validation/agent-regression-page.png`
  - `test/lowcode-page/results/latest/agent-regression-validation/agent-regression-console-errors.log`

## 验证重点

- 打开页面后左侧、主区、底部输入区都可见。
- 会话历史是单行列表，不再使用大卡片。
- 会话历史支持折叠/展开。
- 会话行 hover 后出现编辑和删除提示。
- 最近消息不直接显示 response id。
- 消息按时间从上到下展示。
- 模型切换在主操作区可见。
- 发送按钮和 `Ctrl+Enter` 行为保持一致。
- 页面只保留高频操作，减少高级表单常驻占用。


