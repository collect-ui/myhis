# HTTP兼容代理改造方案

## 目标

在 `webshell-editor-pool` 的 HTTP 面板中，保留当前“前台直发”能力，同时新增“后端代发”模式，支持独立配置：

- 请求模式
- 请求方法
- 请求 URL
- 请求 Header
- 请求 JSON

并将这些配置保存到当前 HTTP 文档，便于下次打开继续使用。

## 当前现状

### 前端

- HTTP 面板当前发送逻辑位于：
  - `sport-ui/src/components/workspace-editor-pool.tsx`
- 现状是浏览器直接 `fetch`
- 当前默认行为：
  - 从请求 JSON 顶层读取 `method`
  - 从请求 JSON 顶层读取 `url`
  - 固定仅发送 `Content-Type: application/json`
- 现状问题：
  - Header 不能独立配置
  - 存在跨域、鉴权、内网访问受限问题
  - URL、Method、Body 混在一个 JSON 里，维护成本高

### 文档存储

- HTTP 文档内容保存在 `collect_doc.code`
- 当前页面编辑弹窗只维护：
  - `code`
  - `code_result`
  - `params`
  - `result`
  - `demo`
  - `important_list`
- 没有单独的请求模式/Header 字段

### 后端

- `collect` 内建已有 `http -> HttpService`
- 但当前不适合直接当调试代理暴露给 HTTP 面板，原因：
  - 返回格式偏通用服务，不是 HTTP 调试界面风格
  - 对非 200、非 JSON 响应的调试体验有限
- 当前仓库 `sport` 已支持注册 `outer` 类型服务，可新增 webshell 专用代理服务

## 目标方案

### 1. 前端保留两种请求模式

- `frontend`
  - 浏览器直接发请求
  - 适合无跨域限制、开放接口、前台联调
- `backend`
  - 浏览器请求本系统后端
  - 后端再代发目标请求
  - 适合跨域、内网、鉴权、第三方服务联调

默认值：

- `frontend`

### 2. Header 使用 JSON 对象录入

- 独立一个 Header JSON 编辑区
- 录入格式示例：

```json
{
  "Authorization": "Bearer xxx",
  "X-App-Code": "demo"
}
```

默认值：

```json
{}
```

### 3. 文档保存方式

不改数据库表结构，继续复用 `collect_doc.code`。

建议在 `code` JSON 顶层保留以下元字段：

- `method`
- `url`
- `_request_mode`
- `_request_headers`

业务请求体仍然保留在同一个 JSON 对象里。

示例：

```json
{
  "method": "post",
  "url": "/template_data/data",
  "_request_mode": "backend",
  "_request_headers": {
    "Authorization": "Bearer xxx"
  },
  "page": 1,
  "size": 20
}
```

发送请求时：

- UI 需要把元字段与真实 body 分离
- 前端直发或后端代发都不能把 `_request_mode`、`_request_headers` 当成真实业务字段发送出去

## 需要改造的模块

## 一、`sport-ui` 前端组件

主要文件：

- `sport-ui/src/components/workspace-editor-pool.tsx`

### 需要新增的状态字段

在 HTTP tab / console tab 状态中新增：

- `httpMode`
- `httpHeadersText`

建议默认值：

- `httpMode: "frontend"`
- `httpHeadersText: "{}"`

### 需要新增的能力

#### 1. 文档解析

新增一组 helper：

- 从 `doc.code` 中拆出：
  - `method`
  - `url`
  - `_request_mode`
  - `_request_headers`
  - 真实 body
- 加载文档时填充到 tab 状态

#### 2. 文档组装

保存文档或打开 console 时，需要把：

- method
- url
- request_mode
- request_headers
- body

重新合并回一个 JSON 文本

#### 3. HTTP 控制台 UI

在当前控制台顶部增加：

- 模式切换：前台 / 后端
- Method 选择
- URL 输入框
- Header JSON 编辑器
- 原请求 JSON 编辑器

建议布局：

- 顶部：模式 + Method + URL + 发送按钮
- 中间：Header 编辑器
- 下方：请求体 / 返回体分栏

#### 4. 前台直发逻辑

`frontend` 模式下：

- 解析 Header JSON
- GET 请求：
  - body 对象转 query string
- 非 GET 请求：
  - body 作为 JSON 发送
- 若 Header 未显式指定 `Content-Type`，默认补：
  - `application/json`

#### 5. 后端代发逻辑

`backend` 模式下：

- 不直接请求目标 URL
- 调用后端接口：
  - `webshell.http_proxy_request`
- 提交参数：
  - `request_method`
  - `request_url`
  - `request_header`
  - `request_data`

#### 6. 兼容旧文档

旧文档没有 `_request_mode`、`_request_headers` 时：

- 默认按 `frontend`
- Header 默认 `{}``

## 二、`sport` 低代码页面配置

主要文件：

- `collect/frontend/page_data/data/server/webshell_editor_pool_http_doc_dialog_fragment.json`
- `collect/frontend/page_data/data/server/webshell_editor_pool_http_fragment.json`
- 如有必要：
  - `collect/frontend/page_data/data/server/webshell_editor_pool_panel_fragment.json`
  - `collect/frontend/page_data/data/server/webshell_editor_pool_workspace_fragment.json`

### 需要改造的点

#### 1. HTTP 文档弹窗增加表单字段

新增：

- `request_mode`
- `request_method`
- `request_url`
- `request_headers`

保留：

- `code`
- `code_result`

#### 2. 编辑文档时的回填逻辑

当前 `config.doc_detail` 返回的 `doc.code` 需要在低代码层拆开：

- `request_mode`
- `request_method`
- `request_url`
- `request_headers`
- `code`

#### 3. 保存文档时的组装逻辑

保存时不要直接把 `code` 原样提交。

而是先把：

- `request_mode`
- `request_method`
- `request_url`
- `request_headers`
- `code`

合成一个最终 JSON 字符串，再作为 `doc.code` 保存。

#### 4. 新建默认值

建议：

- `request_mode = "frontend"`
- `request_method = "post"`
- `request_url = "/template_data/data"`
- `request_headers = "{}"`
- `code = "{}"`

## 三、`sport` 后端代理服务

### 目标

新增一个 webshell 专用服务：

- `webshell.http_proxy_request`

### 需要改造的文件

- `plugins/` 下新增模块服务实现
- `plugins/a_register.go`
- `collect/service_router.yml`
- `collect/webshell/service.yml`
- 建议新增：
  - `collect/webshell/http_proxy/index.yml`

### 服务职责

入参：

- `request_method`
- `request_url`
- `request_header`
- `request_data`

处理逻辑：

1. 校验 URL 非空
2. 解析 Header
3. 构造 `HttpConfig`
4. 复用 `collect` HTTP 请求处理逻辑发请求
5. 将结果转换成前端可直接显示的文本

### 结果建议

为了兼容当前控制台只显示“返回结果文本”的方式，建议先统一返回：

```json
{
  "response_text": "..."
}
```

后续如果要增强，可再补：

- `status_code`
- `response_headers`
- `response_body`

### 关键注意点

#### 1. Header 支持字符串或对象

`request_header` 可能来自：

- JSON 字符串
- 已解析对象

都要兼容。

#### 2. 非 JSON 响应不能直接失败

如果目标接口返回：

- 纯文本
- HTML
- 非 200

也应尽量把结果文本回传给前端，而不是直接中断。

#### 3. Content-Type 处理

若调用方未传 `Content-Type`：

- 对非 GET 请求默认补 `application/json`

## 代码落地顺序建议

### 第一步

只改 `sport-ui/src/components/workspace-editor-pool.tsx`

目标：

- 支持 `httpMode`
- 支持 `httpHeadersText`
- 保持现有页面不报错
- 前台模式先跑通

### 第二步

改低代码文档弹窗 JSON

目标：

- 新增表单字段
- 可回填/保存
- 新旧文档兼容

### 第三步

新增 `webshell.http_proxy_request`

目标：

- 后端模式可用
- 可回传文本结果

### 第四步

再做 UI 收敛

目标：

- HTTP 控制台布局更清晰
- Header 编辑体验更好
- 错误提示更明确

## 验证清单

### 前台模式

- 选择 `frontend`
- 填 URL + Header + 请求 JSON
- 成功发请求
- GET 时 query 拼接正确
- Header 正确带出

### 后端模式

- 选择 `backend`
- 通过 `webshell.http_proxy_request` 成功代发
- 结果写回返回体区域

### 持久化

- 新建 HTTP 文档
- 保存后重开
- 模式、Method、URL、Header、Body 都能回填

### 兼容性

- 旧 HTTP 文档打开不报错
- 不配置 Header 时仍可使用
- 不配置新模式时默认走 `frontend`

## 暂不执行

本文件仅作为后续改造说明，不代表已完成代码修改。
