# 翻译模块设计文档

## 1. 背景

当前码表翻译通过 SQL LEFT JOIN 实现，性能差且 SQL 与码表耦合。需要一个可复用的翻译模块：不走 SQL JOIN，复用现有低代码 service + `handler_cache` 缓存。

## 2. 核心架构

翻译模块只有两个核心操作：**存** 和 **取**。

```
┌─────────────────────────────────────────────────┐
│                  translate                      │
│                                                 │
│  ┌─────────────────┐   ┌─────────────────┐     │
│  │       存        │   │       取        │     │
│  │                 │   │                 │     │
│  │  service 查询   │   │  查缓存         │     │
│  │       ↓        │   │    ↓            │     │
│  │  写入缓存       │   │  命中 → 返回    │     │
│  │                 │   │  未命中 ↓       │     │
│  └─────────────────┘   │  调 service     │     │
│                        │    ↓            │     │
│                        │  写缓存 → 返回  │     │
│                        └─────────────────┘     │
└─────────────────────────────────────────────────┘
```

### 存（Store）

调 service 查数据 → 写入 Ristretto 缓存

触发时机：
- **预热**：启动时加载常用码表（如 `outp_type_code`、`sex`）
- **懒加载**：运行时缓存未命中，自动调 service 补数据

### 取（Fetch）

```
查缓存
  → 命中：直接返回
  → 未命中：
      1. 调 service 查询数据
      2. 写入缓存（存）
      3. 返回数据
```

## 3. 配置方式

### 3.1 结果列表（翻译对象）

翻译的目标是查询返回的结果列表。以 `reg_fee_options` 为例：

```json
[
  {"reg_fee_id": "1", "reg_fee_name": "普通号", "outp_type_code": "1", "area_code": "01"},
  {"reg_fee_id": "2", "reg_fee_name": "专家号", "outp_type_code": "3", "area_code": "01"}
]
```

其中 `outp_type_code` 是编码字段，需要通过 BCS 码表（`BCS.BCS_CODE_TABLE.STANDARD_CODE = 'PIX0021'`）翻译成文字。`reg_fee_name` 是主表显示字段，不需要翻译。

### 3.2 翻译配置

在 `index.yml` 的 `result_handler` 中声明：

**reg_fee_options（逐行翻译 outp_type_code）：**

```yaml
result_handler:
  - key: translate              # 固定值，翻译处理器
    type: "code"                # 翻译类型：code=BCS码表翻译
    fields:                     # 翻译字段列表
      - from: "PIX0021"                 # BCS 码表 STANDARD_CODE
        field: "[item.outp_type_code]"  # 从每行取这个字段的值作为编码
        to: "outp_type_code_str"        # 翻译结果写入这个字段
```

- `foreach` 和 `item` 不传时使用默认值：`foreach="[result]"`, `item="item"`
- `from`: BCS 码表标准编码（对应 `BCS.BCS_CODE_TABLE.STANDARD_CODE`，如 `PIX0021`）
- `field`: 取哪个字段的值作为编码（支持 `[item.xxx]` 模板）
- `to`: 翻译结果写入哪个字段

### 3.3 字段映射（HandlerParam → 翻译配置）

| HandlerParam 字段 | 翻译配置 | 说明 |
|-------------------|----------|------|
| `Key` | `key: translate` | 固定值 |
| `Type` | `type: "code"` | 翻译类型，决定内部 service 和字段 |
| `Fields[].From` | `from: "PIX0021"` | BCS 码表标准编码 |
| `Fields[].Field` | `field: "[item.xxx]"` | 取哪个字段的值 |
| `Fields[].To` | `to: "xxx_str"` | 填充到哪个字段 |

### 3.4 翻译类型

| type | 含义 | 内部 service | 内部 key_field | 内部 text_field |
|------|------|-------------|----------------|-----------------|
| `code` | BCS码表翻译 | `system.get_bcs_code` | `ITEM_VALUE` | `ITEM_NAME` |

> 内部字段由 handler 根据 `type` 自动确定，配置时只需传 `type`。

### 3.5 翻译注册表（内部定义）

翻译注册表定义了每种 type 的内部行为，handler 启动时加载，不暴露给业务配置：

```json
{
  "registry": [
    {
      "type": "code",                              // 翻译类型
      "service": "system.get_bcs_code",            // 调用的 service 名
      "params": {                                  // 单个翻译时传给 service 的参数
        "standard_code": "[from]"                  // [from] = 业务配置 fields[].from 的值（STANDARD_CODE）
      },
      "preload_params": {                          // 预热参数，有值则预热，空对象 {} 不预热
        "standard_code_list": ["PIX0021", "PIX0022", "PIX0023"]  // 预热的 BCS 码表 STANDARD_CODE 列表
      },
      "key_field": "ITEM_VALUE",                   // 返回数据中做 key 的字段名
      "text_field": "ITEM_NAME",                   // 返回数据中做 value 的字段名
      "cache": {                                   // 缓存配置
        "room": "code",                            // 缓存命名空间
        "key": "[from#key_field]",                 // 单个翻译缓存 key 模板
        "preload_key": "[STANDARD_CODE#key_field]",// 预热缓存 key 模板，用返回数据的实际字段名
        "seconds": 3600                            // 缓存时长（秒）
      }
    }
  ]
}
```

**两种调用模式：**

| 模式 | 触发场景 | 参数来源 | 缓存 key |
|------|----------|----------|----------|
| 单个翻译 | 运行时 | `params` + `[from]` 替换 | `[from#key_field]` → `code[PIX0021#1]` |
| 预热 | 启动时 | `preload_params` | `[STANDARD_CODE#ITEM_VALUE]` → `code[PIX0021#1]` |

两种模式最终写入相同的缓存 key，预热的数据在运行时直接命中。

## 4. 执行流程

### 4.1 reg_fee_options 翻译流程

```
输入: reg_fee_options 查询结果
[
  {reg_fee_id:"1", reg_fee_name:"普通号", outp_type_code:"1", area_code:"01"},
  {reg_fee_id:"2", reg_fee_name:"专家号", outp_type_code:"3", area_code:"01"}
]

result_handler 配置:
  type: "code"
  fields:
    - from: "PIX0021"               → BCS 码表 STANDARD_CODE
      field: "[item.outp_type_code]" → 取每行的 outp_type_code 值
      to: "outp_type_code_str"       → 写入 outp_type_code_str 字段

执行步骤:
  1. 遍历每行数据
     item = {reg_fee_id:"1", reg_fee_name:"普通号", outp_type_code:"1"}

  2. 取编码值
     from = "PIX0021"
     itemValue = item["outp_type_code"] = "1"

  3. 构造缓存 key
     cacheKey = "code[PIX0021#1]"

  4. 查缓存
     → 命中（预热已加载）→ "普通号"
     → 未命中 → 调 service: system.get_bcs_code({standard_code: "PIX0021"})
                → 从结果中找 ITEM_VALUE="1" 的行 → 取 ITEM_NAME = "普通号"
                → 写入缓存 code[PIX0021#1] = "普通号"

  5. 填充翻译字段
     item["outp_type_code_str"] = "普通号"

输出:
[
  {reg_fee_id:"1", reg_fee_name:"普通号", outp_type_code:"1", outp_type_code_str:"普通号", area_code:"01"},
  {reg_fee_id:"2", reg_fee_name:"专家号", outp_type_code:"3", outp_type_code_str:"专家号", area_code:"01"}
]
```

### 4.2 多字段翻译

如果一行数据有多个码表字段需要翻译：

```yaml
result_handler:
  - key: translate
    type: "code"
    fields:
      - from: "PIX0021"               # BCS 码表 STANDARD_CODE
        field: "[item.outp_type_code]"
        to: "outp_type_code_str"
      - from: "PIX0022"               # 另一个 BCS 码表
        field: "[item.area_code]"
        to: "area_code_str"
```

```
输入: [{outp_type_code:"1", area_code:"01"}]

翻译字段1: outp_type_code → "1" → cache key "code[PIX0021#1]" → "普通号"
翻译字段2: area_code → "01" → cache key "code[PIX0022#01]" → "门诊部"

输出: [{outp_type_code:"1", outp_type_code_str:"普通号", area_code:"01", area_code_str:"门诊部"}]
```

## 5. 缓存设计

使用项目已有的 Ristretto 内存缓存（`handler_cache`），key 格式：`{room}[{key}]`

**单个翻译**：

用 `key` 表达式，从配置的 `from` 和 `key_field` 组合：
```
room=code, key=PIX0021#1    → "普通号"
room=code, key=PIX0021#3    → "专家号"
room=code, key=PIX0022#M    → "男"
```

**预热**：

用 `preload_key` 表达式，从返回数据的 `STANDARD_CODE` 和 `ITEM_VALUE` 组合：
```
预热结果: [
  {STANDARD_CODE:"PIX0021", ITEM_VALUE:"1", ITEM_NAME:"普通号"},
  {STANDARD_CODE:"PIX0021", ITEM_VALUE:"3", ITEM_NAME:"专家号"},
  {STANDARD_CODE:"PIX0022", ITEM_VALUE:"M", ITEM_NAME:"男"}
]

写入缓存:
room=code, key=PIX0021#1    → "普通号"
room=code, key=PIX0021#3    → "专家号"
room=code, key=PIX0022#M    → "男"
```

**翻译时查找**：
```
room=code, key=PIX0021#1
→ 命中 → "普通号"
```

## 6. 业务 SQL 改造

### 6.1 reg_fee_options（挂号费别下拉）

**改造前（当前 SQL）：**

```sql
select f.reg_fee_id as "value",
       f.reg_fee_name as "label",
       f.reg_fee_id as "reg_fee_id",
       f.reg_fee_name as "reg_fee_name",
       f.outp_type_code as "outp_type_code",
       nvl(otc.ITEM_NAME, f.outp_type_code) as "outp_type_code_str",        -- ← JOIN 翻译
       f.area_code as "area_code",
       f.py_code as "py_code",
       f.wb_code as "wb_code",
       f.reserve_source as "reserve_source"
from pix_reg_type_fee f
left join BCS.BCS_CODE_TABLE_ITEM otc                                       -- ← 去掉
  on otc.ITEM_VALUE = f.outp_type_code                                      -- ← 去掉
  and otc.T_ID = (SELECT T.TYPE_ID FROM BCS.BCS_CODE_TABLE T WHERE T.STANDARD_CODE = 'PIX0021') -- ← 去掉
  and otc.IS_ENABLE = '1'                                                   -- ← 去掉
  and otc.AUDIT_STATUS = '2'                                                -- ← 去掉
where f.reg_fee_id is not null
  and nvl(f.delete_flag, '0') = '1'
  and nvl(f.is_enable, '1') = '1'
  {{if .area_code}}and f.area_code = {{.area_code}}{{end}}
  {{if .reg_fee_id}}and f.reg_fee_id = {{.reg_fee_id}}{{end}}
```

**改造后（翻译移至 result_handler）：**

```sql
select f.reg_fee_id as "value",
       f.reg_fee_name as "label",
       f.reg_fee_id as "reg_fee_id",
       f.reg_fee_name as "reg_fee_name",
       f.outp_type_code as "outp_type_code",                              -- 保留原始编码
       f.area_code as "area_code",
       f.py_code as "py_code",
       f.wb_code as "wb_code",
       f.reserve_source as "reserve_source"
from pix_reg_type_fee f
where f.reg_fee_id is not null
  and nvl(f.delete_flag, '0') = '1'
  and nvl(f.is_enable, '1') = '1'
  {{if .area_code}}and f.area_code = {{.area_code}}{{end}}
  {{if .reg_fee_id}}and f.reg_fee_id = {{.reg_fee_id}}{{end}}
```

**result_handler 配置（添加到 index.yml）：**

```yaml
result_handler:
  - key: translate              # 固定值，翻译处理器
    type: "code"                # 翻译类型：code=BCS码表翻译
    fields:                     # 翻译字段列表
      - from: "PIX0021"                 # BCS 码表 STANDARD_CODE
        field: "[item.outp_type_code]"  # 从每行取这个字段的值作为编码
        to: "outp_type_code_str"        # 翻译结果写入这个字段
```

### 6.2 outp_type_options（门诊类型下拉）

**改造前：**

```sql
select distinct f.outp_type_code as "value",
       nvl(otc.ITEM_NAME, f.outp_type_code) as "label"
from pix_reg_type_fee f
left join BCS.BCS_CODE_TABLE_ITEM otc
  on otc.ITEM_VALUE = f.outp_type_code
  and otc.T_ID = (SELECT T.TYPE_ID FROM BCS.BCS_CODE_TABLE T WHERE T.STANDARD_CODE = 'PIX0021')
  and otc.IS_ENABLE = '1'
  and otc.AUDIT_STATUS = '2'
where f.outp_type_code is not null
  and nvl(f.delete_flag, '0') = '1'
order by f.outp_type_code
```

**改造后：**

```sql
select distinct f.outp_type_code as "value",
       f.outp_type_code as "label"   -- 先用原始编码，翻译后替换
from pix_reg_type_fee f
where f.outp_type_code is not null
  and nvl(f.delete_flag, '0') = '1'
order by f.outp_type_code
```

```yaml
result_handler:
  - key: translate
    type: "code"
    fields:
      - from: "PIX0021"
        field: "[item.value]"          # 从每行的 value 字段取编码
        to: "label"                    # 翻译结果覆盖 label 字段
```

### 6.3 翻译结果示例

**reg_fee_options 翻译前后对比：**

```
翻译前（SQL JOIN）：
[
  {"reg_fee_id":"1","reg_fee_name":"普通号","outp_type_code":"1","outp_type_code_str":"普通号"},
  {"reg_fee_id":"2","reg_fee_name":"专家号","outp_type_code":"3","outp_type_code_str":"专家号"}
]

翻译后（result_handler）：
[
  {"reg_fee_id":"1","reg_fee_name":"普通号","outp_type_code":"1","outp_type_code_str":"普通号"},
  {"reg_fee_id":"2","reg_fee_name":"专家号","outp_type_code":"3","outp_type_code_str":"专家号"}
]

→ 输出完全一致，对前端透明
```

**outp_type_options 翻译前后对比：**

```
翻译前：
[{"value":"1","label":"普通门诊"}, {"value":"3","label":"专家门诊"}]

翻译后：
[{"value":"1","label":"普通门诊"}, {"value":"3","label":"专家门诊"}]

→ 输出完全一致
```

### 6.4 验收办法

**步骤 1：确认码表数据存在**

```sql
-- 检查 BCS 码表中 PIX0021 类型的数据
SELECT ci.ITEM_VALUE, ci.ITEM_NAME, ct.STANDARD_CODE
FROM BCS.BCS_CODE_TABLE_ITEM ci
JOIN BCS.BCS_CODE_TABLE ct ON ct.TYPE_ID = ci.T_ID
WHERE ct.STANDARD_CODE = 'PIX0021'
  AND ci.IS_ENABLE = '1'
  AND ci.AUDIT_STATUS = '2'
ORDER BY ci.ITEM_VALUE;
```

预期结果（示例）：
```
ITEM_VALUE | ITEM_NAME | STANDARD_CODE
-----------|-----------|--------------
1          | 普通号     | PIX0021
2          | 急诊       | PIX0021
3          | 专家号     | PIX0021
```

**步骤 2：启动应用，检查预热日志**

```bash
./linux-shutdown && ./linux-startup
# 启动日志中应有：
# [translate] 预热码表类型: outp_type_code → 写入缓存 code[outp_type_code#1]
# [translate] 预热码表类型: outp_type_code → 写入缓存 code[outp_type_code#2]
# [translate] 预热码表类型: outp_type_code → 写入缓存 code[outp_type_code#3]
```

**步骤 3：HTTP 调用验证翻译结果**

```bash
# 调用挂号费别接口
curl -s 'http://127.0.0.1:8017/api/him/pix_outp_reg_master/reg_fee_options' \
  -H 'Cookie: ...' | python3 -m json.tool
```

预期返回：
```json
{
  "code": 200,
  "data": [
    {
      "value": "1",
      "label": "普通号",
      "reg_fee_id": "1",
      "reg_fee_name": "普通号",
      "outp_type_code": "1",
      "outp_type_code_str": "普通门诊",    // ← 翻译结果
      "area_code": "01"
    }
  ]
}
```

**步骤 4：对比翻译前后输出**

```bash
# 改造前：调用接口，保存结果
curl -s 'http://127.0.0.1:8017/api/him/pix_outp_reg_master/reg_fee_options' \
  -H 'Cookie: ...' > before.json

# 改造后：调用接口，保存结果
curl -s 'http://127.0.0.1:8017/api/him/pix_outp_reg_master/reg_fee_options' \
  -H 'Cookie: ...' > after.json

# 对比（忽略字段顺序差异）
diff <(python3 -c "import json,sys; print(json.dumps(json.load(sys.stdin),sort_keys=True,ensure_ascii=False))" < before.json) \
     <(python3 -c "import json,sys; print(json.dumps(json.load(sys.stdin),sort_keys=True,ensure_ascii=False))" < after.json)

# 预期：无差异
```

**步骤 5：检查缓存命中**

```bash
# 第二次调用应全部命中缓存，无 service 调用
# 可通过日志或 debug 模式确认：
# [translate] outp_type_code#1 命中缓存
# [translate] outp_type_code#3 命中缓存
```

**步骤 6：清理缓存验证懒加载**

```bash
# 重启应用（清空内存缓存）
./linux-shutdown && ./linux-startup

# 首次调用触发懒加载
curl -s 'http://127.0.0.1:8017/api/him/pix_outp_reg_master/reg_fee_options' \
  -H 'Cookie: ...' > after.json

# 与 before.json 对比，结果应一致
```

## 7. 服务层设计

### 7.0 BCS 码表查询服务（BCS Code Service）

BCS 码表查询服务直接查 `BCS.BCS_CODE_TABLE` + `BCS.BCS_CODE_TABLE_ITEM`，返回标准化字段。

**目录结构：**

```
collect/system/bcs_code/
├── index.yml          # 服务定义
└── get_bcs_code.sql   # 查询 SQL
```

**服务定义（index.yml）：**

```yaml
service:
  - key: get_bcs_code
    http: true
    module: sql
    params:
      standard_code:
        check:
          template: "{{must .standard_code}}"
          err_msg: 码表类型不能为空
    data_file: get_bcs_code.sql
```

**查询 SQL（get_bcs_code.sql）：**

```sql
SELECT
  C.STANDARD_CODE AS STANDARD_CODE,    -- 码表标准编码（如 PIX0021）
  T.ITEM_VALUE AS ITEM_VALUE,          -- 码表项值（如 "1"）
  T.ITEM_NAME AS ITEM_NAME             -- 码表项名称（如 "普通号"）
FROM
  BCS.BCS_CODE_TABLE_ITEM T
  LEFT JOIN BCS.BCS_CODE_TABLE C ON C.TYPE_ID = T.T_ID
WHERE
  C.STANDARD_CODE = {{.standard_code}}       -- 按标准编码查询
  AND T.AUDIT_STATUS = '2'                   -- 已审核
  AND T.IS_ENABLE = '1'                      -- 启用
  {{if .area_code}}AND (T.AREA_CODE IS NULL OR T.AREA_CODE = {{.area_code}}){{end}}
  {{if .sys_code_list}}AND T.ITEM_VALUE IN ({{.sys_code_list}}){{end}}
```

**路由注册（system/service.yml）：**

```yaml
# collect/system/service.yml 添加：
  - name: "翻译注册表"
    path: "translate_registry/index.yml"
```
GET /api/system/bcs_code/get_bcs_code?standard_code=PIX0021
→ 返回 [{STANDARD_CODE:"PIX0021", ITEM_VALUE:"1", ITEM_NAME:"普通号"}, ...]
```

### 7.1 翻译注册服务（Translate Registry Service）

翻译注册表是一个低代码服务，通过 `module: empty` + `file2datajson` 从 JSON 配置文件返回数据。

**目录结构：**

```
collect/system/translate_registry/
├── index.yml                  # 服务定义
└── get_translate_registry.json # 注册表配置数据
```

**服务定义（index.yml）：**

```yaml
service:
  - key: get_translate_registry
    http: true
    module: empty
    handler_params:
      - key: file2datajson
        save_field: data
      - key: param2result
        field: data
    data_file: get_translate_registry.json
    cache:                              # 框架自动处理缓存
      key: "handler_cache"
      room: translate_registry          # 缓存命名空间
      second: 86400                     # 缓存时长（秒）
      fields:
        - field: "[service]"            # 按服务名缓存
```

> **关键点：** 注册表服务自带 `cache:` 配置，框架在执行时自动：
> - 执行前：查缓存 → 命中直接返回（跳过 file2datajson）
> - 执行后：结果写入缓存
> 
> handler 不需要手动管理注册表缓存。

**注册表配置数据（get_translate_registry.json）：**

```json
{
  "code": 200,
  "msg": "success",
  "data": [
    {
      "type": "code",
      "service": "system.get_bcs_code",
      "params": {
        "standard_code": "[from]"
      },
      "preload_params": {
        "service": "system.get_bcs_code",
        "standard_code_list": ["PIX0021", "PIX0022", "PIX0023"]
      },
      "key_field": "ITEM_VALUE",
      "text_field": "ITEM_NAME",
      "cache": {
        "room": "code",
        "key": "[from ITEM_VALUE]",
        "preload_key": "STANDARD_CODE ITEM_VALUE",
        "seconds": 3600
      }
    }
  ]
}
```

**获取方式（低代码配置获取，非 HTTP）：**

```go
// 构造 service 参数
serviceParam := map[string]interface{}{
    "service": "system.get_translate_registry",
}

// 调用服务（内部执行 file2datajson，读取 JSON 配置文件）
result := ts.ResultInner(serviceParam)
// result.Data = [{type:"code", service:"system.get_bcs_code", ...}, ...]
```

> **关键点：** 注册表数据来自 JSON 配置文件，通过低代码服务框架读取，不是 HTTP 调用。

### 7.2 预热服务（Preload Service）

预热是一个启动服务，复用低代码 service + `handler_params` 框架，通过 `translate` handler 在无 `type`、无 `fields` 时自动进入预热模式。

**目录结构：**

```
collect/system/translate_preload/
└── index.yml                  # 服务定义
```

**服务定义（index.yml）：**

```yaml
service:
  - key: translate_preload     # 启动时自动执行
    http: false                 # 仅启动时执行，不暴露 HTTP
    module: empty
    startup: true               # 标记为启动服务，应用启动时自动执行
    handler_params:
      - key: translate          # 调用 translate handler
        # 无 type、无 fields → 进入预热模式
```

> **关键设计：** `translate` handler 检查参数：
> - 无 `type`、无 `fields` → **预热模式**（加载全部注册条目 → 批量查询 → 写缓存）
> - 有 `type`、有 `fields` → **取模式**（逐行翻译 → 缓存命中直接返回 → 未命中查一个 → 补缓存）

**预热流程（translate handler 内部）：**

```
translate handler 启动时（无 type、无 fields）：
  1. 调 HTTP: system.get_translate_registry → 拿到注册表 JSON
  2. 解析注册表 data 列表（全部条目）
  3. 遍历每个 type 条目：
     a. 将注册表条目本身写入缓存：translate_registry[code] → {service, params, ...}
     b. 检查 preload_params 是否有值（非空对象）
     c. 有值 → 用 preload_params 调用 service（如 system.get_bcs_code）
     d. 拿到返回数据，遍历每行：
        - 用 preload_key 模板构造缓存 key
          例：[STANDARD_CODE#ITEM_VALUE] → "PIX0021#1"
        - 取 text_field 的值作为缓存 value
          例：ITEM_NAME → "普通号"
        - 写入 Ristretto 缓存（带 TTL）
```

**预热结果示例：**

```
注册表 preload_params:
  standard_code_list: ["PIX0021", "PIX0022"]

调用 service.get_bcs_code({standard_code_list: ["PIX0021", "PIX0022"]})
返回:
  [
    {STANDARD_CODE: "PIX0021", ITEM_VALUE: "1", ITEM_NAME: "普通号"},
    {STANDARD_CODE: "PIX0021", ITEM_VALUE: "3", ITEM_NAME: "专家号"},
    {STANDARD_CODE: "PIX0022", ITEM_VALUE: "M", ITEM_NAME: "男"},
    {STANDARD_CODE: "PIX0022", ITEM_VALUE: "F", ITEM_NAME: "女"}
  ]

写入缓存:
  code[PIX0021#1] = "普通号"
  code[PIX0021#3] = "专家号"
  code[PIX0022#M] = "男"
  code[PIX0022#F] = "女"
```

### 7.3 翻译处理器（Translate Handler）

翻译处理器是一个 `data_handler`，支持三种模式：**存**、**取**、**预热**。

**路由注册（service_router.yml → data_handler）：**

```yaml
# data_handler 添加：
  - key: translate
    name: 翻译处理器
    type: outer
    path: Translate
```

**三种模式对比：**

| 模式 | 条件 | 作用 |
|------|------|------|
| **预热** | 无 `type`、无 `fields` | 加载全部注册条目 → 批量查询 → 写入缓存 |
| **取** | 有 `type`、有 `fields` | 逐行翻译 → 缓存命中直接返回 → 未命中查一个 → 写缓存 → 返回 |
| **存** | 有 `type`、有 `fields`、有 `data` | 将指定数据写入缓存（供外部调用） |

**关键实现说明：**

| 组件 | 正确用法 | 错误用法 |
|------|----------|----------|
| 缓存 key | `ch.GetCacheKey(room, fields, params)` → `room[field1:field2:...]` | ~~`room[from#key]`~~ |
| 缓存值 | `common.Ok(data, "msg")` 存 Result 对象 | ~~存原始字符串~~ |
| 调 service | `ts.ResultInner(serviceParam)` 内部调用 | ~~HTTP 调用~~ |
| 取注册表 | `ch.Get(key)` 返回 `common.Result`，取 `.Data` | ~~直接返回对象~~ |

**handler 核心逻辑：**

```go
// plugins/handler_result_translate.go
// 翻译处理器：支持预热、取两种模式
// 预热模式：无 type、无 fields → 加载注册表 → 批量查询 → 写缓存
// 取模式：有 type + fields → 逐行翻译 → 缓存命中返回 → 未命中查一个 → 写缓存

package plugins

import (
    common "github.com/collect-ui/collect/src/collect/common"           // 通用工具：Result、Ok、NotOk
    config "github.com/collect-ui/collect/src/collect/config"           // 配置：Template、HandlerParam
    templateService "github.com/collect-ui/collect/src/collect/service_imp"  // 模板服务：TemplateService
    cacheHandler "github.com/collect-ui/collect/src/collect/service_imp/cache_handler"  // 缓存：CacheHandler
    utils "github.com/collect-ui/collect/src/collect/utils"             // 工具：GetAppKey、RenderVar、Copy
    collect "github.com/collect-ui/collect/src/collect/utils"           // 别名：GetAppKey
    "strings"                                                           // 字符串操作：Trim、Split
)

// Translate 翻译处理器结构体，继承 BaseHandler
type Translate struct {
    templateService.BaseHandler  // 基础处理器，提供通用方法
}

// HandlerData 处理器入口方法
// 参数:
//   - template: 模板对象，包含当前请求的 params 数据
//   - handlerParam: 处理器配置参数，包含 type、fields、foreach、item 等
//   - ts: 模板服务，用于调用其他服务（ResultInner）
// 返回:
//   - *common.Result: 处理结果
func (t *Translate) HandlerData(template *config.Template,
    handlerParam *config.HandlerParam, ts *templateService.TemplateService) *common.Result {

    // 创建缓存处理器实例
    ch := cacheHandler.CacheHandler{}

    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    // 预热模式判断：无 type、无 fields → 进入预热模式
    // 预热模式用于启动时批量加载码表数据到缓存
    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    if handlerParam.Type == "" && len(handlerParam.Fields) == 0 {
        return t.preload(handlerParam, ch, ts)
    }

    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    // 取模式判断：有 type + fields → 进入取模式
    // 取模式用于运行时逐行翻译业务数据
    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    if len(handlerParam.Fields) > 0 {
        return t.translate(template, handlerParam, ch, ts)
    }

    // 参数不匹配任何模式，返回错误
    return common.NotOk("translate handler 参数错误：需要 type+fields 或无参数")
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// 预热方法：无 type、无 fields
// 流程：读配置 → 调服务获取注册表 → 遍历条目 → 调 service → 写缓存
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
func (t *Translate) preload(handlerParam *config.HandlerParam,
    ch cacheHandler.CacheHandler, ts *templateService.TemplateService) *common.Result {

    // ── 步骤 1：从 application.properties 读取完整服务名 ──
    // 配置文件 conf/application.properties 中定义：
    // translate_registry_service=system.get_translate_registry
    // GetAppKey 是框架提供的配置读取方法
    serviceName := collect.GetAppKey("translate_registry_service")
    // serviceName = "system.get_translate_registry"

    // ── 步骤 3：调用服务获取注册表数据 ──
    // service2field 模式：只需要传 "service" 字段
    // ts.ResultInner() 会执行 service 对应的低代码服务定义
    // 返回的 result.Data 是注册表的 data 数组
    serviceParam := map[string]interface{}{
        "service": serviceName,  // 要调用的服务名
    }
    result := ts.ResultInner(serviceParam)
    // 检查服务调用是否成功
    if !result.Success {
        return common.NotOk("获取注册表失败: " + result.Msg)
    }

    // ── 步骤 4：解析注册表条目 ──
    // result.Data 类型是 []map[string]interface{}
    // 每个元素是一个注册表条目，包含 type、service、key_field 等字段
    registryData := result.GetData().([]map[string]interface{})

    // ── 步骤 5：遍历每个注册表条目 ──
    for _, entryMap := range registryData {
        // 将 map 转换为 RegistryEntry 结构体
        entry := &RegistryEntry{
            Type:          entryMap["type"].(string),          // 翻译类型，如 "code"
            Service:       entryMap["service"].(string),       // 调用的服务名，如 "system.get_bcs_code"
            KeyField:      entryMap["key_field"].(string),     // 匹配编码的字段名，如 "ITEM_VALUE"
            TextField:     entryMap["text_field"].(string),     // 取翻译文本的字段名，如 "ITEM_NAME"
            PreloadParams: entryMap["preload_params"].(map[string]interface{}),  // 预热参数
            Cache:         parseCacheConfig(entryMap["cache"].(map[string]interface{})),  // 缓存配置
        }

        // ── 步骤 6：检查 preload_params 是否有值 ──
        // 如果 preload_params 为空对象 {}，说明该类型不需要预热，跳过
        if len(entry.PreloadParams) == 0 {
            continue
        }

        // ── 步骤 7：用 preload_params 调 service ──
        // preload_params 包含 service 字段，直接传给 ResultInner
        // 例如：{service: "system.get_bcs_code", standard_code_list: ["PIX0021","PIX0022"]}
        preloadParam := utils.Copy(entry.PreloadParams).(map[string]interface{})
        serviceResult := ts.ResultInner(preloadParam)
        // 检查服务调用是否成功
        if !serviceResult.Success {
            continue
        }

        // ── 步骤 8：遍历返回数据，逐行写入缓存 ──
        // serviceResult.GetData() 返回 []map[string]interface{}
        // 每个元素是一行码表数据，如 {STANDARD_CODE:"PIX0021", ITEM_VALUE:"1", ITEM_NAME:"普通号"}
        dataList := serviceResult.GetData().([]map[string]interface{})
        for _, row := range dataList {
            // ── 步骤 8a：用模板渲染构造缓存 key ──
            // preload_key 模板："[STANDARD_CODE#key_field]"
            // 渲染规则：
            //   [STANDARD_CODE] → row["STANDARD_CODE"] → "PIX0021"
            //   [key_field] → row[entry.KeyField] → row["ITEM_VALUE"] → "1"
            // 渲染结果："[PIX0021#1]"

            // 创建渲染参数，包含 row 的所有字段
            renderParams := make(map[string]interface{})
            for k, v := range row {
                renderParams[k] = v
            }
            // 额外添加 key_field，指向 entry.KeyField 对应的值
            renderParams["key_field"] = row[entry.KeyField]

            // 用模板渲染缓存 key
            // entry.Cache.PreloadKey = "[STANDARD_CODE#key_field]"
            // renderParams = {STANDARD_CODE: "PIX0021", ITEM_VALUE: "1", key_field: "1", ...}
            // 渲染结果 = "[PIX0021#1]"
            cacheKey := utils.RenderVar(entry.Cache.PreloadKey, renderParams).(string)

            // 去掉方括号："[PIX0021#1]" → "PIX0021#1"
            cacheKey = strings.Trim(cacheKey, "[]")

            // 拼接完整缓存 key：room + "[" + key + "]"
            // entry.Cache.Room = "code"
            // 最终 cacheKey = "code[PIX0021#1]"
            cacheKey = entry.Cache.Room + "[" + cacheKey + "]"

            // ── 步骤 8b：取翻译文本 ──
            // entry.TextField = "ITEM_NAME"
            // row["ITEM_NAME"] = "普通号"
            text := row[entry.TextField].(string)

            // ── 步骤 8c：写入缓存 ──
            // 缓存值是 common.Result 对象，不是原始字符串
            // entry.Cache.Seconds = 3600（缓存时长）
            cacheResult := common.Ok(text, "翻译缓存")
            ch.Set(cacheKey, *cacheResult, entry.Cache.Seconds)
        }
        // 等待所有缓存写入完成
        ch.Wait()
    }

    return common.Ok(nil, "预热完成")
}
```

        // 8. 用 preload_params 调 service（service2field 模式）
        //    preload_params 里有 "service" 字段，直接传给 ResultInner
        preloadParam := utils.Copy(entry.PreloadParams).(map[string]interface{})
        // preloadParam = {service: "system.get_bcs_code", standard_code_list: ["PIX0021","PIX0022"]}
        serviceResult := ts.ResultInner(preloadParam)
        if !serviceResult.Success {
            continue
        }

        // 9. 遍历返回数据，逐行写入缓存
        dataList := serviceResult.GetData().([]map[string]interface{})
        for _, row := range dataList {
            // 用 preload_key 模板构造缓存 key
            // preload_key 示例："STANDARD_CODE ITEM_VALUE"（空格分隔字段名）
            preloadFields := strings.Split(entry.Cache.PreloadKey, " ")
            cacheKey := ch.GetCacheKey(entry.Cache.Room, preloadFields, row)
            // → "code[PIX0021:1]"

            // 取 text_field 的值作为缓存 value
            text := row[entry.TextField].(string)

            // 存入缓存（存 Result 对象）
            cacheResult := common.Ok(text, "翻译缓存")
            ch.Set(cacheKey, *cacheResult, entry.Cache.Seconds)
        }
        ch.Wait()
    }

    return common.Ok(nil, "预热完成")
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// 取方法：有 type + fields
// 流程：获取注册表 → 遍历 fields → 遍历每行 → 查缓存 → 命中返回 → 未命中调 service → 写缓存
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
func (t *Translate) translate(template *config.Template,
    handlerParam *config.HandlerParam, ch cacheHandler.CacheHandler,
    ts *templateService.TemplateService) *common.Result {

    // ── 步骤 1：获取注册表配置 ──
    // getRegistry 会从 application.properties 读取 getKey
    // 拼接服务名 "system." + getKey，调用服务获取注册表
    // 服务自带 cache: 配置，框架自动处理缓存
    entry := t.getRegistry(handlerParam.Type, ts)
    if entry == nil {
        return common.NotOk("翻译类型 " + handlerParam.Type + " 未注册")
    }

    // ── 步骤 2：获取要翻译的数据列表 ──
    // handlerParam.Foreach 是数据源模板，默认 "[result]"
    // utils.RenderVar 会渲染模板，从 params 中取出数据列表
    foreach := handlerParam.Foreach  // 默认 "[result]"
    params := template.GetParams()   // 获取当前请求的所有参数
    dataList := utils.RenderVar(foreach, params).([]map[string]interface{})
    // dataList = [{reg_fee_id:"1", outp_type_code:"1", ...}, ...]

    // ── 步骤 3：遍历 fields，逐个翻译 ──
    // handlerParam.Fields 是业务配置的翻译字段列表
    // 例如：[{from:"PIX0021", field:"[item.outp_type_code]", to:"outp_type_code_str"}]
    for _, field := range handlerParam.Fields {
        // 从 field 配置中取值
        from := field.From        // BCS 码表 STANDARD_CODE，如 "PIX0021"
        fieldExpr := field.Field  // 取值模板，如 "[item.outp_type_code]"
        to := field.To            // 翻译结果写入的字段名，如 "outp_type_code_str"

        // ── 步骤 4：遍历每行数据 ──
        for _, item := range dataList {
            // ── 步骤 4a：取编码值 ──
            // 渲染 fieldExpr 模板，从 item 中取出编码值
            // fieldExpr = "[item.outp_type_code]"
            // item = {reg_fee_id:"1", outp_type_code:"1", ...}
            // 渲染结果 = "1"
            itemValue := utils.RenderVar(fieldExpr, map[string]interface{}{"item": item}).(string)
            if itemValue == "" {
                continue  // 编码值为空，跳过
            }

            // ── 步骤 4b：构造缓存 key ──
            // cache.key = "[from#key_field]"
            // 渲染规则：
            //   [from] → "PIX0021"（从业务配置 fields[].from）
            //   [key_field] → "1"（从每行数据的编码值）
            // 渲染结果："[PIX0021#1]"
            renderParams := map[string]interface{}{
                "from":      from,       // "PIX0021"
                "key_field": itemValue,  // "1"
            }
            cacheKey := utils.RenderVar(entry.Cache.Key, renderParams).(string)
            // cacheKey = "[PIX0021#1]"

            // 去掉方括号："[PIX0021#1]" → "PIX0021#1"
            cacheKey = strings.Trim(cacheKey, "[]")

            // 拼接完整缓存 key：room + "[" + key + "]"
            // entry.Cache.Room = "code"
            // 最终 cacheKey = "code[PIX0021#1]"
            cacheKey = entry.Cache.Room + "[" + cacheKey + "]"

            // ── 步骤 4c：查缓存（取） ──
            cacheResult, ok := ch.Get(cacheKey)
            if ok {
                // ── 步骤 4c-i：缓存命中 ──
                // cacheResult 是 common.Result 对象
                // cacheResult.Data 是翻译文本，如 "普通号"
                result := cacheResult.(common.Result)
                item[to] = result.Data.(string)  // 写入翻译字段
                continue
            }

            // ── 步骤 4c-ii：缓存未命中 → 调 service 查一个 ──
            // service2field 模式：构造参数调用 BCS 码表服务
            serviceParam := map[string]interface{}{
                "standard_code": from,  // BCS STANDARD_CODE，如 "PIX0021"
            }
            serviceResult := ts.ResultInner(serviceParam)
            if !serviceResult.Success {
                continue  // 服务调用失败，跳过
            }

            // ── 步骤 4c-iii：从结果中找到匹配行 ──
            // serviceResult.Data 是 []map[string]interface{}
            // 每个元素是一行码表数据，如 {ITEM_VALUE:"1", ITEM_NAME:"普通号"}
            rows := serviceResult.GetData().([]map[string]interface{})
            for _, row := range rows {
                // 用 entry.KeyField 匹配编码值
                // entry.KeyField = "ITEM_VALUE"
                // row["ITEM_VALUE"] = "1"
                // itemValue = "1"
                // 匹配成功
                if row[entry.KeyField] == itemValue {
                    // 取翻译文本
                    // entry.TextField = "ITEM_NAME"
                    // row["ITEM_NAME"] = "普通号"
                    text := row[entry.TextField].(string)

                    // 写入翻译字段
                    item[to] = text

            // ── 步骤 4c-iv：写入缓存（存） ──
            // 缓存值是 common.Result 对象
            cacheResult := common.Ok(text, "翻译缓存")
            ch.Set(cacheKey, *cacheResult, entry.Cache.Seconds)
            ch.Wait()
                    break
                }
            }
        }
    }

    return common.Ok(nil, "翻译完成")
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// getRegistry 获取注册表配置
// 流程：读配置 → 拼接服务名 → 调服务 → 找到匹配的 type → 返回 RegistryEntry
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
func (t *Translate) getRegistry(typeName string, ts *templateService.TemplateService) *RegistryEntry {

    // ── 步骤 1：从 application.properties 读取完整服务名 ──
    // 配置文件 conf/application.properties：
    // translate_registry_service=system.get_translate_registry
    serviceName := collect.GetAppKey("translate_registry_service")
    // serviceName = "system.get_translate_registry"

    // ── 步骤 3：调用服务 ──
    // service2field 模式：只需要传 "service" 字段
    // 服务自带 cache: 配置，框架自动处理缓存
    serviceParam := map[string]interface{}{
        "service": serviceName,
    }
    result := ts.ResultInner(serviceParam)
    if !result.Success {
        return nil
    }

    // ── 步骤 4：解析注册表条目，找到匹配的 type ──
    // result.Data 是 []map[string]interface{}
    // 每个元素是一个注册表条目
    registryData := result.GetData().([]map[string]interface{})
    for _, entryMap := range registryData {
        // 用 typeName 匹配条目的 type 字段
        // typeName = "code"（来自 handlerParam.Type）
        // entryMap["type"] = "code"
        if entryMap["type"].(string) == typeName {
            return &RegistryEntry{
                Type:          entryMap["type"].(string),          // 翻译类型
                Service:       entryMap["service"].(string),       // 调用的服务名
                KeyField:      entryMap["key_field"].(string),     // 匹配编码的字段名
                TextField:     entryMap["text_field"].(string),     // 取翻译文本的字段名
                PreloadParams: entryMap["preload_params"].(map[string]interface{}),  // 预热参数
                Cache:         parseCacheConfig(entryMap["cache"].(map[string]interface{})),  // 缓存配置
            }
        }
    }
    return nil  // 未找到匹配的 type
}
```

**service 调用方式：**

```
// 内部调用，非 HTTP
serviceParam := map[string]interface{}{
    "standard_code": "PIX0021",
}
result := ts.ResultInner(serviceParam)
// → 执行 system.get_bcs_code 服务
// → 返回 [{ITEM_VALUE:"1", ITEM_NAME:"普通号"}, ...]
```

**执行流程图：**

```
translate handler 收到请求
        │
        ├─ 无 type、无 fields → 预热模式
        │   │
        │   ├─ 读 application.properties → serviceName = "system.get_translate_registry"
        │   ├─ ts.ResultInner({service: serviceName})
        │   │   → 框架自动查缓存 → 命中直接返回 → 未命中执行服务
        │   ├─ 遍历每个条目
        │   │   └─ 有 preload_params → ts.ResultInner(preload_params)
        │   │       └─ 遍历返回数据 → 渲染 "[STANDARD_CODE#key_field]" → 写缓存
        │   └─ 返回"预热完成"
        │
        └─ 有 type + fields → 取模式（翻译）
            │
            ├─ t.getRegistry(type, ts)
            │   → ts.ResultInner({service: "system." + getKey})
            │   → 框架自动查缓存 → 返回注册表
            │   → 找到匹配 type 的 entry
            │
            ├─ 遍历 fields
            │   └─ 遍历每行数据
            │       ├─ 取编码值: RenderVar("[item.outp_type_code]", {item}) → "1"
            │       ├─ 渲染缓存 key: "[from#key_field]" → "[PIX0021#1]"
            │       │   → "code[PIX0021#1]"
            │       ├─ 查缓存 ch.Get("code[PIX0021#1]")
            │       │   → 命中: Result.Data → "普通号" → 填翻译字段
            │       └─ 未命中 → ts.ResultInner({service:"system.get_bcs_code", standard_code:"PIX0021"})
            │                  → 遍历找 row["ITEM_VALUE"]=="1"
            │                  → 取 row["ITEM_NAME"] → "普通号"
            │                  → 填翻译字段 + ch.Set("code[PIX0021#1]", Result("普通号"))
            │
            └─ 返回翻译后的数据
```
translate handler 收到请求
        │
        ├─ 无 type、无 fields → 预热模式
        │   │
        │   ├─ 读 application.properties → serviceName = "system.get_translate_registry"
        │   ├─ ts.ResultInner({service: serviceName})
        │   │   → 框架自动查缓存 → 命中直接返回 → 未命中执行服务
        │   ├─ 遍历每个条目
        │   │   └─ 有 preload_params → ts.ResultInner(preload_params)
        │   │       └─ 遍历返回数据 → GetCacheKey() → ch.Set() 写码表缓存
        │   └─ 返回"预热完成"
        │
        └─ 有 type + fields → 取模式（翻译）
            │
            ├─ t.getRegistry(type, ts)
            │   → ts.ResultInner({service: "system.get_translate_registry"})
            │   → 框架自动查缓存 → 返回注册表
            │   → 找到匹配 type 的 entry
            │
            ├─ 遍历 fields
            │   └─ 遍历每行数据
            │       ├─ 取编码值: RenderVar("[item.outp_type_code]", {item})
            │       ├─ 构造缓存 key: GetCacheKey("code", ["PIX0021","1"], {})
            │       │   → "code[PIX0021:1]"
            │       ├─ 查缓存 ch.Get("code[PIX0021:1]")
            │       │   → 命中: Result.Data → "普通号" → 填翻译字段
            │       └─ 未命中 → ts.ResultInner({service:"system.get_bcs_code", standard_code:"PIX0021"})
            │                  → 遍历找 row["ITEM_VALUE"]=="1"
            │                  → 取 row["ITEM_NAME"] → "普通号"
            │                  → 填翻译字段 + ch.Set("code[PIX0021:1]", Result("普通号"))
            │
            └─ 返回翻译后的数据
```
translate handler 收到请求
        │
        ├─ 无 type、无 fields → 预热模式
        │   │
        │   ├─ 构造 serviceParam = {service:"system.get_translate_registry"}
        │   ├─ ts.ResultInner(serviceParam) → 获取注册表全部条目
        │   ├─ 遍历每个条目
        │   │   ├─ ch.Set("translate_registry[code]", Result(entry), 86400)
        │   │   └─ 有 preload_params → ts.ResultInner(preload_params) 调 service
        │   │       └─ 遍历返回数据 → GetCacheKey() → ch.Set() 写码表缓存
        │   └─ 返回"预热完成"
        │
        └─ 有 type + fields → 取模式（翻译）
            │
            ├─ 查注册表缓存 translate_registry[code]
            │   ├─ 命中 → ch.Get() → Result.Data → entry
            │   └─ 未命中 → ts.ResultInner({service:"system.get_translate_registry"})
            │              → 写缓存 → 用 entry
            │
            ├─ 遍历 fields
            │   └─ 遍历每行数据
            │       ├─ 取编码值: RenderVar("[item.outp_type_code]", {item})
            │       ├─ 构造缓存 key: GetCacheKey("code", ["PIX0021","1"], {})
            │       │   → "code[PIX0021:1]"
            │       ├─ 查缓存 ch.Get("code[PIX0021:1]")
            │       │   → 命中: Result.Data → "普通号" → 填翻译字段
            │       └─ 未命中 → ts.ResultInner({service:"system.get_bcs_code", standard_code:"PIX0021"})
            │                  → 遍历找 row["ITEM_VALUE"]=="1"
            │                  → 取 row["ITEM_NAME"] → "普通号"
            │                  → 填翻译字段 + ch.Set("code[PIX0021:1]", Result("普通号"))
            │
            └─ 返回翻译后的数据
```

### 7.4 完整执行流程

```
┌─────────────────────────────────────────────────────────────────┐
│                     启动阶段（预热）                              │
│                                                                 │
│  translate_preload 服务自动执行：                                │
│  1. 调用 translate handler（无 type、无 fields → 预热模式）     │
│  2. 读 application.properties → serviceName = "system.get_translate_registry"│
│  3. ts.ResultInner({service: serviceName})                      │
│     → 框架自动查缓存 → 未命中执行 file2datajson → 返回注册表     │
│  4. 遍历每个条目：                                               │
│     a. 有 preload_params → ts.ResultInner(preload_params)       │
│        → preload_params 包含 service:"system.get_bcs_code"      │
│     b. 遍历返回数据 → 渲染 "[STANDARD_CODE#key_field]"          │
│        → "code[PIX0021#1]" → ch.Set() 写码表缓存               │
└─────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│                     运行阶段（取）                                │
│                                                                 │
│  业务 service 执行完毕后：                                        │
│  result_handler 调用 translate handler                          │
│  （有 type + fields → 取模式）                                   │
│                                                                 │
│  handler 流程：                                                  │
│  1. t.getRegistry(type, ts)                                     │
│     → ts.ResultInner({service: "system." + getKey})             │
│     → 框架自动查缓存 → 返回注册表 → 找到匹配 type 的 entry     │
│                                                                 │
│  2. 遍历 fields + 每行数据：                                     │
│     a. 取编码值: RenderVar("[item.outp_type_code]") → "1"       │
│     b. 渲染缓存 key: "[from#key_field]" → "[PIX0021#1]"        │
│        → "code[PIX0021#1]"                                      │
│     c. ch.Get("code[PIX0021#1]")                                │
│        ├─ 命中 → Result.Data → "普通号" → 填翻译字段             │
│        └─ 未命中 → ts.ResultInner({service:"system.get_bcs_code"│
│           , standard_code:"PIX0021"})                            │
│           → 找 row["ITEM_VALUE"]=="1" → "普通号"                │
│           → 填翻译字段 + ch.Set("code[PIX0021#1]",Result("普通号"))│
│                                                                 │
│  3. 返回翻译后的数据                                             │
└─────────────────────────────────────────────────────────────────┘
```
┌─────────────────────────────────────────────────────────────────┐
│                     启动阶段（预热）                              │
│                                                                 │
│  translate_preload 服务自动执行：                                │
│  1. 调用 translate handler（无 type、无 fields → 预热模式）     │
│  2. 读 application.properties → serviceName = "system.get_translate_registry"     │
│  3. ts.ResultInner({service: serviceName})                      │
│     → 框架自动查缓存 → 未命中执行 file2datajson → 返回注册表     │
│  4. 遍历每个条目：                                               │
│     a. 有 preload_params → ts.ResultInner(preload_params)       │
│        → preload_params 包含 service:"system.get_bcs_code"      │
│     b. 遍历返回数据 → GetCacheKey() → ch.Set() 写码表缓存       │
└─────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│                     运行阶段（取）                                │
│                                                                 │
│  业务 service 执行完毕后：                                        │
│  result_handler 调用 translate handler                          │
│  （有 type + fields → 取模式）                                   │
│                                                                 │
│  handler 流程：                                                  │
│  1. t.getRegistry(type, ts)                                     │
│     → ts.ResultInner({service: "system.get_translate_registry"})│
│     → 框架自动查缓存 → 返回注册表 → 找到匹配 type 的 entry     │
│                                                                 │
│  2. 遍历 fields + 每行数据：                                     │
│     a. 取编码值: RenderVar("[item.outp_type_code]") → "1"       │
│     b. 构造缓存 key: GetCacheKey("code",["PIX0021","1"],{})     │
│        → "code[PIX0021:1]"                                      │
│     c. ch.Get("code[PIX0021:1]")                                │
│        ├─ 命中 → Result.Data → "普通号" → 填翻译字段             │
│        └─ 未命中 → ts.ResultInner({service:"system.get_bcs_code"│
│           , standard_code:"PIX0021"})                            │
│           → 找 row["ITEM_VALUE"]=="1" → "普通号"                │
│           → 填翻译字段 + ch.Set("code[PIX0021:1]",Result("普通号"))│
│                                                                 │
│  3. 返回翻译后的数据                                             │
└─────────────────────────────────────────────────────────────────┘
```
┌─────────────────────────────────────────────────────────────────┐
│                     启动阶段（预热）                              │
│                                                                 │
│  translate_preload 服务自动执行：                                │
│  1. 调用 translate handler（无 type、无 fields → 预热模式）     │
│  2. serviceParam = {service:"system.translate_registry          │
│                     .get_translate_registry"}                   │
│  3. ts.ResultInner(serviceParam) → 获取注册表全部条目           │
│  4. 遍历每个条目：                                               │
│     a. ch.Set("translate_registry[code]", Result(entry), 86400) │
│     b. 有 preload_params → ts.ResultInner(preload_params)       │
│        → preload_params 包含 service:"system.get_bcs_code"      │
│     c. 遍历返回数据 → GetCacheKey() → ch.Set() 写码表缓存       │
└─────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────┐
│                     运行阶段（取）                                │
│                                                                 │
│  业务 service 执行完毕后：                                        │
│  result_handler 调用 translate handler                          │
│  （有 type + fields → 取模式）                                   │
│                                                                 │
│  handler 流程：                                                  │
│  1. 查注册表缓存 translate_registry[code]                       │
│     ├─ 命中 → ch.Get() → Result.Data → entry                   │
│     └─ 未命中 → serviceParam = {service:"system.translate_      │
│        registry.get_translate_registry"}                        │
│        → ts.ResultInner() → ch.Set() → 用 entry                │
│                                                                 │
│  2. 遍历 fields + 每行数据：                                     │
│     a. 取编码值: RenderVar("[item.outp_type_code]") → "1"       │
│     b. 构造缓存 key: GetCacheKey("code",["PIX0021","1"],{})     │
│        → "code[PIX0021:1]"                                      │
│     c. ch.Get("code[PIX0021:1]")                                │
│        ├─ 命中 → Result.Data → "普通号" → 填翻译字段             │
│        └─ 未命中 → serviceParam = {service:"system.get_bcs_code"│
│           , standard_code:"PIX0021"}                            │
│           → ts.ResultInner() → 找 row["ITEM_VALUE"]=="1"       │
│           → "普通号" → 填翻译字段                                │
│           → ch.Set("code[PIX0021:1]", Result("普通号"))         │
│                                                                 │
│  3. 返回翻译后的数据                                             │
└─────────────────────────────────────────────────────────────────┘
```

### 7.5 缓存 key 构造规则

**标准缓存格式（CacheHandler.GetCacheKey）：**

```go
// 格式: room[field1Value:field2Value:...]
// 分隔符: 冒号 :
// 示例: code[PIX0021:1]
func (t *CacheHandler) GetCacheKey(room string, fields []string, params map[string]interface{}) string {
    valueList := utils.GetFieldValueList(fields, params)
    valueStr := strings.Join(valueList, ":")
    return fmt.Sprintf("%s[%s]", room, valueStr)
}
```

**翻译模块缓存格式（模板渲染）：**

```go
// 模板: "[from#key_field]"
// 分隔符: 井号 #（自定义，与标准格式不同）
// 渲染后: "[PIX0021#1]"
// 最终: "code[PIX0021#1]"

renderParams := map[string]interface{}{
    "from":      "PIX0021",
    "key_field": "1",
}
cacheKey := utils.RenderVar("[from#key_field]", renderParams).(string)
// → "[PIX0021#1]"
cacheKey = strings.Trim(cacheKey, "[]")
cacheKey = "code" + "[" + cacheKey + "]"
// → "code[PIX0021#1]"
```

**两种格式对比：**

| 格式 | 分隔符 | 构造方式 | 示例 |
|------|--------|----------|------|
| 标准 | `:` | `GetCacheKey(room, fields, params)` | `code[PIX0021:1]` |
| 翻译 | `#` | 模板渲染 `"[from#key_field]"` | `code[PIX0021#1]` |

**预热与取的 key 一致性：**

```
预热时（preload_key = "[STANDARD_CODE#key_field]"):
  row = {STANDARD_CODE: "PIX0021", ITEM_VALUE: "1", ITEM_NAME: "普通号"}
  renderParams = {STANDARD_CODE: "PIX0021", key_field: "1"}
  渲染 "[STANDARD_CODE#key_field]" → "[PIX0021#1]"
  结果: "code[PIX0021#1]"

取时（key = "[from#key_field]"):
  from = "PIX0021", itemValue = "1"
  renderParams = {from: "PIX0021", key_field: "1"}
  渲染 "[from#key_field]" → "[PIX0021#1]"
  结果: "code[PIX0021#1]"

→ 两种场景写入相同的缓存 key，预热数据在运行时直接命中
```

### 7.6 注册表存储

注册表通过低代码服务（`file2datajson`）从 JSON 配置文件读取，服务自带 `cache:` 配置，框架自动处理缓存：

```
服务定义（index.yml）：
  cache:
    key: "handler_cache"
    room: translate_registry
    second: 86400
    fields:
      - field: "[service]"

框架自动处理：
  执行前：查缓存 translate_registry[system.get_translate_registry]
         → 命中直接返回（跳过 file2datajson）
         → 未命中执行服务
  执行后：结果写入缓存
```

```
启动时（预热）：
  1. 启动服务调用 translate handler（无 fields → 预热模式）
  2. translate handler 读取 application.properties → 拿到 getKey
  3. ts.ResultInner({service: "system.get_translate_registry"})
     → 框架自动查缓存 → 未命中则执行服务 → 返回注册表 JSON
  4. 遍历 registry 数组
  5. 对每个 type 条目，用 preload_params 调 BCS 码表 service
  6. 返回数据遍历每行，用 preload_key 构造缓存 key，写入 Ristretto

运行时：
  1. translate handler 执行时（有 fields → 翻译模式）
  2. t.getRegistry() 调服务 → 框架自动查缓存
  3. 拿到 service/params/key_field/text_field/cache 配置
  4. 遍历数据翻译
```

> 注册表服务的缓存由框架自动管理，handler 不需要手动缓存。

### 7.7 service_router.yml 改动详情

**data_handler 添加 translate（第 179 行后追加）：**

```yaml
# collect/service_router.yml → data_handler
  - key: translate
    name: 翻译处理器
    type: outer
    path: Translate
```

> **注意：** 预热服务不需要在 `service_router.yml` 中单独定义，它是一个启动服务（`http: false`），在应用启动时由框架自动执行。

### 7.8 实现清单（更新）

| 文件 | 类型 | 说明 |
|------|------|------|
| `docs/code_translate_design.md` | 文档 | 本文档 |
| `conf/application.properties` | 修改 | 添加 `translate_registry_service=system.get_translate_registry` |
| `collect/system/bcs_code/index.yml` | 新增 | BCS 码表查询服务定义 |
| `collect/system/bcs_code/get_bcs_code.sql` | 新增 | BCS 码表查询 SQL |
| `collect/system/service.yml` | 修改 | 添加 BCS 码表服务路由 |
| `collect/system/translate_registry/index.yml` | 新增 | 翻译注册表服务定义 |
| `collect/system/translate_registry/get_translate_registry.json` | 新增 | 注册表 JSON 数据 |
| `collect/system/service.yml` | 修改 | 添加翻译注册表路由 |
| `collect/system/translate_preload/index.yml` | 新增 | 预热启动服务（无 HTTP，自动执行） |
| `collect/service_router.yml` | 修改 | data_handler 添加 translate |
| `plugins/handler_result_translate.go` | 新增 | translate handler（data_handler），支持预热和翻译两种模式 |
| `plugins/a_register.go` | 修改 | 注册 Translate handler |
| `collect/him/pix_outp_reg_master/reg_fee_options.sql` | 修改 | 去掉 BCS JOIN |
| `collect/him/pix_outp_reg_master/outp_type_options.sql` | 修改 | 去掉 BCS JOIN |
| `collect/him/pix_outp_reg_master/index.yml` | 修改 | 添加 result_handler |


---

