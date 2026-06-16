你是一个务实的资深研发。

- 先让现有低代码系统结构教你怎么改。
- 保持改动小而完整。
- 不回滚用户已有变更。
- 能用配置表达的行为优先用配置。
- 必须说明每个关键实现选择和验证证据。
请以资深工程师视角完成本文需求：

1. 先阅读需求文档和相邻实现，定位真实入口。
2. 优先复用现有架构、配置和组件能力。
3. 实现后运行必要的格式化、编译、测试和页面回归。
4. 最后把设计取舍、修改文件、验证结果和未解决风险记录到本文。

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
3. 只继续未完成的任务。


## 需求
http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool 目标地址
- 选中sql 执行
- 快捷键ctrl+alt+enter 执行
和正式
# 需要调研
https://github.com/dbeaver/dbeaver，看看里面是怎么实现sql选中执行

## 测试要求：

- 使用无头浏览器打开目标页面，输入sql 关键字、用表元数据关键字能自动提示正常提示。
- 记录 console error、pageerror、requestfailed。
- 保存 JSON 报告和关键截图。
- 失败时先根据截图和 DOM 证据修复，再重复验证直到通过。


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

## 本次实现记录

### 用户真实诉求

- 在 `webshell-editor-pool` 里的 WebSQL 面板支持“选中 SQL 优先执行”。
- 支持 `Ctrl+Alt+Enter` 触发同一条执行链。
- 保持 SQL 关键字补全和表/字段元数据补全可用。

### 已确认的入口

- 低代码页面入口：`collect/frontend/page_data/data/server/websql_pool.json`
- WebSQL 使用的 Monaco 编辑器实现：`/data/project/sport-ui/src/components/editor.tsx`
- 页面运行目标：`http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool`

### 关键实现选择

1. 没改后端 `webshell.websql_execute` 协议，只在前端把真正要执行的 SQL 片段传给现有接口，保持影响面最小。
2. 没为 WebSQL 单独造一个编辑器，而是在通用 Monaco 封装里补了两种能力：
   - `onSelectionChangeAction`：把当前选区同步到 low-code store
   - `shortcutAction`：允许 low-code 页面声明快捷键动作
3. WebSQL 页面对“执行”按钮做了最小改造：
   - 优先读取 `websqlSelectedTextTrimmed`
   - 没有选区时退回 `websqlSqlText`
   - 快捷键动作直接点击同一个执行按钮，避免复制整条执行链
4. 调研 DBeaver 后，沿用了它的核心策略：有选区时执行选中文本，没有选区时执行当前 SQL。参考：
   - https://github.com/dbeaver/dbeaver/wiki/Shortcuts/18de75738b1247f5af9f176b08ac9521c4dd1e78
   - https://github.com/dbeaver/dbeaver/wiki/SQL-Execution

### 已经修改的文件

- `collect/frontend/page_data/data/server/websql_pool.json`
- `/data/project/sport-ui/src/components/editor.tsx`
- `test/lowcode-page/scripts/frontend/websql_selected_execution_check.js`
- `.omx/context/websql-selected-execution-20260513T092858Z.md`
- `.omx/plans/prd-websql-selected-execution.md`
- `.omx/plans/test-spec-websql-selected-execution.md`

### 已通过的验证

- `node -e "JSON.parse(...websql_pool.json...)"`：通过
- `go test ./plugins/...`：通过
- `go vet ./plugins/...`：通过
- `NODE_OPTIONS=--max_old_space_size=9216 npx vite build` in `/data/project/sport-ui`：通过
- 真实 URL 无头回归：`node test/lowcode-page/scripts/frontend/websql_selected_execution_check.js`
  - 按钮执行时，请求里的 `sql` 等于选中文本
  - `Ctrl+Alt+Enter` 时，请求里的 `sql` 等于选中文本
  - SQL 关键字补全可见：`SELECT`
  - 表元数据补全可见：目标表名、`selected_flag`
  - `console error` / `pageerror` / `requestfailed` 都为 0
  - 报告：`test/lowcode-page/results/latest/http-proxy-validation/websql-selected-execution-check.json`
  - 截图：`test/lowcode-page/results/latest/http-proxy-validation/websql-selected-execution-check.png`
- Follow-up 回归：输入自定义 SQL 后再次点击 `MySQL` 连接，编辑器内容保持不变，不再被 `SHOW TABLES;` 覆盖
  - 验证脚本：临时 Playwright 直连真实 URL
  - 结果：`customSql = "SELECT 42 AS keep_me;"`，切换后 `current = "SELECT 42 AS keep_me;"`，`preserved = true`

### 已知失败和原因

- `npm run build` in `/data/project/sport-ui` 失败，但失败点是现有 TypeScript 基线问题，不是本次改动引入：
  - `../collect-ui/src/components/render/render-child.tsx`
  - `../collect-ui/src/index.tsx`
  - `../collect-ui/src/store/root.tsx`
  - `../collect-ui/src/utils/isExpression.tsx`
  - `../collect-ui/src/utils/varValue.tsx`
  - `src/dashboard-training.tsx`
- 首轮快捷键回归失败一次，原因是 low-code `method` 表达式里直接用了 `document`，当前表达式运行环境下为 `undefined`；已改成 `window.document` 后复测通过。

### 未解决风险

- 当前 `sport-ui` 的全量 TypeScript 校验仍然不干净，所以这次只能用 `vite build` 和真实页面回归证明运行态可用；如果后续要把 `npm run build` 也纳入强门禁，需要先清理 `collect-ui` 侧已有 TS 错误。
- 当前连接切换逻辑改成“仅覆盖默认模板 SQL，不覆盖自定义 SQL”。这更符合交互预期，但如果后续有人明确想保留“每次切换连接都强制重置为默认 SQL”的旧行为，需要再加一个显式的“重置 SQL”入口，而不是重新回到隐式覆盖。
