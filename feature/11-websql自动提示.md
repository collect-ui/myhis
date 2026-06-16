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
我需要实现sqlite 和mysql 输入自动提示的功能，比如能自动提示表名称、补全字段mysql 
和正式
# 需要调研
https://github.com/dbeaver/dbeaver，看看里面是怎么实现sql 自动提示
需要将设计思路和文档补全在文档请以资深工程师视角完成本文需求：

1. 先阅读需求文档和相邻实现，定位真实入口。
2. 优先复用现有架构、配置和组件能力。
3. 实现后运行必要的格式化、编译、测试和页面回归。
4. 最后把设计取舍、修改文件、验证结果和未解决风险记录到本文。


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

## 实现记录（2026-05-13）

### Vibe Coding 上下文压缩

- 用户真实诉求：WebSQL 编辑器支持 SQLite/MySQL SQL 自动提示，至少覆盖 SQL 关键字、表名、字段名，字段补全需要识别表别名。
- 已经确认的入口：
  - 后端接口：`webshell.websql_execute` 的 `operation=schema` / `schema_scope=completion`。
  - 前端页面：`collect/frontend/page_data/data/server/websql_pool.json`。
  - Monaco 编辑器组件：`/data/project/sport-ui/src/components/editor.tsx`。
- 已经修改的文件：
  - `plugins/module_websql.go`：SQLite/MySQL schema completion 返回 tables/views/indexes/columns。
  - `plugins/module_websql_test.go`：增加 SQLite completion schema 单测。
  - `/data/project/sport-ui/src/components/editor.tsx`：注册 SQL completion provider，构建 metadata，按 SQL 上下文生成关键字/表/字段建议。
  - `collect/frontend/page_data/data/server/websql_pool.json`：WebSQL 页面加载 completion schema，并把 schema/tree 传给 editor。
  - `collect/frontend/page_data/data/server/websql_config.json`：修正内置 MySQL 默认连接 `id/host`，避免清空本地缓存后退回错误地址。
  - `test/lowcode-page/scripts/frontend/websql_completion_check.js`：真实页面无头浏览器回归脚本。
  - `test/lowcode-page/scripts/frontend/websql_pool_regression_check.js`：补充无缓存默认 MySQL 连接断言。
- 已经通过的验证：
  - `jq empty collect/frontend/page_data/data/server/websql_pool.json`
  - `jq empty collect/frontend/page_data/data/server/websql_config.json`
  - `go test ./plugins/...`
  - `go test ./...`
  - `go vet ./...`
  - `bash /home/zz/.codex/skills/sport-ui-build-deploy/scripts/build_and_deploy.sh`
  - `node test/lowcode-page/scripts/frontend/websql_completion_check.js`
- 已知失败和原因：
  - 首次页面回归中表名补全为空。根因是 `websqlCompletionSchema` 的 `ajax.adapt` 写成 `${data||{}}`，当前低代码 ajax 适配器会把无 `.` 的表达式当字段名处理，最终读取 `res.data["data||{}"]` 得到 `undefined`。已改为 `${data}`，直接写入接口返回的 `res.data.data`。
  - `sport-ui-build-deploy` 自带 `img_design` 冒烟在两个历史图片资源上报 `requestfailed`，WebSQL 目标页回归无失败请求。
    - `GET /files/doc/img/2025-10-19/SHBvq8Nn.png => net::ERR_ABORTED`
    - `GET /files/doc/img/2025-10-19/PYwJWVve.png => net::ERR_ABORTED`
  - `websql_pool_regression_check.js` 目前仍有一个历史失败项 `recentCopy=false`，这轮新增的 `builtinMySQLDefaultHost` 已通过，但整份脚本不能算全绿。

### DBeaver 调研结论

- DBeaver 的 SQL Assist 同时提供数据库对象名、SQL 命令和关键字补全；对象名来自已加载元数据和数据库系统表。
- DBeaver 的补全入口是 SQL content assist processor：根据光标位置构造 completion request、提取当前 query、识别分区/注释/控制命令，然后用后台 job 运行 analyzer，避免在 UI 线程直接做重查询。
- DBeaver 旧 analyzer 的关键点不是“看上一个单词”，而是用 `SQLWordPartDetector` 反向扫描光标前 token，拿到 `prevKeyWord`、`prevDelimiter`、`prevWords`，再在 `SQLCompletionAnalyzer` 中把当前请求归类成 `TABLE`、`JOIN`、`COLUMN` 等 query type。
- DBeaver 旧 analyzer 在最终 proposal 过滤阶段还会维护一份 `allowedKeywords`：例如 `DELETE` 后只放 `FROM`，`INSERT` 后只放 `INTO`，对象引用完成后只放下一批合法 clause，而不是把整包 SQL 关键字全塞回来。
- DBeaver 新版语义 analyzer 仍然保留“只分析光标所在 active query”的原则，不会让前一条 SQL 的表别名污染后一条 SQL 的 completion 上下文。
- DBeaver 的 `SQLQueryCompletionAnalyzer` 会按 item kind 和 score 组装 proposals；这意味着“当前明显在输入关键字”时，保留字候选可以压过同前缀的表字段噪音。
- 本实现借鉴的是“元数据预加载 + 光标上下文判断 + provider 只做本地建议生成”的方式：后端一次返回轻量 schema，前端 Monaco provider 根据光标所在语句和 clause 上下文过滤表与字段，避免每次按键请求后端。

参考：

- https://github.com/dbeaver/dbeaver/wiki/SQL-Assist-and-Auto-Complete
- https://raw.githubusercontent.com/dbeaver/dbeaver/devel/plugins/org.jkiss.dbeaver.ui.editors.sql/src/org/jkiss/dbeaver/ui/editors/sql/syntax/SQLCompletionProcessor.java
- https://raw.githubusercontent.com/dbeaver/dbeaver/devel/plugins/org.jkiss.dbeaver.model.sql/src/org/jkiss/dbeaver/model/sql/completion/SQLCompletionAnalyzer.java
- https://raw.githubusercontent.com/dbeaver/dbeaver/devel/plugins/org.jkiss.dbeaver.model.sql/src/org/jkiss/dbeaver/model/sql/parser/SQLWordPartDetector.java
- https://raw.githubusercontent.com/dbeaver/dbeaver/devel/plugins/org.jkiss.dbeaver.model.sql/src/org/jkiss/dbeaver/model/sql/semantics/completion/SQLQueryCompletionAnalyzer.java

### 高级运维视角问题清单与修复

- 问题 1：`FROM` / `JOIN` / `UPDATE` / `INSERT INTO` 后最需要的是表名，但旧实现会混入字段，`SELECT * FROM ` 甚至直接把字段排到最前。
  - 根因：旧逻辑把“倒数第二个单词”当上一关键词，`SELECT * FROM ` 会被误判成还在 `SELECT` 场景。
  - 修复：改成 DBeaver 风格的活动语句 token 反向扫描，识别最近有效 clause，并把当前 completion 归类成 `table` / `column` / `keyword`。
  - 结果：`fromTableSuggest=true`、`joinTableSuggest=true`，表位只返回表候选。
- 问题 2：`WHERE` / `ON` / `alias.` 后面最需要的是当前语句相关字段，但旧实现会回退成全局表噪音，或者把前一条 SQL 的表别名带进来。
  - 根因：alias 解析和上下文判断直接扫描整个 editor model，没有像 DBeaver 那样限定到 active query。
  - 修复：先按分号切出光标所在 statement，再只用该 statement 做 alias 提取、限定符解析和字段候选计算。
  - 结果：`whereFieldSuggest=true`、`qualifiedFieldSuggest=true`、`multiStatementIsolation=true`。
- 问题 3：注释里本来不该弹 SQL 自动提示，但旧页面仍会弹出 `comment` / `completion` 这类文档词噪音。
  - 根因：即使自定义 provider 返回空，Monaco 默认的 word-based suggestions 还会继续给基于文档文本的补全。
  - 修复：SQL 模式下关闭 `wordBasedSuggestions`、`suggest.showWords`，同时关闭 comments/strings 的 quick suggestions。
  - 结果：`commentSilent=true`。
- 问题 4：运维改写 `INSERT INTO table(...)` 时，括号里最需要字段名，不是再列一次表。
  - 根因：旧逻辑没有识别 `INTO table (` 的列上下文。
  - 修复：在 query type 推断里增加 `INTO + (` / `,` 的列场景，并把 `(`、`,` 纳入 trigger characters。
  - 结果：`insertFieldSuggest=true`。
- 问题 5：输入 `sel|` 这种明显是 SQL 关键字的前缀时，旧实现仍会把同前缀字段排到 `SELECT` 前面，不符合运维快速写脚本的预期。
  - 根因：虽然 provider 已有 `sortText`，但在关键字前缀场景仍把表/字段和关键字一起返回，Monaco 会按前缀匹配把字段噪音带回来。
  - 修复：当当前输入已命中 SQL 关键字前缀时，切到 `keyword_only` 模式，只返回关键字、字面量和函数。
  - 结果：`keywordSuggest=true`，最新报告里 `sel|` 首项已经是 `SELECT`。
- 问题 6：`WHERE a.id = |` 这种值位最需要的是 `NULL/TRUE/FALSE` 一类字面量，旧实现却继续推字段。
  - 根因：旧逻辑只区分“字段位”和“表位”，没有识别比较运算符后的 value context。
  - 修复：在字段上下文里补充 value context 识别，比较符后切到 `keyword_only + literals_first`，优先返回 `NULL/TRUE/FALSE/CURRENT_*`。
  - 结果：`valueLiteralSuggest=true`。
- 问题 7：`CREATE TABLE |` 的名字位不该提示现有表名，这会误导成“选已有对象”，对建表操作反而不好用。
  - 根因：旧逻辑把 `TABLE` 一概当对象引用位处理。
  - 修复：识别 `CREATE TABLE/VIEW` 为新对象命名位，直接静默，不回表对象列表。
  - 结果：`createTableSilent=true`。
- 问题 8：`FROM table alias |`、`JOIN table alias |`、`INSERT |`、`DELETE |` 这些 clause 过渡位虽然已经不是表/字段了，但旧实现仍会把整包关键字甚至字面量一起带出来，运维实际写 SQL 时噪音仍然偏大。
  - 根因：provider 的 `keyword_only` 仍在复用全局关键字/字面量集合，没有像 DBeaver `allowedKeywords` 那样按 clause 白名单收敛。
  - 修复：增加 clause 白名单和 continuation 关键字推断：
    - `FROM table alias |` 只回 `WHERE/JOIN/GROUP BY/ORDER BY/LIMIT/OFFSET`
    - `JOIN table alias |` 只回 `ON`
    - `INSERT |` 只回 `INTO`
    - `DELETE |` 只回 `FROM`
    - `GROUP|` / `ORDER|` 这类半截 clause 只回 `GROUP BY` / `BY`
  - 结果：`postTableKeyword=true` 且首批候选不再出现 `SELECT/FROM`；`joinOnKeyword=true`、`insertIntoKeyword=true`、`deleteFromKeyword=true`。
- 问题 9：`WHERE a.id = 1 |` 这种条件已经写完整的位置，旧实现还会继续推字段，不利于快速补 `AND/OR/ORDER BY`。
  - 根因：旧逻辑只识别“正在填值”与“正在填字段”，没有识别“条件表达式已闭合，下一步应该转向逻辑/排序 clause”。
  - 修复：补充 condition boundary 判断；比较表达式结束并进入空格位后，切到过滤 clause 白名单，只回 `AND/OR/GROUP BY/ORDER BY/LIMIT/OFFSET`。
  - 结果：`whereConditionKeyword=true`，首批候选为 `AND/OR/GROUP BY/ORDER BY/LIMIT`。
- 问题 10：MySQL 自动提示默认看起来像“没配好”，清空本地缓存后会退回 `127.0.0.1` 且内置连接 `id` 与历史缓存不一致，导致之前保存过的 MySQL 连接信息无法稳定回灌到内置连接。
  - 根因：仓库内置 `websql_config.json` 里的 MySQL 默认项仍使用 `builtin-local-mysql` + `127.0.0.1:3306`，而页面缓存和既有回归链路已经在使用 `builtin-collect-ui-mysql` + `202.140.140.117:3306`。
  - 修复：统一内置 MySQL 连接 `id` 为 `builtin-collect-ui-mysql`，并把默认主机改成 `202.140.140.117:3306`，这样无缓存首开、清空缓存重开、以及已保存连接的回灌逻辑都落到同一条真实连接上。
  - 结果：清空 `workspace-websql-connections*` 后，页面仍能直接展示正确的内置 MySQL 连接，后续 MySQL 元数据查询不再默认打到错误的 localhost。
- 问题 11：`INSERT INTO table VALUES (|` 这种 value tuple 场景本应优先提示 `NULL/TRUE/FALSE` 和函数，旧实现却沿用列上下文，容易把字段名刷出来。
  - 根因：`VALUES` 被归在列上下文集合里，但 provider 只识别“比较运算符后的 value context”，没有单独处理 `VALUES (` / `VALUES (...,` 这种插入值位。
  - 修复：在 `VALUES` clause 下额外识别 `(`、`,` 后的位置，直接切到 `keyword_only + literals_first`，且不再混入全局 SQL 关键字，只保留字面量和函数。
  - 结果：`insertValuesLiteralSuggest=true`，`INSERT INTO <table> VALUES (|` 首项回到 `NULL`，不再误推表字段。
- 问题 12：`DELETE FROM table |` 属于删除语句的 clause 过渡位，旧实现却直接复用 `SELECT ... FROM` 的 continuation，仍会给出 `JOIN/GROUP BY` 这类无效噪音。
  - 根因：`FROM` 的白名单没有区分它是出现在 `SELECT` 还是 `DELETE` 语句里，导致删除语句沿用了查询语句的后续关键字集合。
  - 修复：`FROM` 过渡位按前导动词分流；当上一关键词是 `DELETE` 时，单独收敛为 `WHERE/ORDER BY/LIMIT`。
  - 结果：`deletePostTableKeyword=true`，`DELETE FROM <table> |` 首项回到 `WHERE`，不再出现 `JOIN/GROUP BY`。
- 问题 13：`UPDATE table SET col = 'x' |` 在赋值表达式已经结束后，旧实现还会回到整包 SQL 关键字集合，继续冒出 `SELECT/FROM/JOIN` 之类的噪音。
  - 根因：`SET` 被视为普通列上下文，但“赋值已闭合”后的 boundary 场景没有单独白名单，最终退回了通用 `SQL_COMPLETION_KEYWORDS`。
  - 修复：给 `SET` clause 增加单独 continuation 白名单；当 `SET` 赋值表达式结束并进入空格位后，只回 `WHERE/ORDER BY/LIMIT`。
  - 结果：`updateSetKeyword=true`，`UPDATE <table> SET completion_note = 'done' |` 首项回到 `WHERE`，不再出现查询类关键字噪音。

### 关键实现取舍

- 后端 completion scope 复用现有 WebSQL 连接参数、驱动分支和 schema 读取逻辑，不新增独立服务，保证 SQLite/MySQL 与结构树使用同一权限和连接配置。
- completion schema 返回表/视图及字段，前端同时兼容 `tables/views/schemas/items/object/children` 多种低代码树形结构，避免强绑定单一接口形态。
- Monaco provider 使用每个 model URI 对应一份 metadata，避免多编辑器共享错误连接的表结构。
- SQL completion 先切 active statement，再做 clause 推断；这一步直接对齐 DBeaver 的 `extractQueryAtPos + QueryType` 思路，而不是依赖单行或全文件的简单正则猜测。
- 字段补全支持 `FROM table alias` / `JOIN table alias`，在 `d.` 这种限定符后只给对应表字段，并隔离前后两条 SQL 的 alias 污染。
- 当当前输入明显是 SQL 关键字前缀时，不再把表/字段和关键字混排，而是切到关键字专用候选集，确保 `SELECT/WHERE/JOIN` 这类 clause 能稳定排前。
- 对 `FROM/JOIN/INSERT/DELETE/UPDATE` 等 clause 过渡位增加关键字白名单，显式过渡位只回下一批合法关键字，不再混入全局关键字、字面量或函数噪音。
- `FROM` 虽然是同一个 token，但在 `SELECT` 与 `DELETE` 后的合法 continuation 不同；按前导动词拆分后，删除语句的过渡提示不会再继承查询语句的 `JOIN` 噪音。
- `SET` 也不是纯字段上下文；一旦赋值表达式已经闭合，就应该从“字段/值输入”切回“语句下一步”白名单，而不是继续复用全局关键字集。
- 比较运算符后的 value context 优先返回 SQL 字面量；`CREATE TABLE` 的命名位则保持静默，避免把“新对象命名”和“已有对象引用”混为一谈。
- `VALUES (...)` 这种插入值位按 value context 单独处理，不复用普通列上下文，避免在纯字面量输入阶段被字段候选淹没。
- 条件表达式完整结束后，从字段补全切换到逻辑/排序 clause 补全，避免 `WHERE a.id = 1 |` 继续刷字段列表。
- 保留关键字和常用函数建议；表/字段建议使用 SQL 方言决定是否需要反引号或双引号转义。
- SQL 模式关闭 Monaco 默认词典补全，只保留 schema 驱动的上下文补全，避免注释和字符串里的无意义建议。
- 对低代码 `${storeField}` 形式的 schema/tree 增加 MobX reaction 同步，避免 schema 异步写入 store 后 Monaco 全局 provider 仍持有旧 metadata。
- 内置连接配置也属于自动提示链路的一部分：如果默认 MySQL 主机或连接 `id` 错了，completion 实现即使正确，用户首开的实际体验仍然会退化成“查不到真实元数据”或“缓存配置不生效”。

### 验证证据

- 无头浏览器目标 URL：`http://127.0.0.1:8015/collect-ui#/collect-ui/framework/websql-pool`
- 验证脚本会创建临时表 `websql_completion_demo_<timestamp>` 和 `websql_completion_side_<timestamp>`，断言：
  - `sel|` 出现 `SELECT`
  - `sel|` 的首项就是 `SELECT`
  - `SELECT * FROM websql_completion_demo_|` 只出现表候选
  - `SELECT * FROM <table> d JOIN websql_completion_side_|` 只出现表候选
  - `SELECT * FROM <table> d |` 首批只出现后续 clause 关键字，不再出现 `SELECT/FROM`
  - `SELECT * FROM <table> d JOIN <otherTable> s |` 首项是 `ON`
  - `SELECT * FROM <table> d WHERE com|` 出现 `completion_note`
  - `SELECT d.com| FROM <table> d` 只出现限定表字段
  - `INSERT INTO <table>(|` 出现字段名
  - `SELECT * FROM <table> d WHERE d.id = |` 首批候选是 `NULL/TRUE/FALSE`
  - `INSERT INTO <table> VALUES (|` 首批候选是 `NULL/TRUE/FALSE`
  - `UPDATE <table> SET completion_note = 'done' |` 首批候选是 `WHERE/ORDER BY/LIMIT`
  - `SELECT * FROM <table> d WHERE d.id = 1 |` 首批候选是 `AND/OR/GROUP BY/ORDER BY/LIMIT`
  - `GROUP|` 只回 `GROUP BY`
  - `INSERT |` 只回 `INTO`
  - `DELETE |` 只回 `FROM`
  - `DELETE FROM <table> |` 首批候选是 `WHERE/ORDER BY/LIMIT`
  - 前一条 SQL 引入的 `other_flag` 不会污染后一条 SQL 的 `WHERE |`
  - `-- completion comment |` 不出现补全
  - `CREATE TABLE |` 不出现补全
- `websql_pool_regression_check.js` 额外断言：清空 `workspace-websql-connections*` 本地缓存后，连接列表仍存在内置 MySQL，且展示 `202.140.140.117:3306`
- 最新报告：`test/lowcode-page/results/latest/http-proxy-validation/websql-completion-check.json`
- 最新截图：`test/lowcode-page/results/latest/http-proxy-validation/websql-completion-check.png`
- 最新结果：`ok=true`，`keywordSuggest/fromTableSuggest/joinTableSuggest/postTableKeyword/joinOnKeyword/whereFieldSuggest/qualifiedFieldSuggest/insertFieldSuggest/valueLiteralSuggest/insertValuesLiteralSuggest/updateSetKeyword/whereConditionKeyword/groupByKeyword/insertIntoKeyword/deleteFromKeyword/deletePostTableKeyword/multiStatementIsolation/commentSilent/createTableSilent=true`，`consoleErrors/pageErrors/requestFailed` 均为空。
- 补充证据：
  - `insertValuesLiteral` 实际候选为 `NULL/TRUE/FALSE/CURRENT_DATE/CURRENT_TIME/CURRENT_TIMESTAMP/COUNT/SUM/AVG/MIN/MAX/COALESCE/IFNULL`
  - `updateSetKeyword` 实际候选为 `WHERE/ORDER BY/LIMIT`
  - `websql_pool_regression_check.json` 中 `builtinMySQLDefaultHost=true`，`builtinMySQLConnection="MySQL202.140.140.117:3306"`

### 未解决风险

- MySQL 回归依赖本地可用 MySQL 连接；当前自动化以 SQLite 为稳定验收路径，MySQL schema 分支通过同一 metadata/provider 结构覆盖，但还需要有真实 MySQL 环境时补一条端到端脚本。
- 大库表很多时 completion schema 可能较大；当前策略是页面加载/刷新连接时拉取，后续可按库/表数量增加分页、缓存失效或按前缀懒加载。
