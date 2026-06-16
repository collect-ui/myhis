# 低代码 Tooltip 文档迁移设计（可恢复）

## 作用
- 说明低代码文档迁移的目标、约束、步骤与恢复入口，保证迁移任务可中断、可续做、可追溯。

## 1. 目标
- 将 `http://192.168.232.130:8015/static/console/index.html` 左侧目录树中的历史文档，逐条迁移/改写到项目根目录 `tooltip-docs`。
- 迁移过程必须可中断、可恢复：下次可直接根据本设计文档和状态文件继续。

## 2. 强约束
- 只允许用无头浏览器读取源文档内容：必须“点开左侧目录树 -> 点开具体文档 -> 读取页面展示内容”。
- 不使用“直接查后台接口返回 JSON”作为文档内容来源。
- 不做批量一键写入脚本；按单条任务推进，一条一条完成并记录。

## 3. 过程文件（恢复入口）
- 设计文档：`/data/project/sport/tooltip-docs/IMPORT_DESIGN.md`
- 状态文件：`/data/project/sport/tooltip-docs/import-state.json`
- 左侧树快照（无头读取结果）：`/data/project/sport/tooltip-docs/_headless_doc_tree_snapshot.json`
- 执行日志：`/data/project/sport/tooltip-docs/import-progress.md`
- 证据目录（每条任务 1 个证据文件）：`/data/project/sport/tooltip-docs/evidence/`

## 4. 源与目标对应原则
- 源：控制台左侧“服务文档”页签（`如何编写服务 / 模板函数 / 模块处理 / 参数处理 / 拦截器`）。
- 目标：`/data/project/sport/tooltip-docs/` 下 `key/*.md`、`kv/key/*.md`、`kv/module/*.md`。
- 对应优先级：
1. `模块处理` -> `kv/module/<module>.md`
2. `参数处理` -> `kv/key/<handler_key>.md`
3. `如何编写服务` + `模板函数` -> `key/*.md`（概念类）
4. `拦截器` -> 先落到 `key/` 文档中的“相关机制”段，必要时新增 `kv/key/` 条目

## 5. 单条任务执行步骤（严格）
1. 用无头浏览器打开页面。
2. 在左侧目录树展开对应分组。
3. 点击目标文档（单条）。
4. 从页面正文读取：标题、子标题、示例、说明（不是接口 JSON 直接取值）。
5. 改写目标 `tooltip-docs` 文件（结合当前 Go 项目语义）。
6. 写证据文件：记录源分组、源标题、读取时间、关键摘录、截图路径。
7. 更新 `import-state.json` 对应任务状态（`pending -> completed`）。
8. 更新 `import-progress.md` 追加一条完成记录。

## 6. 状态机
- `pending`：未开始
- `in_progress`：正在处理（只允许一个任务）
- `completed`：已完成并有证据
- `blocked`：有阻塞（写明原因）

## 7. 分阶段待办（与文件对应）

### 阶段 A：模块处理（11 条）
- `M-001` `模块处理/sql` -> `kv/module/sql.md`
- `M-002` `模块处理/bulk_service` -> `kv/module/bulk_service.md`
- `M-003` `模块处理/model_save` -> `kv/module/model_save.md`
- `M-004` `模块处理/model_update` -> `kv/module/model_update.md`
- `M-005` `模块处理/model_delete` -> `kv/module/model_delete.md`
- `M-006` `模块处理/bulk_create` -> `kv/module/bulk_create.md`
- `M-007` `模块处理/bulk_upsert` -> `kv/module/bulk_upsert.md`
- `M-008` `模块处理/empty` -> `kv/module/empty.md`
- `M-009` `模块处理/http` -> `kv/module/http.md`
- `M-010` `模块处理/ldap` -> `kv/module/ldap.md`
- `M-011` `模块处理/service_flow` -> `kv/module/service_flow.md`

### 阶段 B：参数处理（核心 16 条）
- `H-001` `参数处理/service2field` -> `kv/key/service2field.md`
- `H-002` `参数处理/get_modify_data` -> `kv/key/get_modify_data.md`
- `H-003` `参数处理/update_field` -> `kv/key/update_field.md`（新增）
- `H-004` `参数处理/check_field` -> `kv/key/check_field.md`（新增）
- `H-005` `参数处理/update_array` -> `kv/key/update_array.md`（新增）
- `H-006` `参数处理/arr2obj` -> `kv/key/arr2obj.md`
- `H-007` `参数处理/filter_arr` -> `kv/key/filter_arr.md`
- `H-008` `参数处理/prop_arr` -> `kv/key/prop_arr.md`（新增）
- `H-009` `参数处理/arr2dict` -> `kv/key/arr2dict.md`（新增）
- `H-010` `参数处理/param2result` -> `kv/key/param2result.md`
- `H-011` `参数处理/params2result` -> `kv/key/params2result.md`
- `H-012` `参数处理/result2params` -> `kv/key/result2params.md`（新增）
- `H-013` `参数处理/result2map` -> `kv/key/result2map.md`（新增）
- `H-014` `参数处理/count2map` -> `kv/key/count2map.md`（新增）
- `H-015` `参数处理/combine_array` -> `kv/key/combine_array.md`（新增）
- `H-016` `参数处理/update_array_from_array` -> `kv/key/update_array_from_array.md`（新增）

### 阶段 C：概念类 Key（10 条）
- `K-001` `如何编写服务/什么是服务service` -> `key/service.md`
- `K-002` `如何编写服务/什么是参数` -> `key/params.md`
- `K-003` `如何编写服务/什么是参数处理` -> `key/handler_params.md`
- `K-004` `如何编写服务/什么是参数处理` -> `key/result_handler.md`
- `K-005` `如何编写服务/生命周期` -> `key/module.md`（补充生命周期段）
- `K-006` `模板函数/must` -> `key/check.md`
- `K-007` `模板函数/current_date_time` -> `key/default.md`
- `K-008` `模板函数/get_key` -> `key/template.md`
- `K-009` `模板函数/uuid` -> `key/key.md`（补充唯一键建议）
- `K-010` `模板函数/sub_str + replace` -> `key/if_template.md`

### 阶段 D：剩余模板补齐（按实际需要）
- 对未被 A/B/C 覆盖的 `key/*.md` 保持待办，逐个点读后再补齐：
  - `append_param`, `count_params`, `count_sql`, `data_file`, `data_json`, `model`, `model_field`, `modify_config`, `path`, `request_handler`, `require`, `save_field`, `sql_file`, `table`, `to`, `type`, `class_name`, `method` 等。

## 8. 证据记录格式（每条任务）
证据文件路径示例：`/data/project/sport/tooltip-docs/evidence/M-001.md`

建议模板：
```md
# M-001
- 时间：2026-05-07 16:30:00
- 源分组：模块处理
- 源文档：sql
- 目标文件：kv/module/sql.md
- 无头截图：<路径>
- 关键摘录：
  - 示例：
  - 说明：
- 改写要点：
  - ...
```

## 9. 如何快速恢复（下次接手）
1. 打开 `import-state.json`，看 `current_task_id` 和第一个 `pending` 任务。
2. 打开 `import-progress.md` 看最近完成记录和备注。
3. 打开对应 `evidence/<task_id>.md`，确认上条任务来源与改写风格。
4. 从下一个 `pending` 任务继续，严格执行“单条任务执行步骤”。

## 10. 当前基线状态
- 当前仅完成：流程设计与待办编排。
- 文档改写尚未开始，状态以 `import-state.json` 为准。
