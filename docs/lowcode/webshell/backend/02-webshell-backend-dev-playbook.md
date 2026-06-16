# Webshell 后台开发经验手册

本手册用于新会话快速接手 `auto-desk` 的 webshell 后台模块开发，覆盖整体架构、模块开发流程、数据库建表与 CRUD、低代码配置实践，以及常见踩坑。

## 1. 项目整体架构（先理解再开发）

- 后端：Go + Gin（入口 `main.go`）
- 核心模式：低代码驱动（`collect/**/*.yml` + `collect/**/*.sql`）+ 少量 Go 扩展（`plugins/`）
- 页面：低代码 JSON（`collect/frontend/page_data/**/*.json`）
- 模型：`model/**`（用于 `model_save` / `model_update`）
- 数据库：SQLite，配置在 `conf/application.properties`
  - `driverName=sqlite3`
  - `dataSourceName=./database/price.db`

理解方式：
1. `service.yml` 负责模块挂载。
2. `index.yml` 负责接口定义（query/add/update/delete）。
3. `*.sql` 负责列表与统计查询。
4. `webshell.json` 负责页面行为与接口调用。

## 2. 新增后台模块标准流程（推荐顺序）

1. 设计表结构（主键、业务字段、`is_delete`、索引、唯一约束）。
2. 创建建表 SQL（建议放 `sql/<table>.sql`）。
3. 新增 model 文件（`model/<domain>/<table>.go`）。
4. 在 `model/<domain>/add_table.go` 注册表与主键映射。
5. 新建 `collect/webshell/<biz>/index.yml` 与对应 SQL 文件。
6. 在 `collect/webshell/service.yml` 挂载新模块路径。
7. 前端 `webshell.json` 用接口替换 mock。
8. 验证：`go test ./...` + 页面手测。

## 3. 新表 model 应该插在哪里（关键）

以 `webshell_workspace_project` 为例：

### 3.1 新增 model 文件

路径示例：`model/devops/webshell_workspace_project.go`

最小结构：
- struct 字段（`gorm:"column:..."` 与表字段一致）
- `TableName() string`
- `PrimaryKey() []string`

### 3.2 注册到 add_table（必须）

路径：`model/devops/add_table.go`

在 `GetTable()` 中新增：

```go
workspaceProject := WebshellWorkspaceProject{}
modelMap["webshell_workspace_project"] = workspaceProject
primaryKeyMap["webshell_workspace_project"] = workspaceProject.PrimaryKey()
```

不注册会导致低代码 `model_save` / `model_update` 找不到表模型映射。

## 4. 数据库创建与 CRUD 模板

## 4.1 建表模板（SQLite）

```sql
CREATE TABLE IF NOT EXISTS webshell_workspace_project (
  webshell_workspace_project_id TEXT PRIMARY KEY,
  project_name TEXT NOT NULL,
  project_code TEXT NOT NULL,
  order_id INTEGER NOT NULL DEFAULT 1,
  show_home TEXT NOT NULL DEFAULT '1',
  server_id TEXT NOT NULL,
  server_os_users_id TEXT NOT NULL,
  project_dir TEXT NOT NULL,
  git_repo_url TEXT NOT NULL,
  is_delete TEXT NOT NULL DEFAULT '0',
  create_time TEXT NOT NULL,
  create_user TEXT,
  modify_time TEXT NOT NULL,
  modify_user TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_webshell_workspace_project_code
ON webshell_workspace_project(project_code);
```

建议：
- 统一假删除字段为 `is_delete`
- 统一时间字段为字符串时间 `create_time/modify_time`
- 唯一键在库层也要建立（不仅前端校验）

## 4.2 低代码 query/count

- `base.sql`：公共 where 条件（必须包含 `is_delete='0'`）
- `*_list.sql`：列表查询（可 join 关联展示字段）
- `*_count.sql`：统计数量

## 4.3 低代码增删改

- 新增：`module: model_save`
- 更新：`module: model_update`
- 删除：用 `model_update` 将 `is_delete` 更新为 `1`

## 5. Webshell 前端对接实践（项目管理）

### 5.1 参数拼接规范（强烈建议）

- 保存/查询优先使用 `appendFormFields`
- 分页参数（`page/size`）不在表单时，用 `appendFields` 额外拼接
- 不建议先把 form 值同步到 store 再 ajax，减少中间态错误

### 5.2 项目键统一规则

工作空间项目主键建议统一使用 `project_code`，以下字段全链路保持一致：

- `workspaceCurrent`
- `workspaceOptions` 的 key 字段
- `workspaceLoadedMap` 的 map key
- `tabs.keyField`
- 子面板 `project_id`

任何一处混用 `value/path/project_code`，都可能出现“切换无效或工作区空白”。

### 5.3 reload 分组边界

- `reload-workspace-project`：drawer 项目切换
- `reload-workspace-project-manage`：项目管理弹窗列表

不要串组调用，否则容易出现数据覆盖和循环刷新。

## 6. 常见问题与排障顺序

1. JSON 是否合法（先过语法）
2. 接口是否被高频触发（查看 action 链）
3. `workspaceOptions` 数据形状与消费端字段是否一致
4. `workspaceCurrent` 与 `tabs.keyField` 是否同一键模型
5. 保存后是否先关闭弹窗再刷新

高频问题：
- 接口疯狂刷新：通常是把 reload 放在高频 form action 中。
- 编辑/删除无效：常见是主键没带上或字段名不一致。
- 切换项目无反应：常见是 `project_code` 与 `value` 混用。

## 7. 验证清单（每次改完）

1. `go test ./...`
2. 打开项目管理，确认仅触发预期查询次数
3. 搜索/重置/分页正确
4. 新增成功并关闭弹窗
5. 编辑成功并可见更新
6. 删除有二次确认，删除后不再显示
7. drawer 项目切换可用，工作空间按项目键加载

## 8. 快速定位目录

- 后台服务入口：`collect/webshell/service.yml`
- 模块接口定义：`collect/webshell/<biz>/index.yml`
- 模块 SQL：`collect/webshell/<biz>/*.sql`
- 页面配置：`collect/frontend/page_data/data/server/webshell.json`
- 模型定义：`model/devops/*.go`
- 表注册：`model/devops/add_table.go`
- 数据库配置：`conf/application.properties`

---

如果是新 session，建议先读本文件，再读：

1. `AGENTS.md`
2. `docs/lowcode/webshell/README.md`
3. `docs/lowcode/webshell/frontend/11-项目管理配置实战.md`
4. `.opencode/skills/lowcode-json/SKILL.md`
