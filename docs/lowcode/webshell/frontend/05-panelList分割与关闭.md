# PanelList 分割与关闭

## 分割规则

- 水平分割：`arr-push` 到 `workspacePanelList`
- 垂直分割：`method: ${row.parent.parent.addChildren(value)}`

## 渲染刷新关键

- 顶层 list 推荐：`"itemData": "${[...workspacePanelList]}"`
- 日志显示 arr-push 成功但 UI 不变时，优先检查这一条

## editor 实例隔离

- 避免同一路径复用 editor 实例：
  - `path: "${row.path+'?token='+row.token}"`

## 关闭 tab 策略

- 先 `row.removeByKey(key)`
- 若删的是 activeKey，回退到同组最后一个 tab
- 仅在子分组为空时清理分组，避免全量重建对象
