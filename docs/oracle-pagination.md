# Oracle 分页模式

本项目数据库为 Oracle，分页统一使用 `rownum` 模式（与老系统一致），**禁止**在 SQL 模板中手写 `OFFSET ... FETCH NEXT`。

## 三文件结构

每个需要分页的查询拆为三个文件：

```
module/
├── xxx.sql           # 基础查询（不含分页）
├── xxx_query.sql     # 分页包装（require 基础查询 + rownum）
└── xxx_count.sql     # 总数查询（与基础查询条件一致，select count(*)）
```

### 1. 基础查询 `xxx.sql`

纯业务 SQL，不带任何分页逻辑：

```sql
select a.id as "id", a.name as "name"
from some_table a
where 1=1
  {{if .area_code}}and a.area_code = {{.area_code}}{{end}}
order by a.id desc
```

### 2. 分页包装 `xxx_query.sql`

**固定写法**，用 `require()` 引入基础查询并包裹 `rownum`：

```sql
select * from (select a.*, rownum rn from (require('./xxx.sql')) a where rownum <= {{.end}}) where rn > {{.start}}
```

- `{{.end}}` = `page * size`（由 index.yml 模板计算）
- `{{.start}}` = `(page - 1) * size`（由 index.yml 模板计算）
- **不要**在基础查询里写 `OFFSET/FETCH NEXT`，GORM 的 Oracle 驱动会自动追加，手写会导致 `ORA-00933`

### 3. 总数查询 `xxx_count.sql`

与基础查询完全相同的 WHERE 条件，但只返回 `count(*)`：

```sql
select count(*) from some_table a
where 1=1
  {{if .area_code}}and a.area_code = {{.area_code}}{{end}}
```

## index.yml 配置

```yaml
- key: module_xxx_query
  http: true
  must_login: false
  module: sql
  params:
    area_code:
      default: ""
    # ... 业务参数 ...
    page:
      type: int
      default: 1
    size:
      type: int
      default: 20
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
  data_file: xxx_query.sql        # 指向分页包装
  count_file: xxx_count.sql       # 指向 count
  pagination: pagination          # 分页开关字段
```

关键字段说明：

| 字段 | 作用 |
|---|---|
| `page` / `size` | 前端传入当前页码和每页条数，默认 1/20 |
| `start` / `end` | 由模板自动计算，基础查询无需感知 |
| `pagination` | 布尔开关，`true` 时框架执行 count 并分页 |
| `data_file` | 必须指向 `xxx_query.sql`（分页包装），**不是**基础查询 |
| `count_file` | 必须指向 `xxx_count.sql`，框架用它算总数 |

## 前端调用

前端请求 `template_data/data?service=module_xxx_query` 时：

- **带分页**：传 `page` 和 `size` 参数，返回 `{ data: [...], count: 100 }`
- **不传 page/size**：使用 index.yml 中的默认值（page=1, size=20）

```json
// initAction 加载初始数据
{
  "tag": "ajax",
  "api": "post:/template_data/data?service=module_xxx_query",
  "adapt": { "dataList": "${data}", "count": "${count}" }
}

// pagination 组件翻页
{
  "tag": "pagination",
  "total": "${count}",
  "current": "${page}",
  "pageSize": "${size}"
}
```

## 常见错误

### ❌ SQL 模板里手写 OFFSET/FETCH NEXT

```sql
-- 错误！框架的 GORM Oracle 驱动会自动追加 OFFSET/FETCH，
-- 手写会导致 ORA-00933: SQL command not properly ended
select * from (...) order by id OFFSET 0 ROWS FETCH NEXT 20 ROWS ONLY
```

**原因**：GORM 的 `Limit()` + `Offset()` 会在 SQL 末尾自动追加 `OFFSET m ROWS FETCH NEXT n ROWS ONLY`。如果模板里已经写了，就会出现两个分页子句。

### ❌ data_file 直接指向基础查询

```yaml
# 错误！基础查询没有 rownum 包装，分页无效
data_file: xxx.sql
```

**正确**：`data_file` 必须指向分页包装 `xxx_query.sql`。

### ❌ count.sql 与基础查询条件不一致

count 查询必须与基础查询的 WHERE 条件完全一致（除了 ORDER BY 和 SELECT 列），否则分页总数不正确。

## 参考实现

`collect/him/pix_outp_reg_master/` 下的文件是完整的分页示例：
- `doctor_options.sql` — 基础查询
- `doctor_options_query.sql` — 分页包装
- `doctor_options_count.sql` — 总数查询
- `index.yml` 第 79-112 行 — 分页参数配置
