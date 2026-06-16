# Webshell 双变量拆分实践（SSH + Editor）

本文沉淀 `webshell.json` 大文件拆分经验：如何拆成 2 个变量、每个变量负责什么渲染、最终效果是什么。

## 1. 目标

- 将单一超大页面拆成两个可独立维护的片段：
  - `webshell_ssh_fragment`：SSH 模块（登录、执行、历史/文件管理弹窗等）。
  - `webshell_editor_fragment`：工作空间 Drawer 整块（源码树、HTTP 树、右侧 panel、项目管理等）。
- 主文件保留骨架和装配关系，片段放同目录按服务注入。

## 2. 服务层拆分（核心）

在 `collect/frontend/page_data/index.yml` 中：

1. 在 `webshell` 服务里，先加两个 `service2field`，再 `file2datajson`：
   - `frontend.webshell_ssh_fragment -> webshell_ssh_fragment`
   - `frontend.webshell_editor_fragment -> webshell_editor_fragment`
2. 新增两个空服务（`module: empty`）：
   - `webshell_ssh_fragment`，`data_file: data/server/webshell_ssh_fragment.json`
   - `webshell_editor_fragment`，`data_file: data/server/webshell_editor_fragment.json`
3. 两个服务都使用：
   - `file2datajson -> save_field: data`
   - `param2result -> field: data`

这样低代码运行时会把两个 JSON 先读成变量，再给 `webshell.json` 用 `to_json` 引入。

## 3. 页面层如何渲染 2 个变量

### 3.1 主页面引用方式

`collect/frontend/page_data/data/server/webshell.json` 中只保留占位：

- `{{to_json .webshell_ssh_fragment.sider}}`
- `{{to_json .webshell_ssh_fragment.content_main.panel_group}}`
- `{{to_json .webshell_ssh_fragment.content_main.quick_start}}`
- `{{to_json .webshell_editor_fragment}}`
- `{{to_json .webshell_ssh_fragment.dialogs.history}}`
- `{{to_json .webshell_ssh_fragment.dialogs.file_manage}}`
- `{{to_json .webshell_ssh_fragment.dialogs.play}}`
- `{{to_json .webshell_ssh_fragment.float_btn}}`

说明：SSH 片段采用“对象分段”，方便在主页面多个位置挂载；Editor 片段是单一大节点（整个 Drawer）。

### 3.2 片段文件职责边界

- `webshell_ssh_fragment.json`
  - 包含左侧服务器树、SSH 终端主区域、浮动按钮、SSH 相关弹窗。
  - 可包含 `sider/content_main/float_btn/dialogs` 等分段字段。
- `webshell_editor_fragment.json`
  - 包含整个 workspace drawer 节点。
  - drawer 内部的源码目录、HTTP 目录、请求控制台、文件/HTTP 内容面板全部在这个文件维护。

## 4. 最终效果

- 业务效果不变：页面交互、接口、状态行为保持一致。
- 维护效果更好：
  - 主文件变短，阅读成本降低。
  - SSH 与 Editor 改动互不干扰。
  - 低代码里可以单独拉一个文件定位问题。
    - 例如只查 `service=frontend.webshell_editor_fragment` 对应 JSON。
    - 或只查 `service=frontend.webshell_ssh_fragment`。

## 5. 必做校验与常见坑

### 5.1 必做校验

- `jq empty` 校验 3 个 JSON：主文件 + 2 个片段。
- 校验 `index.yml` 可被 YAML 正常解析。
- 回归关键链路：SSH 登录执行、源码/HTTP tab 切换、Drawer 打开关闭。

### 5.2 常见坑

- 片段里残留 `{{to_json .webshell_xxx_fragment}}`，会造成递归引用。
- 拆分边界过细（只拆到深层 children），会导致“模块语义不完整”。
- `service2field` 顺序放错，导致变量未注入就被模板引用。

