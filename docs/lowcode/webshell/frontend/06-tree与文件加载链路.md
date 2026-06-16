# Tree 与文件加载链路

## 标准流程

1. tree 选中文件（非目录）
2. update-store 写入：
   - `workspaceActivePath`
   - `workspaceActiveName`
   - `workspaceToken`
3. 计算目标分组位置（LastX/LastY）
4. 若文件已打开，切换 activeKey
5. 否则 push 新 tab，并 initialize

## 易错点

- `treeData` 必须是数组，否则会报 `newProps.treeData is not iterable`
- 依赖 `row.*` 的逻辑放在 action，不要塞进不支持表达式的初始化路径
