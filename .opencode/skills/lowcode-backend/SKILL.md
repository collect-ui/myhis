# 低代码后端开发指南

> 由 AGENTS.md 提取，适用于新增 CRUD / 后端服务 / 前端页面的低代码开发。
> 加载方式：`skill({ name: "lowcode-backend" })` 或自动匹配。

## 3‑file CRUD 目录结构（标准模式）

每个业务模块在 `collect/<module>/<biz>/` 目录下按以下结构组织：

```
collect/<module>/<biz>/
├── index.yml        # 接口定义（增删改查）
├── base.sql         # 公共 WHERE 条件 + 排序 + 分页
├── query.sql        # 列表查询（require base.sql + 可选 join）
└── count.sql        # 统计查询（require base.sql 包 count）
```

### `base.sql` 模式 — 消除 query / count 重复 WHERE

不要分别在 query.sql 和 count.sql 里写两遍完全相同的 WHERE 条件。正确做法：

1. **`base.sql`** — 存放完整查询骨架（SELECT + FROM + WHERE + ORDER BY + 分页）：

```sql
-- collect/<module>/<biz>/base.sql
select a.*
from <table> a
where 1=1
{{ if .field1 }}
and a.field1 = {{.field1}}
{{ end }}
{{ if .search }}
and a.name like {{.search}}
{{ end }}
order by a.create_time desc
{{ if .pagination }}
offset {{.start}} rows fetch next {{.size}} rows only
{{ end }}
```

2. **`query.sql`** — 只需 `require('./base.sql')`，可追加 JOIN：

```sql
-- collect/<module>/<biz>/query.sql
select a.*
from (require('./base.sql')) a
left join other_table b on a.xxx_id = b.xxx_id
```

3. **`count.sql`** — 只需 `require('./base.sql')` 包 count：

```sql
-- collect/<module>/<biz>/count.sql
select count(1) as count
from (require('./base.sql')) a
```

`require()` 路径规则：
- `require('base.sql')` — 同目录，不加 `./` 也可以
- `require('./base.sql')` — 同目录，加 `./` 明确
- 不支持跨目录引用

### `base.sql` 内分页约定

`base.sql` 中统一用 `{{ if .pagination }}` 控制分页开关。因数据库类型为 **Oracle**，分页语法使用 `OFFSET...FETCH`，不能用 `LIMIT`：

```sql
{{ if .pagination }}
offset {{.start}} rows fetch next {{.size}} rows only
{{ end }}
```

⚠️ 本项目数据库是 Oracle，禁止使用 `LIMIT` 分页。如果未来切换为 SQLite/MySQL，此处改为：
```sql
limit {{.start}} , {{.size}}
```

### `base.sql` LIKE 查询写法

`base.sql` 中直接写 `like {{.field}}`，**不要**在 YAML `params` 中用 `template` 拼接 `%`。LIKE 的 `%` 由前端传参时自带：

```sql
{{ if .field }}
and a.field like {{.field}}
{{ end }}
```

YAML `params` 只设 `default: ""`：

```yaml
params:
  field:
    default: ""
```

### `index.yml` 分页配置

```yaml
- key: xxx_query
  http: true
  module: sql
  params:
    page:
      type: int
      default: 1
    size:
      default: 20
      type: int
    start:
      template: " ({{.page}}-1) * {{.size}}"
      exec: true
      type: int
    end:
      template: " {{.page}} * {{.size}}"
      exec: true
      type: int
    pagination:
      default: true
  data_file: query.sql
  count_file: count.sql
  pagination: pagination
```

`pagination: pagination` 含义：
- 让框架自动计算总条数（走 count.sql）
- 返回结构会带上 `count` 字段
- `data_file` 走 query.sql（带 `pagination=true` 时加 limit/fetch）
- `count_file` 走 count.sql（框架自动设置 `pagination=false` 以屏蔽分页）

## CRUD 增删改配置

### 新增（model_save）

```yaml
- key: xxx_save
  http: true
  module: model_save
  table: xxx_table
  params:
    xxx_id:
      template: "{{uuid}}"
    create_time:
      template: "{{current_date_time}}"
    create_user:
      template: "{{.session_user_id}}"
    field1:
      check:
        template: "{{must .field1}}"
        err_msg: "字段不能为空"
    field2:
      default: "some_default"
```

### 更新（model_update）

```yaml
- key: xxx_update
  http: true
  module: model_update
  table: xxx_table
  params:
    xxx_id:
      check:
        template: "{{must .xxx_id}}"
        err_msg: "ID不能为空"
    modify_time:
      template: "{{current_date_time}}"
  filter:
    xxx_id: "[xxx_id]"
```

`update_fields` 可以限定只更新指定字段：

```yaml
  update_fields:
    - field1
    - field2
```

### 软删除（推荐用 model_update，不走 model_delete）

```yaml
- key: xxx_delete
  http: true
  module: model_update
  table: xxx_table
  params:
    xxx_id_list:
      check:
        template: "{{must .xxx_id_list}}"
        err_msg: "请选择要删除的记录"
    is_delete:
      default: "1"
    modify_time:
      template: "{{current_date_time}}"
  filter:
    xxx_id__in: "[xxx_id_list]"
```

### 物理删除（model_delete）

```yaml
- key: xxx_delete
  http: true
  module: model_delete
  table: xxx_table
  params:
    xxx_id_list:
      check:
        template: "{{must .xxx_id_list}}"
        err_msg: "请选择要删除的记录"
  filter:
    xxx_id__in: "[xxx_id_list]"
```

## 后端 Service 路由注册

1. `collect/service_router.yml` 定义顶级 `services` 入口：

```yaml
services:
  - key: him
    name: 门诊号池
    path: 'him/service.yml'
```

2. 每个模块有自己的 `service.yml` 挂载子模块：

```yaml
# collect/him/service.yml
service:
  - name: "pix_outp_reg_master"
    path: "pix_outp_reg_master/index.yml"
```

前端调用 key 格式：`<module>.<biz_key>`，例如 `him.pix_outp_reg_master_query`。

## 前端低代码页面开发

### 最小页面接入需要 4 个落点

1. `collect/frontend/page_data/data/<group>/<page>.json` — 页面 JSON（渲染 + 动作编排）
2. `collect/frontend/page_data/index.yml` — key `frontend.<page>` 映射到 JSON 文件路径
3. `sys_menu` 菜单记录 — `api` 指向 `post:/template_data/data?service=frontend.<page>`
4. 目标页面依赖的后端 service

### 页面配置约束

- `ajax` 优先只做取数和最小字段映射（如 `dataList: "${data}"`、`count: "${count}"`）。
- 不要在 `ajax.adapt` 里同时处理列表筛选、选中态切换、表单回填、删除后重选等多段逻辑。
- 状态变更应拆到独立的 `update-store` / `update-form`，让"取数"和"状态变更"分开。
- 能直接取 `row.xxx`、`data.xxx` 的场景，不要额外套自执行函数 `(()=>{})()`。

### Store 作用域与 targetStore

- `__this__`：当前动作所在 store。
- `__parent__`：当前 store 的上级 store。
- `targetStore`：指定 `update-store` / `reload-init-action` 的目标 store。
- `activeStore`：业务变量，用于"从下往上传递回调 store"，常见写法 `activeStore: "${__this__}"`。

方向规则：
- 下层改上层：`targetStore: "__parent__"`
- 上层改下层：先拿到下层传上来的 `activeStore`，再按 `activeStore` 定位执行
- 同一状态（如弹框开关）必须始终由同一层维护

推荐模板 — 子层按钮打开上层弹框：

```json
{
  "tag": "update-store",
  "targetStore": "__parent__",
  "value": {
    "activeStore": "${__this__}",
    "dialogVisible": true,
    "dialogOp": "add"
  }
}
```

推荐模板 — 保存后刷新列表 + 树：

```json
[
  { "tag": "submit-form", "formName": "xxx-form" },
  {
    "tag": "ajax",
    "enable": "${xxxOp==='add'}",
    "api": "post:/template_data/data?service=module.xxx_save",
    "appendFormFields": "xxx-form"
  },
  {
    "tag": "ajax",
    "enable": "${xxxOp==='edit'}",
    "api": "post:/template_data/data?service=module.xxx_update",
    "appendFormFields": "xxx-form"
  },
  { "tag": "reload-init-action", "targetStore": "__parent__", "group": "reload-xxx-list" },
  { "tag": "update-store", "value": { "dialogVisible": false } }
]
```

### 表单实战约定

- 编辑场景优先用 `update-form`，不依赖 `initialValues`。
- 推荐链路：`查询当前行 → update-store 缓存 row → 延迟 update-form(整行透传) → submit-form → appendFormFields 一次性保存`。
- 表单不展示但接口必填字段（如主键），在 `ajax.data` 单独补：`xxx_id: "${xxx_id || currentRow.xxx_id || ''}"`。
- 保存/查询优先使用 `appendFormFields`。
- 分页参数（`page/size`）不在表单时，用 `appendFields` 额外拼接。

## 禁止事项

- **禁止写 Go 业务代码**：除 model 定义、表注册等模型层变更外，不要私自新增或修改 Go 业务代码。确实需要写 Go 时，必须先向用户说明要写什么、为什么必须写、为什么不能用低代码配置或已有服务替代，并获得确认后再实施。
- **禁止前端 schema 匿名函数**：绝对禁止在前端 JSON 中使用 `${(()=>{...})()}`、`function(){...}`、`JSON.parse`、`JSON.stringify` 等。数据组装放到后端低代码服务中处理。
- **低代码优先原则**：能用低代码（sql/model_save/model_update/empty + handler_params + service2field 编排）解决的功能，禁止新增或修改 Go 插件/handler。
