# Webshell Drawer 复用改造测试计划

## 1. 变更目标
- 页面：`#/collect-ui/framework/webshell`
- 范围：仅改 `drawer` 内部工作空间实现。
- 目标：`webshell` 的 drawer 内部不再维护旧版内联逻辑，改为整文件引入 `webshell_editor_pool_route`。
- 非目标：`webshell` 主体（SSH、历史、非 drawer 区域）不做功能改造。

## 2. 改造前备份

### 2.1 备份策略
- 采用时间戳目录备份，避免覆盖历史备份。
- 备份目录：
  - `test/lowcode-page/results/latest/backups/webshell-drawer-reuse-20260505_202506/`

### 2.2 已备份文件
- `collect/frontend/page_data/data/server/webshell_editor_fragment.json`
- `collect/frontend/page_data/data/server/webshell.json`
- `collect/frontend/page_data/data/server/webshell_editor_pool_route.json`

### 2.3 备份校验
- `backup-manifest.sha256`：文件哈希清单
- `backup-manifest.txt`：文件大小与备份元信息

## 3. 环境准备与启动校验

在仓库根目录执行：

```bash
./linux-shutdown
ss -ltnp | rg ':8015' || true
./linux-startup
ss -ltnp | rg ':8015'
```

预期：最后一条命令可看到 `8015` 监听。

## 4. 手工测试清单（必须全过）

### 4.1 页面与抽屉基础
1. 打开 `http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell`。
2. 点击“工作空间”打开 drawer。
3. 关闭 drawer 后再次打开，确认状态正常。
4. 切换宽度档位（窄/中/宽），确认 UI 生效。
5. 刷新页面后再次打开 drawer，确认宽度记忆正常。

### 4.2 Drawer 内部复用完整性
1. 确认 drawer 内加载的是工作台结构（源码目录 / HTTP目录 / 右侧工作台）。
2. 切换项目后，左树与右侧内容同步刷新。
3. 打开项目管理弹窗，确认可正常打开/关闭。
4. 打开文件管理相关弹窗，确认可正常打开/关闭。
5. 打开 HTTP 分组/接口弹窗，确认可正常打开/关闭。

### 4.3 源码目录链路
1. 源码目录树可加载。
2. 搜索关键字（至少 2 个字符）可过滤结果。
3. 清空关键字后恢复树。
4. 文件/目录新增、编辑、删除动作均可执行。
5. 文件可打开到右侧工作台，标签可切换与关闭。
6. 分屏/分栏操作可用，无异常报错。

### 4.4 HTTP 目录链路
1. HTTP 树加载正常。
2. 选择接口节点可加载详情与控制台。
3. 右键菜单操作可用：新增接口、编辑接口、删除接口。
4. 分组相关操作可用：新增分组、编辑分组、删除分组。
5. 文档保存后，树与详情刷新正常。

### 4.5 HTTP 控制台链路
1. `frontend` 模式可发送请求并返回结果。
2. `backend` 模式可发送请求并返回结果。
3. method/url/header/body 修改后可保存到文档。
4. 刷新页面后再打开同一接口，保存内容可回显。
5. 测试数据 CRUD（查询/新增/更新/删除）能力正常。

### 4.6 回归验证（webshell 主体）
1. SSH 区域基础功能可用。
2. 历史记录与回放入口可正常打开。
3. 非 drawer 区域无明显 UI/交互回归。
4. 页面无阻断级错误（白屏、JS 崩溃）。

## 5. 自动化测试清单（建议全量执行）

## 5.1 环境变量

```bash
export WEBSHELL_EDITOR_POOL_PAGE_URL='http://192.168.232.130:8015/collect-ui#/collect-ui/framework/webshell'
export WEBSHELL_EDITOR_POOL_BASE_URL='http://192.168.232.130:8015'
```

## 5.2 执行脚本

在仓库根目录执行：

```bash
node test/lowcode-page/scripts/frontend/webshell_editor_pool_http_full_flow_check.js
node test/lowcode-page/scripts/frontend/webshell_editor_pool_http_mode_flow_check.js
node test/lowcode-page/scripts/frontend/webshell_editor_pool_http_project_isolation_check.js
node test/lowcode-page/scripts/frontend/webshell_editor_pool_http_doc_dialog_ui_check.js
node test/lowcode-page/scripts/frontend/webshell_editor_pool_console_check.js
node test/lowcode-page/scripts/frontend/webshell_editor_pool_console_request_matrix_check.js
node test/lowcode-page/scripts/frontend/webshell_editor_pool_console_save_check.js
node test/lowcode-page/scripts/frontend/webshell_editor_pool_console_save_to_doc_check.js
node test/lowcode-page/scripts/frontend/webshell_editor_pool_console_create_doc_closure_check.js
node test/lowcode-page/scripts/frontend/webshell_editor_pool_console_store_crud_check.js
node test/lowcode-page/scripts/frontend/webshell_editor_pool_content_search_check.js
node test/lowcode-page/scripts/frontend/webshell_editor_pool_http_test2_login_chain_check.js
node test/lowcode-page/scripts/frontend/webshell_http_proxy_page_check.js
```

## 5.3 结果目录
- 默认结果目录：`test/lowcode-page/results/latest/http-proxy-validation/`
- 关键检查项：
  - 各脚本返回码为 0
  - 结果 JSON 标记成功
  - 截图与日志输出完整

## 6. 验收标准
- 阻断/严重缺陷：0
- 自动化脚本：全通过（或有明确豁免说明）
- Drawer 复用目标达成：
  - `webshell_editor_fragment.json` 不再维护旧版内联工作空间逻辑
  - 仅通过整文件引入 `webshell_editor_pool_route` 复用
- `webshell` 主体功能无回归

## 7. 缺陷记录模板

```md
- 缺陷编号：
- 严重级别：
- 发现时间：
- 复现路径：
- 复现步骤：
- 预期结果：
- 实际结果：
- 截图/日志：
- 关联脚本：
- 处理状态：
```
