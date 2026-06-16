# 邮箱批量导入需求文档

## 1. 目标

新增一个“邮箱登记/邮箱批量导入”功能，用于把类似下面的原始文本批量导入系统并管理：

```text
ayeleelhdad1995@outlook.com----fmaya5864777----9e5f94bc-e8a4-4e73-b8be-63364c29d753----M.C544_BAY.0.U.-...
hilleeniels0713@outlook.com----eqpua3222546----9e5f94bc-e8a4-4e73-b8be-63364c29d753----M.C534_BAY.0.U.-...
```

需求目标不是只做一个页面，而是同时把：

- 成品功能
- 页面接入方式
- 菜单配置方式
- 文档模板
- 测试脚本与结果目录

一起沉淀下来，后续新需求可以直接复用。

## 1.1 成品与文档联动原则

这次不是“先做完功能，最后补文档”，而是功能和文档同步推进。

要求：

- 每做出一个成品能力，都要反向沉淀到 `docs/` 标准文档
- 每补一条文档规则，都要能在本需求成品里找到对应落点
- 文档不是一次性成稿，而是随着本需求推进不断修订
- 如果成品实现和文档描述不一致，以成品真实链路为准，立即回写文档

因此本需求的关系是：

- `feature/01-mail.md`：当前需求总说明 + 当前进度 + 下个 session 接手入口
- `docs/lowcode/page/*.md`：抽象后的标准方法论
- 最终成品页面/接口/脚本：对标准文档的反向验证样例

目标是做到：

- 这次需求做完，文档也一起升级
- 下次只要提一个类似需求，就可以按模板直接推进
- 文档与成品相辅相成，而不是两套脱节内容

## 1.2 文档迭代原则

由于这套 `docs/lowcode/page/` 是第一次系统化整理，默认认为一定会有疏漏。

因此执行规则固定为：

1. 先用本需求把标准文档跑通一轮
2. 实施过程中凡是出现“文档没写到”“写得不够实”“和实际不一致”，都必须补回文档
3. 每完成一个阶段，要同时更新：
   - 本需求文档 `feature/01-mail.md`
   - 对应标准文档 `docs/lowcode/page/*.md`
4. 后续新需求开始前，优先回看本文件和 `docs/lowcode/page/README.md`

也就是说，这次需求不仅是业务实现，也是第一轮文档打磨过程。

## 2. 已确认约束

### 2.0 总原则：低代码优先

这是本需求的最高优先级原则：

- 能用低代码配置完成，就不要写 Go 代码
- 能复用已有 `module / handler / service`，就不要新增专用模块
- 能通过前端解析 + 后端配置组合完成，就不要下沉到 Go
- Go 代码只用于以下情况：
  - 框架现有能力完全做不到
  - 低代码配置会极度复杂且不可维护
  - 多个后续需求都能明确复用该代码能力

对本需求而言，当前结论已经明确：

- 不推荐写专用 `mail_import` 模块
- 不推荐为了这次简单导入额外写 Go 逻辑
- 推荐方案是：
  - 前端解析对象数组
  - 后端查询已有数据
  - `update_array_from_array` 标记
  - `filter_arr` 过滤
  - `bulk_create` 批量导入

后续如果下个 session 想新增 Go 代码，必须先回答两个问题：

1. 现有低代码 handler 是否真的无法完成？
2. 这个 Go 能力是否能被后续多个需求复用？

如果两个问题都答不出来，则默认不允许写 Go。

### 2.1 菜单来源

运行时菜单来自数据库表 `sys_menu`，不是静态 `collect/frontend/page_data/data/menu/menu.json`。

真实链路：

- 菜单数据：`sys_menu`
- 权限：`role_menu`
- 页面映射：`collect/frontend/page_data/index.yml`
- 页面 JSON：`collect/frontend/page_data/data/**/*.json`
- 页面菜单查询接口：`system.menu_query`

### 2.2 导入实现方式

用户明确要求：

- 尽量用低代码方式配置
- 不新增专用 `mail_import` Go 模块
- 前端先解析成固定对象数组
- 后端再查询现有数据，对比、标记、过滤后批量导入

因此本需求采用：

- 前端解析
- 后端低代码 `empty + service2field + update_array + update_array_from_array + filter_arr + bulk_create`

不新增自定义导入模块。

这里再强调一次：

- 这个需求是“低代码能力复用案例”
- 不是“补一个 mail_import Go 模块”
- 文档也必须把这种取舍写清楚，作为以后类似需求的默认做法

### 2.3 重复策略

已确认策略：

- 若邮箱已存在数据库中：跳过，不报整体失败
- 同一批导入文本内部如果有重复邮箱：前端解析阶段就标记并排除

### 2.4 页面范围

V1 范围固定为：

- 批量导入
- 列表查看
- 搜索
- 删除
- 单条设置：
  - 当前运行标记
  - Proton 注册状态
  - Proton 邮箱/密码

暂不做：

- 单条新增
- 复杂字段编辑

## 2.5 验收环境与访问地址

本需求联调和无头浏览器验收，默认使用以下地址：

- 页面访问基地址：`http://192.168.232.130:8015/`
- 接口访问基地址：`http://192.168.232.130:8015/template_data/data`

说明：

- 页面验收不以本地 `localhost` 为唯一标准
- 无头浏览器测试脚本默认访问上述 IP 地址
- 后台接口测试脚本默认对上述接口地址发起请求

后续如果切换环境，必须在本文件和测试脚本中同步更新。

## 3. 业务规则

### 3.1 输入格式

每一行文本按 `----` 分段，至少 4 段：

1. 邮箱
2. 密码
3. guid / uuid
4. 恢复码长串

如果第 4 段之后还有额外 `----`，统一并入恢复码字段。

### 3.2 字段映射

表字段设计：

- `mail_account_id`：主键
- `order_index`：导入顺序
- `email_name`：邮箱
- `password`：密码
- `guid_code`：第三段 GUID
- `recovery_code`：第四段及之后拼接结果
- `raw_text`：原始整行
- `create_time`
- `create_user`
- `is_delete`

### 3.3 前端解析结果

前端把原始文本解析成固定数组对象：

```json
[
  {
    "order_index": 1,
    "email_name": "ayeleelhdad1995@outlook.com",
    "password": "fmaya5864777",
    "guid_code": "9e5f94bc-e8a4-4e73-b8be-63364c29d753",
    "recovery_code": "M.C544_BAY.0.U.-...",
    "raw_text": "ayeleelhdad1995@outlook.com----..."
  }
]
```

同时前端还需要生成：

- `valid_list`
- `error_list`
- `duplicate_in_batch_list`
- `email_name_list_sql`

其中：

- `email_name_list_sql` 供后端 SQL `in (...)` 查询已存在邮箱
- 格式错误和批内重复的数据不进入 `valid_list`

## 3.4 接口访问格式

本项目标准接口入口为：

```text
/template_data/data
```

常见调用方式有 2 种，文档中都要体现：

### 方式一：query 参数指定 service

前端页面里最常见的写法：

```text
post:/template_data/data?service=project.server_os_users_query
```

示例：

```json
{
  "server_id": "03c1c48c-cdda-4eef-b059-fa163b5cf2ab"
}
```

### 方式二：请求体显式带 service

后台接口测试或脚本里常用写法：

```json
{
  "service": "project.server_os_users_query",
  "server_id": "03c1c48c-cdda-4eef-b059-fa163b5cf2ab"
}
```

### 方式三：两者同时保留

为了和前端实际调用形式保持一致，也允许写成：

```text
POST http://192.168.232.130:8015/template_data/data?service=project.server_os_users_query
```

请求体：

```json
{
  "service": "project.server_os_users_query",
  "server_id": "03c1c48c-cdda-4eef-b059-fa163b5cf2ab"
}
```

这个格式必须写进测试文档与测试脚本说明，因为它最贴近现网前端联调方式。

### 3.5 本需求接口格式约定

本需求建议统一采用：

```text
POST http://192.168.232.130:8015/template_data/data?service=system.mail_account_import_batch
```

请求体示例：

```json
{
  "service": "system.mail_account_import_batch",
  "mail_account_list": [
    {
      "order_index": 1,
      "email_name": "ayeleelhdad1995@outlook.com",
      "password": "fmaya5864777",
      "guid_code": "9e5f94bc-e8a4-4e73-b8be-63364c29d753",
      "recovery_code": "M.C544_BAY.0.U.-...",
      "raw_text": "ayeleelhdad1995@outlook.com----fmaya5864777----9e5f94bc-e8a4-4e73-b8be-63364c29d753----M.C544_BAY.0.U.-..."
    }
  ]
}
```

查询接口建议采用：

```text
POST http://192.168.232.130:8015/template_data/data?service=system.mail_account_query
```

请求体示例：

```json
{
  "service": "system.mail_account_query",
  "search": "outlook.com",
  "page": 1,
  "size": 20
}
```

## 4. V1 成品范围

### 4.1 页面

菜单名建议：

- `邮箱登记`

菜单编码建议：

- `mail_account`

建议路由：

- `/framework/mail_account`

建议前端 service：

- `frontend.mail_account`

建议页面文件：

- `collect/frontend/page_data/data/system/mail_account.json`

页面结构：

1. 顶部查询栏
2. 中部批量导入区域
3. 解析结果提示区
4. 下部列表表格

### 4.2 后端

建议后端 service 放在：

- `collect/system/mail_account/index.yml`

建议后端 service：

- `system.mail_account_query`
- `system.mail_account_bulk_create`
- `system.mail_account_import_batch`
- `system.mail_account_delete`

### 4.3 模型

建议模型放在：

- `model/base/mail_account.go`

并注册到：

- `model/base/add_table.go`

## 5. 低代码实现原则

本需求要求最大化复用低代码能力，优先使用：

- `service2field`
- `update_array`
- `update_array_from_array`
- `filter_arr`
- `prop_arr`
- `params2result`
- `bulk_create`

不推荐做法：

- 为本需求新增专用 Go `module`
- 为本需求新增一次性 Go `handler`
- 用 Go 手写导入、查重、分流，而绕开现有低代码配置能力

推荐做法：

- 前端把原始字符串清洗成固定对象数组
- 后端通过低代码服务组合完成：
  - 查询已有数据
  - 对比
  - 标记
  - 分流
  - 批量导入
  - 汇总返回结果

推荐后端导入链路：

1. 接收前端 `mail_account_list`
2. 根据 `email_name_list_sql` 查询现有邮箱
3. 对 `mail_account_list` 先统一打 `exists=0`
4. 用 `update_array_from_array` 按 `email_name` 匹配现有数据，命中则改成 `exists=1`
5. 用 `filter_arr` 拆成：
   - `skip_list`
   - `create_list`
6. 对 `create_list` 补：
   - `mail_account_id`
   - `create_time`
   - `create_user`
   - `is_delete=0`
7. 用 `bulk_create` 入库
8. 返回导入摘要

关键原则：

- 不写专用导入模块
- 不把简单导入逻辑下沉到 Go
- 低代码配置要反向沉淀到文档里，作为以后复用模板
- 以后只要是“文本解析 -> 对象数组 -> 查重 -> 批量入库”这类需求，优先复用这套低代码范式

## 6. 文档映射

这些文档不是并列孤立文件，而是互相补位。

联动关系如下：

- `00-最小接入模版demo.md`
  - 负责最低门槛接入
  - 回答“怎么最快新增一个页面”
- `01-运行时菜单与页面链路.md`
  - 负责真实链路认知
  - 回答“菜单和页面到底从哪来”
- `02-需求分析与组件选型流程.md`
  - 负责开始做之前的分析方法
  - 回答“该用哪些组件、怎么拆需求”
- `03-新增菜单与页面标准步骤.md`
  - 负责实施顺序
  - 回答“具体先改什么、后改什么”
- `04-邮箱登记页面实战.md`
  - 负责把本需求作为完整例子跑通
  - 回答“标准文档怎样落到真实需求”
- `05-联调与测试标准.md`
  - 负责验收闭环
  - 回答“如何确认真的做完”

后续优化原则：

- 如果本需求里某一步难以执行，优先判断是落地问题还是文档问题
- 如果是文档问题，先改文档再继续实现
- 标准文档必须随着本需求成品一起变好
- 文档里必须明确写出“低代码优先”的判断标准，防止后续 session 默认走写 Go 模块的路线

本需求将同步沉淀到以下文档：

### 6.1 最小模板

- `docs/lowcode/page/00-最小接入模版demo.md`

作用：

- 讲清楚新增一个最小页面需要哪些最少改动
- 作为以后所有页面需求的起手模板

### 6.2 运行时链路

- `docs/lowcode/page/01-运行时菜单与页面链路.md`

作用：

- 说明菜单来自 `sys_menu`
- 页面来自 `frontend.page_data`
- 防止以后再误判静态 `menu.json`

### 6.3 需求分析与组件选型

- `docs/lowcode/page/02-需求分析与组件选型流程.md`

作用：

- 先分析需求能力，再选组件
- 强制先查组件文档目录：
  - `/data/project/collect-ui/docs/readme/components/`

### 6.4 标准步骤

- `docs/lowcode/page/03-新增菜单与页面标准步骤.md`

作用：

- 讲清楚菜单、页面、service、模型、seed SQL、测试如何接入

### 6.5 本需求实战

- `docs/lowcode/page/04-邮箱登记页面实战.md`

作用：

- 用本需求完整演示
- 以后做类似“文本解析 -> 批量导入”页面时直接照抄

### 6.6 联调与测试

- `docs/lowcode/page/05-联调与测试标准.md`

作用：

- 统一后台接口测试
- 统一前台无头浏览器测试
- 统一结果产物目录

## 7. 组件选型映射

本需求对应组件如下：

- 页面容器：`layout-fit`
- 查询栏：`form`、`form-item`、`input`
- 多行导入：`input` + `isTextarea: true`
- 主按钮：`button`
- 数据展示：`table`
- 删除确认：`confirm`

组件文档应优先参考：

- `/data/project/collect-ui/docs/readme/components/layout-fit.md`
- `/data/project/collect-ui/docs/readme/components/form.md`
- `/data/project/collect-ui/docs/readme/components/form-item.md`
- `/data/project/collect-ui/docs/readme/components/input.md`
- `/data/project/collect-ui/docs/readme/components/button.md`
- `/data/project/collect-ui/docs/readme/components/table.md`
- `/data/project/collect-ui/docs/readme/components/confirm.md`

## 8. 菜单接入策略

注意：当前菜单在数据库里，不在仓库源码里。

因此本需求菜单接入策略为：

- 不直接修改已跟踪且已脏的 `database/price.db`
- 生成 seed SQL 和文档说明
- 需要时由脚本或人工执行 seed

建议挂在当前 `base` 项目的系统设置分组下：

- 父菜单：`7c8b9620-db64-4586-97f6-a715c6d477b7`
- 父菜单名称：`系统设置`

建议菜单字段：

- `menu_type=2`
- `menu_name=邮箱登记`
- `menu_code=mail_account`
- `url=/framework/mail_account`
- `api=post:/template_data/data?service=frontend.mail_account`
- `router_group=framework`
- `in_menu=1`
- `is_common=1`

## 9. 测试与结果目录规划

测试目录统一规划为：

- `test/lowcode-page/scripts/backend/`
- `test/lowcode-page/scripts/frontend/`
- `test/lowcode-page/results/latest/`
- `test/lowcode-page/results/history/mail_account/<timestamp>/`

建议脚本：

- `apply_mail_account_seed.sh`
- `backend/mail_account_api_check.sh`
- `frontend/mail_account_page_check.js`
- `run_mail_account_check.sh`

结果产物：

- `summary.md`
- `backend.log`
- `frontend.log`
- `api-response.json`
- `page-after-import.png`
- `console-errors.log`
- `checklist.md`

### 9.1 无头浏览器测试验收地址

无头浏览器脚本默认访问：

```text
http://192.168.232.130:8015/
```

如果页面菜单已经接入成功，则脚本需要直接访问真实可达路由，例如：

```text
http://192.168.232.130:8015/collect-ui#/collect-ui/framework/mail_account
```

测试脚本中要把以下内容写清楚：

- 页面基地址
- 路由地址
- 菜单点击路径
- 页面关键文本
- 导入前后表格差异

### 9.2 后台接口测试格式示例

测试文档和脚本必须体现本项目接口访问格式。

参考示例：

请求地址：

```text
http://192.168.232.130:8015/template_data/data?service=project.server_os_users_query
```

请求体：

```json
{
  "service": "project.server_os_users_query",
  "server_id": "03c1c48c-cdda-4eef-b059-fa163b5cf2ab"
}
```

因此本需求脚本也按相同结构组织：

- URL 用 `?service=...`
- Body 中保留 `service`
- 业务参数放在同级 JSON

这样和页面联调方式一致，也方便后续排查。

## 10. 启停规则

### 10.1 生效规则

以下改动后都按“先停再启”处理：

- Go 代码
- 模型注册
- `collect/**/*.yml`

### 10.2 命令

停止：

```bash
./shutdown.sh
```

启动：

```bash
./linux-start-dev.sh
```

### 10.3 注意事项

- 不能只重复执行 `./linux-start-dev.sh`
- 该脚本发现旧 PID 仍在时会直接退出
- 所以正确顺序必须是：
  1. `./shutdown.sh`
  2. `./linux-start-dev.sh`
  3. 检查 `run-dev.log`

## 11. 当前状态

本轮已完成的事项：

- 已新增 `mail_account` 模型并注册
- 已新增 `collect/system/mail_account/` 后端 service 与 SQL
- 已新增 `frontend.mail_account` 页面映射与页面 JSON
- 已补齐 seed SQL 与测试脚本骨架
- 已补齐 `docs/lowcode/page/` 标准文档首版
- 已在真实环境库执行 seed，确认表结构、菜单、权限已落库
- 已完成后台接口联调与删除回归
- 已完成前台页面无头验收

本轮联调确认的真实运行信息：

- 后台接口基地址：`http://192.168.232.130:8015/template_data/data`
- 页面真实可达地址：`http://192.168.232.130:8015/collect-ui#/collect-ui/framework/mail_account`
- 根地址 `http://192.168.232.130:8015/#/framework/mail_account` 在当前运行时会被默认页重定向到 `webshell`
- 菜单按公共菜单处理：`is_common=1`，不写 `role_menu` 授权

## 12. 运行日志

### 2026-04-14 / Session 01

1. 核对项目菜单来源，确认运行时菜单来自 `sys_menu`，不是静态 `menu.json`。
2. 核对 `system.menu_query`、`sys_menu`、`role_menu`、`frontend.page_data` 的真实链路。
3. 梳理“邮箱导入”需求，确认页面目标不是单纯录入，而是成品 + 文档 + 测试一起沉淀。
4. 初步方案曾考虑写专用导入模块。
5. 用户明确要求最大限度用低代码配置，不新增 `mail_import` 模块。
6. 重新收敛方案为：
   - 前端解析对象数组
   - 后端查询现有邮箱
   - `update_array_from_array` 标记是否已存在
   - `filter_arr` 拆分新增/跳过
   - `bulk_create` 批量导入
7. 确认输入解析规则：
   - 第 1 段邮箱
   - 第 2 段密码
   - 第 3 段 guid
   - 第 4 段及之后为恢复码
8. 确认重复策略：
   - 库内重复：跳过
   - 同批重复：前端解析阶段剔除
9. 确认页面范围：
   - 批量导入 + 列表 + 删除
10. 创建目录骨架：
   - `docs/lowcode/page/`
   - `test/lowcode-page/scripts/backend/`
   - `test/lowcode-page/scripts/frontend/`
   - `test/lowcode-page/results/latest/`
   - `collect/system/mail_account/`
   - `feature/`
11. 补充要求：
   - 文档之间要体现相辅相成，不是平铺目录说明
   - 成品功能与标准文档要双向优化
   - 无头浏览器验收地址使用 `http://192.168.232.130:8015/`
   - 接口访问格式要在文档中明确体现，特别是：
     - `/template_data/data?service=xxx`
     - body 中保留 `service`
     - 业务参数与 `service` 同级传递

### 2026-04-14 / Session 02

1. 新增 `model/base/mail_account.go`，并注册到 `model/base/add_table.go`。
2. 新增 `collect/system/mail_account/index.yml`、`base.sql`、`query.sql`、`count.sql`。
3. 挂载 `system.mail_account_query`、`system.mail_account_bulk_create`、`system.mail_account_import_batch`、`system.mail_account_delete`。
4. 新增 `frontend.mail_account` 页面映射。
5. 新增 `collect/frontend/page_data/data/system/mail_account.json` 页面，实现：
   - 前端原始文本解析
   - 有效/错误/重复预览
   - 导入摘要展示
   - 列表查询、分页、删除
6. 新增 seed SQL：
   - `test/lowcode-page/scripts/sql/mail_account_seed.sql`
7. 新增脚本：
   - `test/lowcode-page/scripts/apply_mail_account_seed.sh`
   - `test/lowcode-page/scripts/backend/mail_account_api_check.sh`
   - `test/lowcode-page/scripts/frontend/mail_account_page_check.js`
   - `test/lowcode-page/scripts/run_mail_account_check.sh`
8. 补齐 `docs/lowcode/page/README.md` 与 `00` 到 `05` 文档。
12. 再次确认实现策略：
   - 尽可能用低代码
   - 我之前偏向写 Go 模块的思路不推荐
   - 本需求默认不新增专用导入 Go 代码
   - 能配置就配置，优先把能力沉淀成低代码模板

### 2026-04-15 / Session 03

1. 在真实库 `database/price.db` 上执行 seed，确认 `mail_account` 表、索引、菜单已生效。
2. 确认可用联调账号：
   - 用户名：`001`
   - 密码：`A123456!`
   - 角色：`admin`
3. 在真实服务上完成接口联调：
   - `system.mail_account_query`
   - `system.mail_account_import_batch`
   - `system.mail_account_delete`
4. 修正联调暴露的两个真实问题：
   - `guid_code` 字段映射修正为 `GUIDCode`
   - 导入前查重改为基于 `email_name_list` 走 SQL `in (...)`
5. 重新执行后台验收脚本，确认：
   - 首次导入 `create_count=2`
   - 第二次导入 `skip_count=2`
   - 查询能看到明文字段
   - 删除后记录消失
6. 安装 Playwright 与 Chromium，补齐真实无头浏览器验收环境。
7. 确认页面脚本默认地址原先写错；真实可达地址应为：
   - `http://192.168.232.130:8015/collect-ui#/collect-ui/framework/mail_account`
8. 已将真实地址回写到验收脚本与标准文档，避免后续 session 再走到默认 `webshell` 页面。
9. 根据最新要求，菜单改为公共菜单：
   - 保留 `is_common=1`
   - 不再写任何 `role_menu` 授权
   - seed 会先删除旧授权，再只保留菜单本身
10. 页面无头验收已输出：
   - 截图
   - `api-response.json`
   - `console-errors.log`
   - `summary.md`

### 2026-04-15 / Session 04

1. 根据最新使用反馈，页面布局调整为“邮箱列表优先”：
   - 批量导入改为通过弹框打开
   - 首屏优先展示搜索栏和邮箱表格
2. 页面右上角新增：
   - `批量导入` 按钮
3. 批量导入表单改为对话框，不再占用页面主体高度。
4. 原先占页面主体的“解析与导入摘要”改为对话框查看。
5. 摘要明细在对话框内使用 tabs 展示，减少页面纵向长度。
6. 批量导入输入框高度下调，避免打开导入区后仍然过度挤压列表。
7. 同步更新前台无头脚本，适配“先点批量导入，再解析/导入”的新交互。
8. 解析与导入摘要对话框取消 `tab` 切换，改为普通分组卡片，避免标题变量替换异常并减少视线跳转。
9. 列表新增“当前运行”高亮标记与排序规则：
   - 可将任一邮箱设为当前运行
   - 当前行显示高亮小图标
   - 查询排序按“当前运行序号及其后续记录靠后”处理
10. 新增单条“邮箱设置”对话框：
   - 可设置是否为当前运行
   - 可设置是否已注册 Proton
   - 可填写 Proton 邮箱与密码
   - 默认 Proton 邮箱按当前邮箱前缀拼接 `@proton.me`
   - 默认 Proton 密码为 `Zhangzhi@888`
11. 真实库 `mail_account` 已补列并回填默认值：
   - `is_current_running`
   - `current_run_mark_time`
   - `proton_registered`
   - `proton_email`
   - `proton_password`
12. 已完成真实环境回归：
   - `go test ./...` 通过
   - 后台脚本已验证导入、重复跳过、当前运行设置、Proton 设置、删除
   - 前台无头脚本已验证导入、列表渲染、删除链路正常

## 13. 下一步执行清单

如需下个 session 复验，直接按以下顺序执行：

1. 执行 seed：
   - `test/lowcode-page/scripts/apply_mail_account_seed.sh`
2. 启动服务：
   - 推荐直接前台运行 `go run main.go`
   - 若使用脚本，先 `./shutdown.sh` 再 `./linux-start-dev.sh`
3. 执行后台验收：
   - `MAIL_ACCOUNT_USERNAME=001 MAIL_ACCOUNT_PASSWORD='A123456!' test/lowcode-page/scripts/backend/mail_account_api_check.sh`
4. 执行前台验收：
   - `MAIL_ACCOUNT_USERNAME=001 MAIL_ACCOUNT_PASSWORD='A123456!' test/lowcode-page/scripts/frontend/mail_account_page_check.js`
5. 如需一键跑：
   - `MAIL_ACCOUNT_USERNAME=001 MAIL_ACCOUNT_PASSWORD='A123456!' test/lowcode-page/scripts/run_mail_account_check.sh`

## 14. 本文用途

这份文档是：

- 当前需求说明
- 当前 session 进度记录
- 下个 session 的接手入口
- 后续 `docs/lowcode/page/` 标准文档的源需求文档

后续如果继续推进本需求，优先更新本文件的：

- 当前状态
- 运行日志
- 下一步执行清单

这样只看 `feature/01-mail.md` 就能知道：

- 需求是什么
- 已经决定了什么
- 当前做到哪
- 下一步该做什么
