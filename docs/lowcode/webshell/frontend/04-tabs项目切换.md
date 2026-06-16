# Tabs 项目切换

## 推荐模式

- 点击项目时：
  - `workspaceLoadedMap[project_id] = true`
  - `workspaceCurrent = project_id`
- tabs 渲染：
  - `itemData: "${workspaceOptions.filter(item=>workspaceLoadedMap[item.path || item.value])}"`

## 标识统一

- 优先使用 `row.path` 作为 `project_id`
- 若无 path，统一 fallback 到 `row.value`
- 避免 path/value 混用造成取值失败

## 注意

- 外层 tabs 不建议滥用 `withHistory`，可能影响顶层 panelList 重绘
- 需要隐藏标签头可用 `hideTabBar: true`
