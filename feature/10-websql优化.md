
# 研发角色定义
请仔细阅读需求项，每一项都读懂你是一个务实的资深研发。
- 先让现有系统低代码结构教你怎么改。
- 保持改动小而完整。
- 不回滚用户已有变更。
- 能用配置表达的行为优先用配置。
- 必须说明每个关键实现选择和验证证据。
- 业务理解有用文件，代码，回归文件，必须落地到本文
## 上下文压缩

保留这些信息：


- 已经确认的入口：
- 已经修改的文件：
- 已经通过的验证：
- 已知失败和原因：

丢弃这些噪音：

- 无关日志全文。
- 重复的命令输出。
- 已排除的猜测。


## 待办需求/优化/bug
- sql 模块目前是直接commit 的模式，对于生产环境不安全，需要增加有是否直接commit ，还是手动commit 模式 ，
- 如果要commit 新增修改、删除，生成一个事件事件再提交，注意事件要存储下来，能排查，2是如果页面刷新丢失了，需要有个超时的定时任务去掉，避免锁库
- 增加个sql 收藏夹功能，执行sql 复制收藏起来
- 目前执行更新sql 这种模式不好看，就一个文字，请搞点图标执行成功，影响多少行，高亮
- http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell 
在webshell 中新建数据库连接，没有办法选择mysql 类型 
- 现在sql 查询，出来的数据，要实现游标的功能，展示50条记录，点下一页，下一页看，支持滚动加载
- http://192.168.232.130:8015/collect-ui#/collect-ui/framework/websql-pool 里面有个快捷引入
我需要支持复制功能
- 现在sql 编辑器 是白色系，但是editor 编辑器是深色系，如果http://192.168.232.130:8015/collect-ui#/collect-ui/framework/websql-pool 从这个进入
是白色的，但是从http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool 从这里进有变成深色，如果新增一个sql 面板，全部又变成了白色，包括editor 这个请修复

你是严格的回归测试负责人。

- 先列核心路径、边界路径和历史易回归点。
- 每个测试必须有明确前置条件、操作步骤、预期结果和证据文件。
- 发现问题时记录最小复现路径和修复后复## 测试：冒烟路径


检查项：

1. 打开入口页面。http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell-editor-pool
2. 不要登录，免登录


失败处理：

- 保存截图。

- 记录最小复现步骤。验结果。

测试要求：

- 使用无头浏览器打开目标页面，按用户真实路径完成操作。
- 记录 console error、pageerror、requestfailed。
- 保存 JSON 报告和关键截图。
- 失败时先根据截图和 DOM 证据修复，再重复验证直到通过。

# 测试回归要求
- 用无头浏览器，回归测试，测试每个待办需求/优化/bug，必须一一验证通过
- 需求所有功能必须全部验证通过
- 反复压力测试验证，验证、修复、测试、验证修复
- 验证和日志必须写到本文，以便后续排查，和接着后开发
- 包括微调的需求记录，写要写在里面，改动文件记录也要记录

## 2026-05-13 继续记录：WebSQL 分页/回滚回归收敛

### 核心路径

- 入口一：`http://127.0.0.1:8015/collect-ui#/collect-ui/framework/websql-pool`
- 入口二：`http://127.0.0.1:8015/collect-ui#/collect-ui/framework/webshell-editor-pool`
- API：`/template_data/data?service=webshell.websql_execute`
- 页面配置：`collect/frontend/page_data/data/server/websql_pool.json`
- 回归脚本：`test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`

### 本轮失败和原因

- 已复现：分页按钮请求已发出，但低代码 action 的 `data` 阶段不能稳定读取 `row.info`，导致 `cursor_offset` 仍为 `0`。
- 已复现：手动事务 UI 显示“待提交”，点击“回滚”请求也发出，但请求体 `event_id` 为空，服务端返回 `event_id 不能为空`。
- 根因判断：嵌套 tabs 的按钮 action 执行时，`row` 和全局 `websqlResultInfo` 都可能不是当前结果的稳定来源；`websqlRecentExecutions` 是结果 tabs 的 `itemData` 来源，当前激活结果应优先从这里读取。

### 本轮改动文件

- `collect/frontend/page_data/data/server/websql_pool.json`
  - 分页请求的 `sql`、`page_size`、`cursor_offset` 改为优先读取当前激活的 `websqlRecentExecutions` 结果项，再回退到 `websqlResultInfo` 和 `row.info`。
  - 提交/回滚请求的 `event_id` 改为优先读取当前激活结果项的 `info.event_id`，避免 action 阶段 `row.info` 丢失。
  - 去掉回滚按钮二次确认，保留提交确认；回滚是解除手动事务锁的安全动作，减少嵌套 Popconfirm 在 tabs 场景下的选择器和触发不稳定。
- `test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`
  - 增加按 `operation` 过滤 `webshell.websql_execute` 响应。
  - 记录回滚请求体 `rollbackRequest`，并断言 `event_id` 非空。
  - 覆盖 API 游标、API 手动事务回滚、直接提交事件、独立 WebSQL 入口、webshell 内嵌入口、MySQL 选项、分页下一页/加载更多、收藏复制、最近执行复制、手动事务 UI 回滚。

### 验证证据

- `jq empty collect/frontend/page_data/data/server/websql_pool.json`：通过。
- `node --check test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`：通过。
- `go test ./plugins -count=1`：通过，`ok moon/plugins 0.015s`。
- 服务重启：
  - `./linux-shutdown`：已停止旧进程。
  - `ss -ltnp | rg ':8015' || true`：启动前为空。
  - `./linux-startup`：已启动，pid `112644`。
  - `ss -ltnp | rg ':8015'`：`run-dev-main` 正在监听 `8015`。
- 无头浏览器完整回归：
  - 命令：`node test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`
  - 结果：`pass: true`
  - JSON 报告：`test/lowcode-page/results/latest/http-proxy-validation/websql-pool-regression-check.json`
  - 截图：
    - `test/lowcode-page/results/latest/http-proxy-validation/websql-pool-regression-standalone.png`
    - `test/lowcode-page/results/latest/http-proxy-validation/websql-pool-regression-webshell.png`
    - `test/lowcode-page/results/latest/http-proxy-validation/websql-pool-regression-final.png`
  - 关键断言：
    - `loadMore.cursor_offset = 50`
    - `nextPage.cursor_offset = 50`
    - `rollbackRequest.event_id = websql_event_d13b63fa-ab77-40b1-ae4b-45019c90a13b`
    - `rollback.commit_status = rolled_back`
    - `consoleErrors = []`
    - `pageErrors = []`
    - `failedRequests = []`

### 回归清单

- 核心路径：打开 webshell-editor-pool，免登录进入 WebSQL 面板，执行查询，下一页/加载更多，收藏和复制，手动事务回滚。
- 边界路径：独立 websql-pool 入口和 webshell 内嵌入口都检查深色编辑器；新增 SQL tab 后仍保持深色。
- 历史易回归点：嵌套 tabs 的 `row.info` 在按钮 action 阶段不稳定，分页游标和事务 `event_id` 均通过当前激活结果项读取。

## 2026-05-13 继续记录：SELECT 0 行结果误显示执行卡片

### 现象

- 用户执行 `SELECT * FROM attachment LIMIT 100;` 后，页面显示“执行成功 / rows affected 0 / last insert id 0 / event -”，看起来像查询结果被清空。
- 后端查询接口对空结果集会正确返回 `statement_type=select`、`row_count=0`、`columns` 非空、`rows=[]`；问题在前端把 `rows.length<=0` 的结果统一当成执行信息卡片。

### 修复

- `collect/frontend/page_data/data/server/websql_pool.json`
  - 结果表格继续只在 `rows.length>0` 时显示。
  - 新增查询空态：`select/show/desc/describe/explain/with/pragma` 或返回 `columns` 的结果，如果 `rows.length<=0`，显示 `查询结果为空`。
  - 执行信息卡片只用于非查询语句，避免 SELECT 空结果再显示 `rows affected`。
  - 顶部状态从 `执行完成 · 0 rows` 调整为 `查询完成 · 0 行`。
- `test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`
  - 增加 `SELECT 1 AS n WHERE 1=0;` 回归路径。
  - 断言后端返回 `row_count=0` 且 `columns` 非空。
  - 断言 UI 显示查询结果区域/空态，并且不显示 `.websql-exec-info`。

### 验证证据

- `jq empty collect/frontend/page_data/data/server/websql_pool.json`：通过。
- `node --check test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`：通过。
- `go test ./plugins -count=1`：通过，`ok moon/plugins 0.015s`。
- 服务重启：
  - `./linux-shutdown`
  - `./linux-startup`，pid `116740`
  - `ss -ltnp | rg ':8015'`：`run-dev-main` 正在监听 `8015`
- 无头浏览器完整回归：
  - 命令：`node test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`
  - 结果：`pass: true`
  - 报告：`test/lowcode-page/results/latest/http-proxy-validation/websql-pool-regression-check.json`
  - 新增关键证据：
    - `emptyQuery.row_count = 0`
    - `emptyQuery.column_count = 1`
    - `emptyQuery.statement_type = select`
    - `emptyQueryRender.emptyVisible = true`
    - `emptyQueryRender.execInfoVisible = false`
    - 页面文本包含 `查询完成 · 0 行` 和 `查询结果为空`

## 2026-05-13 继续记录：分页 loading 与 SQL 收藏目录化

### 需求

- 上一页、下一页切换时要有 loading 状态。
- SQL 收藏要按目录树管理，像文件夹结构一样组织。
- SQL 收藏时必须能定义名称。

### 修复

- `collect/frontend/page_data/data/server/websql_pool.json`
  - 新增 `websqlPagingAction`，上一页/下一页 ajax 使用 `start/end` 写入该状态。
  - 上一页按钮 loading 条件：`websqlPagingAction==='prev'`；下一页按钮 loading 条件：`websqlPagingAction==='next'`。
  - 新增 `收藏 SQL` 弹窗，表单字段：`名称`、`目录`、`SQL`。
  - 收藏数据结构升级为兼容旧数据的本地存储数组：SQL 项为 `type=sql`，目录项为 `type=folder`。
  - `快捷引入` 的收藏区从平铺列表改为目录树，支持目录路径如 `回归/分页`。
  - 收藏树支持选中 SQL 后查看 SQL、引入、复制、删除；选中目录后可删除目录及其子项。
  - 新增 `新建目录` 弹窗，可直接创建空目录。
- `test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`
  - 收藏回归改为真实用户路径：打开收藏弹窗，填写名称 `递归数字查询`，填写目录 `回归/分页`，保存。
  - 打开快捷引入，验证收藏树出现 `回归`、`分页`、`递归数字查询`。
  - 选中收藏 SQL 后验证复制、引入；最近执行复制仍保留验证。
  - 修复 AntD 按钮文案被渲染为 `确 定` 时的测试选择器。

### 验证证据

- `jq empty collect/frontend/page_data/data/server/websql_pool.json`：通过。
- `node --check test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`：通过。
- `go test ./plugins -count=1`：通过，`ok moon/plugins 0.015s`。
- `go test ./...`：通过。
- 服务重启：
  - `./linux-shutdown`
  - `./linux-startup`，pid `119145`
  - `ss -ltnp | rg ':8015'`：`run-dev-main` 正在监听 `8015`
- 无头浏览器完整回归：
  - 命令：`node test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`
  - 结果：`pass: true`
  - 报告：`test/lowcode-page/results/latest/http-proxy-validation/websql-pool-regression-check.json`
  - 关键断言：
    - `queryNextPage = true`
    - `queryLoadMore = true`
    - `favoriteSaved = true`
    - `favoriteNamed = true`
    - `favoriteTree = true`
    - `favoriteCopy = true`
    - `recentCopy = true`
    - `manualRollbackUi = true`
    - `consoleErrors = []`
    - `pageErrors = []`
    - `failedRequests = []`
	  - pending 事务检查：`pending_count = 0`

## 2026-05-13 继续记录：SQL 收藏目录增删改查与弹窗层级

### 现象

- 用户在 `http://192.168.232.130:8015/collect-ui#/collect-ui/framework/websql-pool` 点击 `快捷引入 -> 新建目录` 后，目录弹窗打开但被 `快捷引入` 弹窗遮住，看起来像没有弹框。
- 同时收藏目录只支持新增/删除，缺少目录重命名和 SQL 收藏编辑路径，无法完整做目录和 SQL 收藏的增删改查。

### 修复

- `collect/frontend/page_data/data/server/websql_pool.json`
  - `收藏 SQL`、`新建/编辑收藏目录` 弹窗增加更高 `zIndex`，确保从 `快捷引入` 内打开时位于最上层。
  - 新增 `websqlFavoriteEditingId`、`websqlFavoriteFolderEditingPath`，区分新建和编辑。
  - 目录保存支持编辑模式：重命名目录时同步更新子目录路径和目录下 SQL 收藏的 `folder`。
  - 收藏详情区新增 `编辑` 按钮：选中目录时打开目录编辑弹窗，选中 SQL 收藏时打开 `编辑收藏 SQL` 弹窗。
  - 删除保持本地存储落盘：目录删除会删除该目录、子目录以及目录下 SQL 收藏；SQL 删除只删除当前收藏。
- `test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`
  - 增加无头浏览器收藏 CRUD 回归：新建目录、改名目录、删除目录、编辑 SQL 收藏名称/目录/内容、删除 SQL 收藏。
  - 增加弹窗可操作断言，覆盖目录弹窗从快捷引入内打开后能真实填表并提交。
  - 删除收藏后显式关闭快捷引入，再继续后续手动事务回滚验证。

### 验证证据

- `jq empty collect/frontend/page_data/data/server/websql_pool.json`：通过。
- `node --check test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`：通过。
- 服务重启：
  - `./linux-shutdown`
  - `ss -ltnp | rg ':8015' || true`：启动前为空。
  - `./linux-startup`，pid `128234`。
  - `ss -ltnp | rg ':8015'`：`run-dev-main` 正在监听 `8015`。
- 无头浏览器完整回归：
  - 命令：`node test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`
  - 结果：`pass: true`
  - 报告：`test/lowcode-page/results/latest/http-proxy-validation/websql-pool-regression-check.json`
  - 关键断言：
    - `favoriteFolderDialogTop = true`
    - `favoriteFolderCreate = true`
    - `favoriteFolderUpdate = true`
    - `favoriteFolderDelete = true`
    - `favoriteEdit = true`
    - `favoriteDelete = true`
    - `favoriteCopy = true`
    - `recentCopy = true`
    - `manualRollbackUi = true`
    - `consoleErrors = []`
    - `pageErrors = []`
    - `failedRequests = []`
- Go 验证：
  - `go test ./plugins -count=1`：通过，`ok moon/plugins 0.015s`。
  - `go test ./...`：通过。
- pending 事务检查：
  - `pending_count = 0`

### 2026-05-13 补充验证：已有 test 目录下新增 test2/test3

- `collect/frontend/page_data/data/server/websql_pool.json`
  - 新建目录时如果当前选中目录，会预填当前目录并追加 `/`，例如选中 `test` 后表单预填 `test/`，便于继续输入 `test2/test3`。
- `test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`
  - 新增回归路径：先创建 `test` 目录，再选中 `test` 新建 `test/test2/test3`。
  - 断言新建子目录前输入框预填为 `test/`。
  - 断言树中出现 `test2`、`test3`，再删除 `test`，验证子目录一并删除。
- 验证证据：
  - `jq empty collect/frontend/page_data/data/server/websql_pool.json`：通过。
  - `node --check test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`：通过。
  - `node test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`：`pass: true`。
  - 报告关键断言：`favoriteExistingFolderNestedCreate = true`、`favoriteExistingFolderNestedDelete = true`。
  - `go test ./plugins -count=1`：通过。
  - `go test ./...`：通过。
  - pending 事务检查：`pending_count = 0`。

## 2026-05-13 继续记录：SQL 收藏目录内联编辑体验调整

### 现象

- 上一版从 `快捷引入` 中维护目录时，会关闭/打开不同弹窗，新增完成后父弹窗重新出现，视觉跳动明显。
- 用户要求继续自测多层目录的增删改查，避免“弹出一个框、关闭一个框”的体验。

### 修复

- `collect/frontend/page_data/data/server/websql_pool.json`
  - 新增 `websqlFavoriteEditMode`，在 `快捷引入` 右侧详情区内切换查看态/编辑态。
  - `新建目录` 不再打开 `新建收藏目录` Modal，改为右侧内联目录表单。
  - 选中目录点 `编辑` 不再打开二级 Modal，改为右侧内联目录表单。
  - 选中 SQL 收藏点 `编辑` 不再打开 `编辑收藏 SQL` Modal，改为右侧内联 SQL 表单。
  - 目录树 `openLevel` 调整为 `5`，多层目录默认能展开到更深层。
  - 多层目录重命名继续级联更新子目录和目录下 SQL 收藏路径。
- `test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`
  - 收藏目录回归改为多层路径：`回归/多层/一级/二级`。
  - 新增空目录多层 CRUD：`回归/多层/空目录/子目录` -> `回归/多层/空目录改/子目录改` -> 删除。
  - 新增目录级联回归：把含 SQL 的 `二级` 重命名为 `回归/多层/一级改/二级改`，断言 SQL 收藏详情目录同步变化。
  - 新增体验断言：新增目录和编辑 SQL 收藏时不出现嵌套 Modal。

### 验证证据

- `jq empty collect/frontend/page_data/data/server/websql_pool.json`：通过。
- `node --check test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`：通过。
- 服务重启：
  - `./linux-shutdown`
  - `ss -ltnp | rg ':8015' || true`：启动前为空。
  - `./linux-startup`，pid `132169`。
  - `ss -ltnp | rg ':8015'`：`run-dev-main` 正在监听 `8015`。
- 无头浏览器完整回归：
  - 命令：`node test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`
  - 结果：`pass: true`
  - 报告：`test/lowcode-page/results/latest/http-proxy-validation/websql-pool-regression-check.json`
  - 关键断言：
    - `favoriteMultilevelTree = true`
    - `favoriteNoNestedModal = true`
    - `favoriteInlineEdit = true`
    - `favoriteFolderCreate = true`
    - `favoriteFolderUpdate = true`
    - `favoriteFolderDelete = true`
    - `favoriteFolderCascadeUpdate = true`
    - `favoriteEdit = true`
    - `favoriteDelete = true`
    - `manualRollbackUi = true`
    - `consoleErrors = []`
    - `pageErrors = []`
    - `failedRequests = []`
- Go 验证：
  - `go test ./plugins -count=1`：通过，`ok moon/plugins 0.015s`。
  - `go test ./...`：通过。
- pending 事务检查：
  - `pending_count = 0`

