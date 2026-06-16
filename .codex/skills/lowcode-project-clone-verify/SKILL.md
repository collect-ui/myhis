---
name: lowcode-project-clone-verify
description: 复制一份低代码母项目为新项目（含 SQLite 清库与最小种子），修复登录/首页路由循环风险，并用无头浏览器输出页面验证报告。
---

# Lowcode Project Clone & Verify

## 适用场景
- 用户要从母项目快速复制一个新项目（目录、端口、项目标识隔离）。
- 项目是低代码后端（YAML/JSON 驱动）+ SQLite。
- 需要“干净数据库 + 最小可登录菜单”。
- 需要无头浏览器验证页面可用性与无循环跳转。

## 输入参数（执行前先确定）
- 母项目目录：例如 `/data/project/sport`
- 新项目目录：例如 `/data/project/auto-check`
- 新项目标识：例如 `auto-check`
- 新端口：例如 `8016`
- 管理员账号：默认 `admin / 123456`

## 标准执行流程

### 1) 复制项目
- 复制目录（不带 `.git`）：
  - `rsync -a <source>/ <target>/ --exclude .git`
- 不回写母项目，所有改动仅在新项目目录执行。

### 2) 修改新项目配置
- 修改 `conf/application.properties`：
  - `dataSourceName=./database/<new_db>.db`
  - `server_port=<new_port>`
  - `project=<new_project_code>`
  - `current_project_code=<new_project_code>`
  - 登录模式建议：
    - 需要登录：`must_login=true`
    - 调试免登录：`must_login=false`
  - 建议关闭新项目静态强缓存（避免旧前端缓存导致“打不开”假象）：
    - `dirList` 中 `/collect-ui,./frontend/collect-ui,false`

### 3) 初始化 SQLite（全表结构 + 最小种子）
- 从母库复制**表结构**，不复制业务数据。
- 注入最小数据：
  - `role`：`admin`
  - `user_account`：`admin`（密码 md5）
  - `user_role_id_list`：绑定 admin 用户与 admin 角色
  - `sys_menu` / `role_menu`：最小系统菜单与授权
  - `sys_code`：最小必要码表（至少 menu_type / user_job_status 等）

### 4) 处理登录与首页路由（重点）
- 防止“登录页与首页互跳”：
  - `sys_menu` 必须有 `login` 菜单（`url=/login`，`api=frontend_autodesk.login`）。
  - 推荐策略：
    - 登录后默认首页：业务页（如 `user`）`is_index=1`
    - 未登录默认首页：动态落到 `login`（不要把 DB 中 `is_index` 永久设为 login）
- 在 `collect/system/menu/menu_query.sql` 中处理：
  - `with_role=true` 且未登录时，仅返回 `is_common=1` 菜单
  - 未登录时动态 `is_index`：仅 `login` 为 `1`
  - 建议 SQL 形态：
    - `SELECT` 里对 `is_index` 用 `case when a.menu_code='login' then '1' else '0' end`（仅未登录分支）
    - `where ... with_role` 分支里未登录条件不要写成 `1=1`

### 4.1) `is_common` 约束（实战必需）
- 若 `must_login=true`，建议仅 `login` 为 `is_common=1`，其余业务菜单设为 `0`。
- 否则未登录也会拿到业务菜单，前端可能出现 `menu_query / framework / home` 请求风暴。

### 4.2) `is_index` 约束（实战必需）
- `is_index=1` 必须唯一。
- 推荐：登录后首页菜单（如 `user`）设为 `is_index=1`。
- 未登录首页不要依赖 DB 固定值，使用 `menu_query.sql` 动态将 `login` 设为首页。

### 5) 启动并验证服务
- 启动新项目：
  - `cd <target> && go run main.go`
- 端口检查：
  - `ss -ltnp | rg ':<new_port>'`
- 登录接口检查：
  - `POST /template_data/data` with `{"service":"system.login","username":"admin","password":"123456"}`

### 6) 无头浏览器验证（Playwright）
- 未登录验证：
  - 打开 `/collect-ui`
  - 预期落到 `#/.../login`，且无请求风暴
- 已登录验证：
  - 先调用登录接口，再打开 `/collect-ui`
  - 预期落到业务首页（如 `#/.../framework/user`）
- 页面遍历验证：
  - 从 `system.menu_query` 抽取有 `url` 的菜单
  - 逐页访问，记录：
    - final URL
    - `#root` 渲染
    - `console.error` / `pageerror` / `requestfailed`

## 验收标准
- 新项目端口不冲突，可稳定监听。
- 未登录时进入登录页，不循环。
- 已登录时进入业务首页，不循环。
- 菜单页面可访问，`#root` 渲染正常。
- 前端错误计数可控（目标为 0）。

## 常见问题与修复
- 症状：一直在 `menu_query` / `framework` / `home` 循环  
  - 原因：未登录也拿到受保护首页菜单（`is_common` 配置不当），或 `with_role` SQL 未登录分支写错。  
  - 修复：
    - 未登录仅返回 `is_common=1`
    - `is_common` 仅保留 `login=1`
    - SQL 中未登录分支动态 `is_index=login`

- 症状：登录后仍停在登录页  
  - 原因：DB 中 `is_index=1` 固定在 `login`。  
  - 修复：登录后首页应是业务页（例如 `user`），未登录首页由 SQL 动态覆盖为 `login`。

- 症状：复制后菜单为空或权限错乱  
  - 原因：`project` / `belong_project` 不一致。  
  - 修复：统一 `application.properties` 的 `project` 与菜单数据的 `belong_project`。

## 输出报告模板（建议）
- 新项目路径、端口、数据库文件
- 配置变更摘要
- 最小种子计数（role/user/menu/role_menu/sys_code）
- 未登录验证结果
- 已登录验证结果
- 页面遍历统计（通过/失败、错误明细）
