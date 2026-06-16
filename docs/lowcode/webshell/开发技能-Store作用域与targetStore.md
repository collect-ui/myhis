# Webshell 开发技能：Store 作用域与 targetStore

适用场景：`collect/frontend/page_data/data/server/webshell.json` 这类多层布局（页面 -> dialog -> panel -> 子组件）里，`update-store`、`reload-init-action` 出现“变量改了但界面没动”的问题。

## 核心概念

- `__this__`：当前动作所在 store（固定关键字）。
- `__parent__`：当前 store 的上级 store（固定关键字）。
- `targetStore`：`update-store` / `reload-init-action` 的目标 store。
- `activeStore`：业务变量，用于“从下往上传递一个可回调的 store 引用”。常见写法：`activeStore: "${__this__}"`。

## 方向规则（必须统一）

- 下层改上层：用 `targetStore: "__parent__"`。
- 上层改下层：先拿到下层传上来的 `activeStore`，再按 `activeStore` 定位执行。
- 同一状态（如弹框开关）必须始终由同一层维护，避免同名变量分散在多层。

## 推荐动作模板

### 1) 子层按钮打开上层弹框

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

说明：`activeStore` 一并上抛，方便后续“上层回刷当前 panel”。

### 2) 弹框关闭（写回拥有状态的那层）

```json
{
  "tag": "update-store",
  "value": {
    "dialogVisible": false
  }
}
```

说明：是否写 `targetStore` 取决于 `dialogVisible` 定义在哪层。若状态定义在本层，就不要写 `targetStore`。

### 3) 保存后刷新指定容器

- 列表刷新：在拥有查询条件的 store 执行 `reload-init-action`。
- 面板刷新：基于 `activeStore` 指向当前 panel 的 store，再执行 panel 相关 reload group。

## Webshell 目录管理的落地约定

- `workspaceFileDialogVisible/workspaceFileOp/workspaceFileForm` 由同一层 store 维护。
- 左侧树“新增目录”从 panel 子层触发，必须 `targetStore: "__parent__"`。
- 目录弹框关闭写回“弹框状态所在层”，不要机械使用 `__parent__`。
- `reload-init-action` 需要明确是列表刷新还是 panel 刷新，避免串刷新；刷新 panel 时应使用 `targetStore: ${activeStore}`。

## 实战动作链模板（目录新增/编辑确认）

下面模板对应 webshell 目录场景，重点是“列表刷新在管理层，树刷新在触发来源 panel 层”。

```json
[
  {
    "tag": "submit-form",
    "formName": "workspace-file-form"
  },
  {
    "tag": "ajax",
    "enable": "${workspaceFileOp==='add'}",
    "api": "post:/template_data/data?service=webshell.workspace_file_add",
    "appendFormFields": "workspace-file-form",
    "data": {
      "project_code": "${workspaceFileCurrentProject.project_code||workspaceCurrent}"
    }
  },
  {
    "tag": "ajax",
    "enable": "${workspaceFileOp==='edit'}",
    "api": "post:/template_data/data?service=webshell.workspace_file_update",
    "appendFormFields": "workspace-file-form",
    "data": {
      "project_code": "${workspaceFileCurrentProject.project_code||workspaceCurrent}"
    }
  },
  {
    "tag": "reload-init-action",
    "targetStore": "__parent__",
    "group": "reload-workspace-file-manage"
  },
  {
    "tag": "reload-init-action",
    "enable": "${!!activeStore}",
    "targetStore": "${activeStore}",
    "group": "reload-workspace-file-tree"
  },
  {
    "tag": "update-store",
    "value": {
      "workspaceFileDialogVisible": false
    }
  }
]
```

## 常见坑

- 只写 `workspaceFileDialogVisible=true`，没写 `targetStore`，结果写到了子层。
- 打开弹框用了 `__parent__`，关闭却误写其它层，导致“开得了关不掉”或反之。
- 依赖默认上下文刷新，导致刷新到了错误 panel。

## 调试建议

1. 先确认变量定义在哪一层。
2. 再看触发动作发生在哪一层。
3. 核对 `targetStore` 是否明确。
4. 需要反向操作 panel 时，检查是否已有 `activeStore: ${__this__}` 上抛。

## 表单实战约定（update-form 优先）

- 编辑场景优先用 `update-form`，不依赖 `initialValues`；这样后续字段增加时，不需要在多处默认值里重复维护。
- 推荐链路：`查询当前行 -> update-store 缓存 row -> 延迟 update-form(整行透传) -> submit-form -> appendFormFields 一次性保存`。
- 对于表单不展示但接口必填字段（如 `file_id`）：
  - 编辑时：`update-store` 保存 `currentRow`。
  - 保存时：在 `ajax.data` 单独补 `file_id: ${file_id || currentRow.file_id || ''}`。
- 若出现“首次编辑表单为空，第二次正常”，优先用“延迟 update-form”处理时序，不强行切到 `initialValues`。

## 本次经验补充（工具栏与搜索）

- 树面板的“新增+搜索”尽量放 `searchToolBar`，不要放 `topRight`，在可拖拽 panel 下更稳定，不易被裁切。
- 搜索输入统一交互：`allowClear + addonAfterIcon: SearchOutlined`，提升一致性；服务器列表和工作区目录树保持同一视觉语义。
- 紧凑布局建议：按钮只保留图标，输入框使用 `className: flex1` 自适应，避免固定宽度导致遮挡。
